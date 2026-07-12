package service

import (
	"errors"
	"fmt"
	"strings"

	"storecore/internal/dto"
	"storecore/internal/model"

	"gorm.io/gorm"
)

func (s *SalesService) Update(id uint64, in *dto.StoreSalesOrderDTO) (*model.StoreSalesOrder, error) {
	r := s.repos.Sales.ForTenant(s.tenantID)
	order, err := r.GetByID(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if order.Status != "draft" && order.Status != "preview" {
		return nil, ErrInvalidStatus
	}
	if len(in.Items) == 0 {
		return nil, ErrBadRequest
	}
	items, originalTotal, payableTotal := buildSalesItems(in.Items)
	serviceItems, svcOrig, svcPay := buildSalesServiceItems(in.ServiceItems)
	originalTotal = roundMoney(originalTotal + svcOrig)
	payableTotal = roundMoney(payableTotal + svcPay)

	if err := s.applySalesDTO(order, in); err != nil {
		return nil, err
	}
	if order.FulfillmentType == "delivery" || order.FulfillmentType == "express" {
		if strings.TrimSpace(order.ShippingAddress) == "" {
			return nil, ErrBadRequest
		}
	}
	order.OriginalAmount = originalTotal
	order.DiscountAmount = roundMoney(originalTotal - payableTotal)
	order.TotalAmount = payableTotal
	if in.IsPreview {
		order.Status = "preview"
	} else if order.Status == "preview" {
		order.Status = "draft"
	}
	s.attachReceipt(order, items, serviceItems, order.Status == "preview")
	if err := r.ReplaceItems(order.ID, items, serviceItems); err != nil {
		return nil, err
	}
	if err := r.Save(order); err != nil {
		return nil, err
	}
	order.Items = items
	order.ServiceItems = serviceItems
	return order, nil
}

func (s *SalesService) transition(id uint64, from []string, to string, mutate func(*model.StoreSalesOrder) error) (*model.StoreSalesOrder, error) {
	r := s.repos.Sales.ForTenant(s.tenantID)
	order, err := r.GetByID(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	ok := false
	for _, st := range from {
		if order.Status == st {
			ok = true
			break
		}
	}
	if !ok {
		return nil, ErrInvalidStatus
	}
	if mutate != nil {
		if err := mutate(order); err != nil {
			return nil, err
		}
	}
	order.Status = to
	if err := r.Save(order); err != nil {
		return nil, err
	}
	return order, nil
}

func (s *SalesService) Confirm(id uint64) (*model.StoreSalesOrder, error) {
	r := s.repos.Sales.ForTenant(s.tenantID)
	order, err := r.GetByID(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if order.Status != "draft" && order.Status != "preview" {
		return nil, ErrInvalidStatus
	}
	switch order.FulfillmentType {
	case "pickup", "install":
		if order.AppointmentAt == nil {
			return nil, ErrBadRequest
		}
	case "delivery", "express":
		if strings.TrimSpace(order.ShippingAddress) == "" {
			return nil, ErrBadRequest
		}
	}
	if order.FulfillmentType == "install" {
		if len(order.ServiceItems) == 0 {
			return nil, ErrBadRequest
		}
		if order.ServiceOrderID == 0 {
			hasCatalog := false
			for _, it := range order.ServiceItems {
				if it.ServiceItemID > 0 {
					hasCatalog = true
					break
				}
			}
			if hasCatalog {
				if err := s.createLinkedServiceOrder(order); err != nil {
					return nil, err
				}
			} else {
				order.ServiceStatus = "pending"
			}
		}
	}
	initSubStatusesOnConfirm(order)
	order.Status = "confirmed"
	s.attachReceipt(order, order.Items, order.ServiceItems, false)
	if err := r.Save(order); err != nil {
		return nil, err
	}
	return order, nil
}

func (s *SalesService) createLinkedServiceOrder(order *model.StoreSalesOrder) error {
	lines := make([]dto.ServiceOrderLineDTO, 0, len(order.ServiceItems))
	manualNames := make([]string, 0)
	for _, it := range order.ServiceItems {
		if it.ServiceItemID > 0 {
			lines = append(lines, dto.ServiceOrderLineDTO{
				ServiceItemID: it.ServiceItemID,
				Quantity:      it.Quantity,
			})
		} else if strings.TrimSpace(it.ServiceName) != "" {
			manualNames = append(manualNames, it.ServiceName)
		}
	}
	if len(lines) == 0 {
		return nil
	}
	var appointment *string
	if order.AppointmentAt != nil {
		v := order.AppointmentAt.Format("2006-01-02T15:04:05")
		appointment = &v
	}
	remark := strings.TrimSpace(order.Remark)
	if remark != "" {
		remark += "；"
	}
	remark += fmt.Sprintf("来自销售单 %s（到店安装）", order.OrderNo)
	if len(manualNames) > 0 {
		remark += "；手动服务：" + strings.Join(manualNames, "、")
	}
	in := &dto.ServiceOrderDTO{
		StoreID:       order.StoreID,
		OrderMode:     "appointment",
		CustomerName:  order.CustomerName,
		CustomerPhone: order.CustomerPhone,
		AppointmentAt: appointment,
		Remark:        remark,
		Items:         lines,
	}
	svc := NewServiceOrderService(s.repos).ForTenant(s.tenantID)
	so, err := svc.Create(in, order.CreatedBy)
	if err != nil {
		return err
	}
	so.SalesOrderID = order.ID
	so.SalesOrderNo = order.OrderNo
	if err := s.repos.Service.ForTenant(s.tenantID).Update(so, nil); err != nil {
		return err
	}
	order.ServiceOrderID = so.ID
	order.ServiceOrderNo = so.OrderNo
	order.ServiceStatus = so.Status
	return nil
}

func (s *SalesService) Cancel(id uint64) (*model.StoreSalesOrder, error) {
	return s.transition(id, []string{"draft", "preview", "confirmed", "ready", "shipping"}, "cancelled", func(order *model.StoreSalesOrder) error {
		if order.FulfillStatus != "none" && order.FulfillStatus != "picked_up" && order.FulfillStatus != "delivered" && order.FulfillStatus != "received" {
			order.FulfillStatus = "none"
		}
		return nil
	})
}

// Delete 物理删除：仅草稿 / 预结算 / 已取消可删
func (s *SalesService) Delete(id uint64) error {
	r := s.repos.Sales.ForTenant(s.tenantID)
	order, err := r.GetByID(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	switch order.Status {
	case "draft", "preview", "cancelled":
		// ok
	default:
		return ErrInvalidStatus
	}
	if err := r.Delete(id); errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound
	} else if err != nil {
		return err
	}
	return nil
}

func (s *SalesService) MarkReady(id uint64) (*model.StoreSalesOrder, error) {
	return s.transition(id, []string{"confirmed"}, "ready", func(order *model.StoreSalesOrder) error {
		if order.FulfillmentType != "pickup" && order.FulfillmentType != "install" {
			return ErrInvalidStatus
		}
		order.FulfillStatus = "awaiting_pickup"
		return nil
	})
}

func (s *SalesService) Ship(id uint64) (*model.StoreSalesOrder, error) {
	return s.transition(id, []string{"confirmed", "ready"}, "shipping", func(order *model.StoreSalesOrder) error {
		switch order.FulfillmentType {
		case "delivery":
			order.FulfillStatus = "delivering"
		case "express":
			order.FulfillStatus = "expressed"
		default:
			return ErrInvalidStatus
		}
		return nil
	})
}

// ScheduleExpress 预约快递（后续对接发货中心）；先记录状态与预约时间。
func (s *SalesService) ScheduleExpress(id uint64, scheduledAt *string, company string) (*model.StoreSalesOrder, error) {
	r := s.repos.Sales.ForTenant(s.tenantID)
	order, err := r.GetByID(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if order.FulfillmentType != "express" {
		return nil, ErrBadRequest
	}
	if order.Status != "confirmed" && order.Status != "ready" {
		return nil, ErrInvalidStatus
	}
	t, err := parseOptionalTime(scheduledAt)
	if err != nil {
		return nil, err
	}
	if t != nil {
		order.ExpressScheduledAt = t
	}
	if strings.TrimSpace(company) != "" {
		order.ExpressCompany = strings.TrimSpace(company)
	}
	order.FulfillStatus = "expressed"
	// 发货中心对接预留：此处仅更新状态
	if err := r.Save(order); err != nil {
		return nil, err
	}
	return order, nil
}

func (s *SalesService) Complete(id uint64) (*model.StoreSalesOrder, error) {
	r := s.repos.Sales.ForTenant(s.tenantID)
	order, err := r.GetByID(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	switch order.FulfillmentType {
	case "pickup", "install":
		if order.Status != "ready" {
			return nil, ErrInvalidStatus
		}
		order.FulfillStatus = "picked_up"
	case "delivery", "express":
		if order.Status != "shipping" && order.Status != "confirmed" {
			return nil, ErrInvalidStatus
		}
		if order.FulfillmentType == "delivery" {
			order.FulfillStatus = "delivered"
		} else {
			order.FulfillStatus = "received"
		}
	default:
		return nil, ErrInvalidStatus
	}
	order.Status = "completed"
	if err := r.Save(order); err != nil {
		return nil, err
	}
	inv := s.repos.Inventory.ForTenant(s.tenantID)
	for _, line := range order.Items {
		_ = inv.AddQuantity(order.StoreID, line.SkuID, line.SkuCode, line.ProductName, line.SpecLabel, line.Pic, -line.Quantity)
	}
	return order, nil
}

// RefreshReceipt 重新生成销售单预览/小票 HTML（给顾客看）
func (s *SalesService) RefreshReceipt(id uint64, preview bool) (*model.StoreSalesOrder, error) {
	r := s.repos.Sales.ForTenant(s.tenantID)
	order, err := r.GetByID(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	s.attachReceipt(order, order.Items, order.ServiceItems, preview || order.Status == "preview")
	if err := r.Save(order); err != nil {
		return nil, err
	}
	return order, nil
}

// SyncServiceStatus 服务工单状态变更时回写销售单服务状态
func (s *SalesService) SyncServiceStatus(salesOrderID uint64, serviceStatus string) error {
	if salesOrderID == 0 {
		return nil
	}
	r := s.repos.Sales.ForTenant(s.tenantID)
	order, err := r.GetByID(salesOrderID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	order.ServiceStatus = serviceStatus
	return r.Save(order)
}

// MarkPurchaseOrdered 从销售单生成采购单后回写
func (s *SalesService) MarkPurchaseOrdered(salesOrderID, purchaseOrderID uint64) error {
	if salesOrderID == 0 {
		return nil
	}
	r := s.repos.Sales.ForTenant(s.tenantID)
	order, err := r.GetByID(salesOrderID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	order.PurchaseStatus = "ordered"
	order.PurchaseOrderID = purchaseOrderID
	return r.Save(order)
}

// MarkPurchaseReceived 采购到货后回写
func (s *SalesService) MarkPurchaseReceived(salesOrderID uint64) error {
	if salesOrderID == 0 {
		return nil
	}
	r := s.repos.Sales.ForTenant(s.tenantID)
	order, err := r.GetByID(salesOrderID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	order.PurchaseStatus = "received"
	return r.Save(order)
}
