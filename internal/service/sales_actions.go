package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

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
	// 草稿允许暂缺收货地址；确认订单时再强制校验
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
	order.Items = nil
	order.ServiceItems = nil
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
	return s.ConfirmWithContext(context.Background(), id)
}

func (s *SalesService) ConfirmWithContext(ctx context.Context, id uint64) (*model.StoreSalesOrder, error) {
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
			return nil, fmt.Errorf("%w：请填写预约时间", ErrBadRequest)
		}
	case "delivery", "express":
		if strings.TrimSpace(order.ShippingAddress) == "" {
			return nil, fmt.Errorf("%w：请填写收货地址", ErrBadRequest)
		}
	}
	if order.FulfillmentType == "install" {
		if len(order.ServiceItems) == 0 {
			return nil, fmt.Errorf("%w：到店安装请先选择服务项目后再确认", ErrBadRequest)
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

	plan, err := s.buildSalesStockPlan(ctx, order.StoreID, order.Items)
	if err != nil {
		return nil, err
	}
	order.NeedProcurement = plan.NeedProcurement
	if plan.NeedProcurement {
		if order.PurchaseStatus == "none" || order.PurchaseStatus == "" {
			order.PurchaseStatus = "pending"
		}
	} else {
		order.PurchaseStatus = "none"
		order.PurchaseOrderID = 0
	}
	if plan.NeedTransfer && order.StockTransferOrderID == 0 {
		st, err := s.createTransferFromPlan(order, plan)
		if err != nil {
			return nil, err
		}
		if st != nil {
			order.StockTransferOrderID = st.ID
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

// MarkPaid 销售单结算付款；付款完成后若需采购则自动生成采购单草稿并关联。
func (s *SalesService) MarkPaid(id uint64) (*model.StoreSalesOrder, error) {
	return s.MarkPaidWithContext(context.Background(), id, 0, nil)
}

func (s *SalesService) MarkPaidWithContext(ctx context.Context, id uint64, userID uint64, in *dto.SalesMarkPaidDTO) (*model.StoreSalesOrder, error) {
	r := s.repos.Sales.ForTenant(s.tenantID)
	order, err := r.GetByID(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	switch order.Status {
	case "confirmed", "ready", "shipping":
		// ok
	default:
		return nil, ErrInvalidStatus
	}
	if order.PayStatus == "paid" {
		return order, nil
	}

	method := "transfer"
	proof := ""
	if in != nil {
		m, err := normalizeOfflinePayMethod(in.PaymentMethod)
		if err != nil {
			return nil, err
		}
		method = m
		proof = strings.TrimSpace(in.PaymentProofURL)
		if method == "transfer" && proof == "" {
			return nil, fmt.Errorf("%w：转账收款请上传付款截图", ErrBadRequest)
		}
	}

	now := time.Now()
	order.PayStatus = "paid"
	order.PaymentMethod = method
	order.PaymentProofURL = proof
	if in != nil {
		if t := parseOptionalPaidAt(in.PaidAt); t != nil {
			order.PaidAt = t
		} else {
			order.PaidAt = &now
		}
	} else {
		order.PaidAt = &now
	}

	if order.NeedProcurement && order.PurchaseOrderID == 0 {
		plan, err := s.buildSalesStockPlan(ctx, order.StoreID, order.Items)
		if err != nil {
			return nil, err
		}
		if !plan.NeedProcurement {
			order.NeedProcurement = false
			order.PurchaseStatus = "none"
		} else {
			uid := userID
			if uid == 0 {
				uid = order.CreatedBy
			}
			po, err := s.createPurchaseDraftFromPlan(order, plan, uid)
			if err != nil {
				return nil, err
			}
			if po != nil {
				order.PurchaseOrderID = po.ID
				order.PurchaseStatus = "pending"
			}
		}
	}

	s.attachReceipt(order, order.Items, order.ServiceItems, false)
	if err := r.Save(order); err != nil {
		return nil, err
	}
	if order.ServiceOrderID > 0 {
		_ = NewServiceOrderService(s.repos).ForTenant(s.tenantID).MarkPaidFromSalesOrder(order.ServiceOrderID, order)
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
	salesSvcTotal := 0.0
	for _, it := range order.ServiceItems {
		salesSvcTotal = roundMoney(salesSvcTotal + it.TotalAmount)
	}
	// 关联销售单金额以销售明细为准；零元或销售已付款则无需收银台
	so.EstimatedAmount = salesSvcTotal
	if order.PayStatus == "paid" || salesSvcTotal <= 0 {
		so.PayStatus = "paid"
	}
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

// Delete 物理删除：仅草稿 / 预结算 / 已取消可删；级联删除关联服务工单及其收银订单。
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
	if order.ServiceOrderID > 0 {
		svc := NewServiceOrderService(s.repos).ForTenant(s.tenantID)
		if err := svc.deleteWithCascade(order.ServiceOrderID, false); err != nil && !errors.Is(err, ErrNotFound) {
			return err
		}
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
		// 履约就绪：进入待提货（与付款状态无关）
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
	if err := s.validateReadyForHandover(order); err != nil {
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

// validateReadyForHandover 提货/交付前：有关联服务工单须已完成；需采购须已到货。
func (s *SalesService) validateReadyForHandover(order *model.StoreSalesOrder) error {
	if order.ServiceOrderID > 0 || order.FulfillmentType == "install" {
		svcStatus := order.ServiceStatus
		if order.ServiceOrderID > 0 {
			so, err := s.repos.Service.ForTenant(s.tenantID).GetByID(order.ServiceOrderID)
			if err == nil {
				svcStatus = so.Status
				order.ServiceStatus = so.Status
			}
		}
		if svcStatus != "completed" {
			return fmt.Errorf("%w：服务工单未完成，无法标记已提货", ErrBadRequest)
		}
	}
	if order.NeedProcurement || order.PurchaseOrderID > 0 || (order.PurchaseStatus != "" && order.PurchaseStatus != "none") {
		purchaseStatus := order.PurchaseStatus
		if order.PurchaseOrderID > 0 {
			po, err := s.repos.Purchase.ForTenant(s.tenantID).GetByID(order.PurchaseOrderID)
			if err == nil {
				// 采购单 received → 销售侧 purchaseStatus 应为 received
				if po.Status == "received" {
					purchaseStatus = "received"
					order.PurchaseStatus = "received"
				} else {
					purchaseStatus = po.Status
				}
			}
		}
		if purchaseStatus != "received" {
			return fmt.Errorf("%w：采购订单未到货，无法标记已提货", ErrBadRequest)
		}
	}
	return nil
}

// RefreshReceipt 按订单状态重新生成销售单 HTML（草稿/预结算→预结算单；已确认及之后→正式销售单）
func (s *SalesService) RefreshReceipt(id uint64, _ bool) (*model.StoreSalesOrder, error) {
	r := s.repos.Sales.ForTenant(s.tenantID)
	order, err := r.GetByID(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	s.attachReceipt(order, order.Items, order.ServiceItems, false)
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

// LinkPurchaseDraft 关联采购草稿（付款后自动生成）
func (s *SalesService) LinkPurchaseDraft(salesOrderID, purchaseOrderID uint64) error {
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
	order.PurchaseOrderID = purchaseOrderID
	order.NeedProcurement = true
	if order.PurchaseStatus == "none" || order.PurchaseStatus == "" {
		order.PurchaseStatus = "pending"
	}
	return r.Save(order)
}

// MarkPurchaseOrdered 采购单提交后回写
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
