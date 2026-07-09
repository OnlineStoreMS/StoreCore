package admin

import "github.com/gin-gonic/gin"

func RegisterRoutes(
	g *gin.RouterGroup,
	storeH *StoreHandler,
	posH *PosHandler,
	salesH *SalesHandler,
	serviceH *ServiceHandler,
	inventoryH *InventoryHandler,
	purchaseH *PurchaseHandler,
	surveillanceH *SurveillanceHandler,
	skuH *ProductSkuHandler,
	supplierH *SupplierHandler,
) {
	g.GET("/stores", storeH.List)
	g.POST("/stores", storeH.Create)
	g.GET("/stores/:id", storeH.Get)
	g.PUT("/stores/:id", storeH.Update)
	g.DELETE("/stores/:id", storeH.Delete)

	g.GET("/product-skus/search", skuH.Search)
	g.GET("/product-catalog/categories", skuH.CategoryTree)
	g.GET("/product-catalog/products", skuH.ListProducts)
	g.GET("/product-catalog/products/:id/skus", skuH.GetProductSkus)
	g.GET("/suppliers", supplierH.List)

	g.GET("/pos-orders", posH.List)
	g.POST("/pos-orders", posH.Create)
	g.GET("/pos-orders/:id", posH.Get)
	g.POST("/pos-orders/:id/mark-paid", posH.MarkPaid)

	g.GET("/sales-orders", salesH.List)
	g.POST("/sales-orders", salesH.Create)
	g.GET("/sales-orders/:id", salesH.Get)
	g.PUT("/sales-orders/:id", salesH.Update)
	g.POST("/sales-orders/:id/confirm", salesH.Confirm)
	g.POST("/sales-orders/:id/cancel", salesH.Cancel)
	g.POST("/sales-orders/:id/mark-ready", salesH.MarkReady)
	g.POST("/sales-orders/:id/ship", salesH.Ship)
	g.POST("/sales-orders/:id/complete", salesH.Complete)
	g.POST("/sales-orders/:id/purchase-orders", purchaseH.CreateFromSales)

	g.GET("/service-orders", serviceH.List)
	g.POST("/service-orders", serviceH.Create)
	g.GET("/service-orders/:id", serviceH.Get)
	g.PUT("/service-orders/:id", serviceH.Update)
	g.POST("/service-orders/:id/status", serviceH.UpdateStatus)

	g.GET("/inventories", inventoryH.List)
	g.POST("/inventories/adjust", inventoryH.Adjust)

	g.GET("/purchase-orders", purchaseH.List)
	g.POST("/purchase-orders", purchaseH.Create)
	g.GET("/purchase-orders/:id", purchaseH.Get)
	g.POST("/purchase-orders/:id/submit", purchaseH.Submit)
	g.POST("/purchase-orders/:id/receive", purchaseH.Receive)
	g.POST("/purchase-orders/:id/cancel", purchaseH.Cancel)

	g.GET("/surveillance-devices", surveillanceH.List)
	g.POST("/surveillance-devices", surveillanceH.Create)
}
