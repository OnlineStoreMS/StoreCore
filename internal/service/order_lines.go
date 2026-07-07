package service

import (
	"storecore/internal/dto"
	"storecore/internal/model"
)

func buildSalesItems(lines []dto.OrderLineDTO) ([]model.StoreSalesOrderItem, float64) {
	total := 0.0
	items := make([]model.StoreSalesOrderItem, 0, len(lines))
	for _, line := range lines {
		lineTotal := line.UnitPrice * float64(line.Quantity)
		total += lineTotal
		items = append(items, model.StoreSalesOrderItem{
			SkuID: line.SkuID, ProductName: line.ProductName, SkuCode: line.SkuCode,
			SpecLabel: line.SpecLabel, Quantity: line.Quantity,
			UnitPrice: line.UnitPrice, TotalAmount: lineTotal,
		})
	}
	return items, total
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
