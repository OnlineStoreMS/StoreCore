package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"storecore/internal/dto"
	"storecore/internal/model"

	"gorm.io/gorm"
)

func (s *PurchaseService) Get(id uint64) (*model.StorePurchaseOrder, error) {
	item, err := s.repos.Purchase.ForTenant(s.tenantID).GetByID(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return item, err
}

func (s *PurchaseService) transition(id uint64, from []string, to string) (*model.StorePurchaseOrder, error) {
	r := s.repos.Purchase.ForTenant(s.tenantID)
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
	order.Status = to
	if err := r.Save(order); err != nil {
		return nil, err
	}
	return order, nil
}

func (s *PurchaseService) Submit(id uint64) (*model.StorePurchaseOrder, error) {
	order, err := s.transition(id, []string{"draft"}, "submitted")
	if err != nil {
		return nil, err
	}
	if order.RefSalesID > 0 {
		_ = NewSalesService(s.repos, nil).ForTenant(s.tenantID).MarkPurchaseOrdered(order.RefSalesID, order.ID)
	}
	return order, nil
}

func (s *PurchaseService) Cancel(id uint64) (*model.StorePurchaseOrder, error) {
	return s.transition(id, []string{"draft", "submitted"}, "cancelled")
}

func (s *PurchaseService) Receive(id uint64) (*model.StorePurchaseOrder, error) {
	r := s.repos.Purchase.ForTenant(s.tenantID)
	order, err := r.GetByID(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if order.Status != "submitted" {
		return nil, ErrInvalidStatus
	}
	inv := s.repos.Inventory.ForTenant(s.tenantID)
	for _, line := range order.Items {
		if err := inv.AddQuantity(order.StoreID, line.SkuID, line.SkuCode, line.ProductName, "", "", line.Quantity); err != nil {
			return nil, err
		}
	}
	order.Status = "received"
	if err := r.Save(order); err != nil {
		return nil, err
	}
	if order.RefSalesID > 0 {
		_ = NewSalesService(s.repos, nil).ForTenant(s.tenantID).MarkPurchaseReceived(order.RefSalesID)
	}
	return order, nil
}

func (s *PurchaseService) CreateFromSales(salesID uint64, in *dto.StorePurchaseOrderDTO, userID uint64) (*model.StorePurchaseOrder, error) {
	return s.CreateFromSalesWithContext(context.Background(), salesID, in, userID)
}

func (s *PurchaseService) CreateFromSalesWithContext(ctx context.Context, salesID uint64, in *dto.StorePurchaseOrderDTO, userID uint64) (*model.StorePurchaseOrder, error) {
	so, err := s.repos.Sales.ForTenant(s.tenantID).GetByID(salesID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if so.PayStatus != "paid" {
		return nil, fmt.Errorf("%w：请先完成销售单付款后再生成采购单", ErrBadRequest)
	}
	if so.PurchaseOrderID > 0 {
		existing, err := s.Get(so.PurchaseOrderID)
		if err == nil {
			return existing, nil
		}
	}
	salesSvc := NewSalesService(s.repos, s.pc).ForTenant(s.tenantID).WithAuth(s.authToken)
	plan, err := salesSvc.buildSalesStockPlan(ctx, so.StoreID, so.Items)
	if err != nil {
		return nil, err
	}
	if !plan.NeedProcurement {
		return nil, fmt.Errorf("%w：当前订单无需采购（门店/仓库库存可覆盖）", ErrBadRequest)
	}
	if in == nil {
		in = &dto.StorePurchaseOrderDTO{}
	}
	if len(in.Items) == 0 {
		items := make([]dto.OrderLineDTO, 0)
		for _, p := range plan.Lines {
			if p.PurchaseQty <= 0 {
				continue
			}
			items = append(items, dto.OrderLineDTO{
				SkuID: p.Item.SkuID, ProductName: p.Item.ProductName, SkuCode: p.Item.SkuCode,
				SpecLabel: p.Item.SpecLabel, Pic: p.Item.Pic, Quantity: p.PurchaseQty, UnitPrice: p.Item.UnitPrice,
			})
		}
		in.Items = items
	}
	in.StoreID = so.StoreID
	in.RefSalesID = salesID
	in.PurchaseType = "sales_driven"
	if strings.TrimSpace(in.Remark) == "" {
		in.Remark = "来自销售单 " + so.OrderNo + "（采购草稿）"
	}
	po, err := s.Create(in, userID)
	if err != nil {
		return nil, err
	}
	_ = salesSvc.LinkPurchaseDraft(salesID, po.ID)
	return po, nil
}
