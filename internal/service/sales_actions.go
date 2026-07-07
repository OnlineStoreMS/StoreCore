package service

import (
	"errors"

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
	if order.Status != "draft" {
		return nil, ErrInvalidStatus
	}
	if len(in.Items) == 0 {
		return nil, ErrBadRequest
	}
	items, total := buildSalesItems(in.Items)
	ft := in.FulfillmentType
	if ft == "" {
		ft = "pickup"
	}
	order.FulfillmentType = ft
	order.CustomerName = in.CustomerName
	order.CustomerPhone = in.CustomerPhone
	order.ShippingAddress = in.ShippingAddress
	order.NeedProcurement = in.NeedProcurement
	order.Remark = in.Remark
	order.TotalAmount = total
	if err := r.ReplaceItems(order.ID, items); err != nil {
		return nil, err
	}
	if err := r.Save(order); err != nil {
		return nil, err
	}
	order.Items = items
	return order, nil
}

func (s *SalesService) transition(id uint64, from []string, to string) (*model.StoreSalesOrder, error) {
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
	order.Status = to
	if err := r.Save(order); err != nil {
		return nil, err
	}
	return order, nil
}

func (s *SalesService) Confirm(id uint64) (*model.StoreSalesOrder, error) {
	return s.transition(id, []string{"draft"}, "confirmed")
}

func (s *SalesService) Cancel(id uint64) (*model.StoreSalesOrder, error) {
	return s.transition(id, []string{"draft", "confirmed", "ready", "shipping"}, "cancelled")
}

func (s *SalesService) MarkReady(id uint64) (*model.StoreSalesOrder, error) {
	return s.transition(id, []string{"confirmed"}, "ready")
}

func (s *SalesService) Ship(id uint64) (*model.StoreSalesOrder, error) {
	return s.transition(id, []string{"confirmed"}, "shipping")
}

func (s *SalesService) Complete(id uint64) (*model.StoreSalesOrder, error) {
	order, err := s.transition(id, []string{"ready", "shipping"}, "completed")
	if err != nil {
		return nil, err
	}
	// 提货/发货完成时扣减门店库存（有货则扣）
	inv := s.repos.Inventory.ForTenant(s.tenantID)
	for _, line := range order.Items {
		_ = inv.AddQuantity(order.StoreID, line.SkuID, line.SkuCode, line.ProductName, line.SpecLabel, -line.Quantity)
	}
	return order, nil
}
