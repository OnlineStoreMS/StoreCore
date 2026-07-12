package service

import (
	"context"
	"strconv"
	"strings"
	"time"

	"storecore/internal/dto"
	"storecore/internal/model"
)

type salesStockLinePlan struct {
	Item         model.StoreSalesOrderItem
	StoreQty     int
	WarehouseQty int
	TransferQty  int
	PurchaseQty  int
}

type salesStockPlan struct {
	Lines           []salesStockLinePlan
	NeedProcurement bool
	NeedTransfer    bool
}

func (s *SalesService) buildSalesStockPlan(ctx context.Context, storeID uint64, items []model.StoreSalesOrderItem) (*salesStockPlan, error) {
	plan := &salesStockPlan{Lines: make([]salesStockLinePlan, 0, len(items))}
	if len(items) == 0 {
		return plan, nil
	}
	skuIDs := make([]uint64, 0, len(items))
	for _, it := range items {
		if it.SkuID > 0 {
			skuIDs = append(skuIDs, it.SkuID)
		}
	}
	storeQty, err := s.repos.Inventory.ForTenant(s.tenantID).MapQtyBySkuIDs(storeID, skuIDs)
	if err != nil {
		return nil, err
	}
	for _, it := range items {
		line := salesStockLinePlan{Item: it, StoreQty: storeQty[it.SkuID]}
		gap := it.Quantity - line.StoreQty
		if gap < 0 {
			gap = 0
		}
		if gap > 0 {
			line.WarehouseQty = s.lookupWarehouseStock(ctx, it.SkuID, it.SkuCode)
			line.TransferQty = gap
			if line.TransferQty > line.WarehouseQty {
				line.TransferQty = line.WarehouseQty
			}
			if line.TransferQty < 0 {
				line.TransferQty = 0
			}
			line.PurchaseQty = gap - line.TransferQty
		}
		if line.TransferQty > 0 {
			plan.NeedTransfer = true
		}
		if line.PurchaseQty > 0 {
			plan.NeedProcurement = true
		}
		plan.Lines = append(plan.Lines, line)
	}
	return plan, nil
}

func (s *SalesService) lookupWarehouseStock(ctx context.Context, skuID uint64, skuCode string) int {
	if s.pc == nil || skuID == 0 {
		return 0
	}
	keyword := strings.TrimSpace(skuCode)
	if keyword == "" {
		keyword = strconv.FormatUint(skuID, 10)
	}
	list, _, err := s.pc.SearchSkus(ctx, s.authToken, keyword, 1, 30)
	if err != nil {
		return 0
	}
	for _, row := range list {
		if row.SkuID == skuID {
			if row.Stock < 0 {
				return 0
			}
			return row.Stock
		}
	}
	return 0
}

func salesExpectedAt(order *model.StoreSalesOrder) *time.Time {
	if order.AppointmentAt != nil {
		t := *order.AppointmentAt
		return &t
	}
	if order.ExpectedDeliveryAt != nil {
		t := *order.ExpectedDeliveryAt
		return &t
	}
	if order.ExpressScheduledAt != nil {
		t := *order.ExpressScheduledAt
		return &t
	}
	return nil
}

func formatTimePtr(t *time.Time) *string {
	if t == nil {
		return nil
	}
	v := t.Format("2006-01-02T15:04:05")
	return &v
}

func (s *SalesService) createTransferFromPlan(order *model.StoreSalesOrder, plan *salesStockPlan) (*model.StockTransferOrder, error) {
	lines := make([]dto.StockTransferLineDTO, 0)
	for _, p := range plan.Lines {
		if p.TransferQty <= 0 {
			continue
		}
		lines = append(lines, dto.StockTransferLineDTO{
			SkuID:       p.Item.SkuID,
			SkuCode:     p.Item.SkuCode,
			ProductName: p.Item.ProductName,
			SpecLabel:   p.Item.SpecLabel,
			Pic:         p.Item.Pic,
			Quantity:    p.TransferQty,
		})
	}
	if len(lines) == 0 {
		return nil, nil
	}
	in := &dto.StockTransferOrderDTO{
		StoreID:    order.StoreID,
		ExpectedAt: formatTimePtr(salesExpectedAt(order)),
		Remark:     "来自销售单 " + order.OrderNo + "（门店缺货，仓库有货）",
		RefSalesID: order.ID,
		Items:      lines,
	}
	return NewStockTransferService(s.repos).ForTenant(s.tenantID).Create(in, order.CreatedBy)
}

func (s *SalesService) createPurchaseDraftFromPlan(order *model.StoreSalesOrder, plan *salesStockPlan, userID uint64) (*model.StorePurchaseOrder, error) {
	items := make([]dto.OrderLineDTO, 0)
	for _, p := range plan.Lines {
		if p.PurchaseQty <= 0 {
			continue
		}
		items = append(items, dto.OrderLineDTO{
			SkuID:       p.Item.SkuID,
			ProductName: p.Item.ProductName,
			SkuCode:     p.Item.SkuCode,
			SpecLabel:   p.Item.SpecLabel,
			Pic:         p.Item.Pic,
			Quantity:    p.PurchaseQty,
			UnitPrice:   p.Item.UnitPrice,
		})
	}
	if len(items) == 0 {
		return nil, nil
	}
	in := &dto.StorePurchaseOrderDTO{
		StoreID:      order.StoreID,
		PurchaseType: "sales_driven",
		RefSalesID:   order.ID,
		Remark:       "来自销售单 " + order.OrderNo + "（付款后自动生成草稿，暂无供应商）",
		Items:        items,
	}
	return NewPurchaseService(s.repos, s.pc).ForTenant(s.tenantID).Create(in, userID)
}
