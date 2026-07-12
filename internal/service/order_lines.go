package service

import (
	"storecore/internal/dto"
	"storecore/internal/model"
)

func buildSalesItems(lines []dto.OrderLineDTO) (items []model.StoreSalesOrderItem, originalTotal, payableTotal float64) {
	items = make([]model.StoreSalesOrderItem, 0, len(lines))
	for _, line := range lines {
		orig, disc, unit := normalizeLinePrices(line.OriginalPrice, line.Discount, line.UnitPrice)
		qty := line.Quantity
		if qty <= 0 {
			qty = 1
		}
		lineOrig := roundMoney(orig * float64(qty))
		linePay := roundMoney(unit * float64(qty))
		originalTotal += lineOrig
		payableTotal += linePay
		items = append(items, model.StoreSalesOrderItem{
			SkuID:         line.SkuID,
			ProductName:   line.ProductName,
			SkuCode:       line.SkuCode,
			SpecLabel:     line.SpecLabel,
			Pic:           line.Pic,
			Quantity:      qty,
			OriginalPrice: orig,
			Discount:      disc,
			UnitPrice:     unit,
			TotalAmount:   linePay,
		})
	}
	return items, roundMoney(originalTotal), roundMoney(payableTotal)
}

func buildSalesServiceItems(lines []dto.SalesServiceLineDTO) []model.StoreSalesOrderServiceItem {
	items := make([]model.StoreSalesOrderServiceItem, 0, len(lines))
	for _, line := range lines {
		if line.ServiceItemID == 0 {
			continue
		}
		qty := line.Quantity
		if qty <= 0 {
			qty = 1
		}
		items = append(items, model.StoreSalesOrderServiceItem{
			ServiceItemID: line.ServiceItemID,
			ServiceName:   line.ServiceName,
			ServiceCode:   line.ServiceCode,
			Quantity:      qty,
			UnitPrice:     line.UnitPrice,
			TotalAmount:   roundMoney(line.UnitPrice * float64(qty)),
			DurationMin:   line.DurationMin,
			Pic:           line.Pic,
		})
	}
	return items
}

func buildPurchaseItems(lines []dto.OrderLineDTO) ([]model.StorePurchaseOrderItem, float64) {
	total := 0.0
	items := make([]model.StorePurchaseOrderItem, 0, len(lines))
	for _, line := range lines {
		lineTotal := line.UnitPrice * float64(line.Quantity)
		total += lineTotal
		items = append(items, model.StorePurchaseOrderItem{
			SkuID: line.SkuID, ProductName: line.ProductName, SkuCode: line.SkuCode,
			Quantity: line.Quantity, UnitPrice: line.UnitPrice, TotalAmount: lineTotal,
		})
	}
	return items, total
}
