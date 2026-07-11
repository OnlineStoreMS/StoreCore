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
	receiptH *ReceiptTemplateHandler,
	catalogH *ServiceCatalogHandler,
	uploadH *UploadHandler,
) {
	g.GET("/stores", storeH.List)
	g.POST("/stores", storeH.Create)
	g.GET("/stores/:id", storeH.Get)
	g.PUT("/stores/:id", storeH.Update)
	g.DELETE("/stores/:id", storeH.Delete)

	g.POST("/upload", uploadH.Upload)

	g.GET("/product-skus/search", skuH.Search)
	g.GET("/product-catalog/categories", skuH.CategoryTree)
	g.GET("/product-catalog/products", skuH.ListProducts)
	g.GET("/product-catalog/products/:id/skus", skuH.GetProductSkus)
	g.GET("/suppliers", supplierH.List)

	g.GET("/pos-orders", posH.List)
	g.POST("/pos-orders", posH.Create)
	g.GET("/pos-orders/:id", posH.Get)
	g.POST("/pos-orders/:id/mark-paid", posH.MarkPaid)

	g.GET("/receipt-templates", receiptH.List)
	g.POST("/receipt-templates", receiptH.Create)
	g.GET("/receipt-templates/:id", receiptH.Get)
	g.PUT("/receipt-templates/:id", receiptH.Update)
	g.DELETE("/receipt-templates/:id", receiptH.Delete)

	g.GET("/service-categories/tree", catalogH.CategoryTree)
	g.POST("/service-categories", catalogH.CreateCategory)
	g.PUT("/service-categories/:id", catalogH.UpdateCategory)
	g.DELETE("/service-categories/:id", catalogH.DeleteCategory)
	g.GET("/service-items", catalogH.ListItems)
	g.POST("/service-items", catalogH.CreateItem)
	g.GET("/service-items/:id", catalogH.GetItem)
	g.PUT("/service-items/:id", catalogH.UpdateItem)
	g.DELETE("/service-items/:id", catalogH.DeleteItem)

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
