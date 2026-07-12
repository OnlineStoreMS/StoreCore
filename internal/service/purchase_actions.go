package service

import (
	"errors"

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
	return s.transition(id, []string{"draft"}, "submitted")
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
		_ = NewSalesService(s.repos).ForTenant(s.tenantID).MarkPurchaseReceived(order.RefSalesID)
	}
	return order, nil
}

func (s *PurchaseService) CreateFromSales(salesID uint64, in *dto.StorePurchaseOrderDTO, userID uint64) (*model.StorePurchaseOrder, error) {
	so, err := s.repos.Sales.ForTenant(s.tenantID).GetByID(salesID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if !so.NeedProcurement {
		return nil, ErrBadRequest
	}
	if len(in.Items) == 0 {
		items := make([]dto.OrderLineDTO, 0, len(so.Items))
		for _, line := range so.Items {
			items = append(items, dto.OrderLineDTO{
				SkuID: line.SkuID, ProductName: line.ProductName, SkuCode: line.SkuCode,
				SpecLabel: line.SpecLabel, Quantity: line.Quantity, UnitPrice: line.UnitPrice,
			})
		}
		in.Items = items
	}
	in.StoreID = so.StoreID
	in.RefSalesID = salesID
	in.PurchaseType = "sales_driven"
	po, err := s.Create(in, userID)
	if err != nil {
		return nil, err
	}
	_ = NewSalesService(s.repos).ForTenant(s.tenantID).MarkPurchaseOrdered(salesID, po.ID)
	return po, nil
}
