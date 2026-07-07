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
) {
	g.GET("/stores", storeH.List)
	g.POST("/stores", storeH.Create)
	g.GET("/stores/:id", storeH.Get)
	g.PUT("/stores/:id", storeH.Update)
	g.DELETE("/stores/:id", storeH.Delete)

	g.GET("/product-skus/search", skuH.Search)

	g.GET("/pos-orders", posH.List)
	g.POST("/pos-orders", posH.Create)
	g.GET("/pos-orders/:id", posH.Get)
	g.POST("/pos-orders/:id/mark-paid", posH.MarkPaid)

	g.GET("/sales-orders", salesH.List)
	g.POST("/sales-orders", salesH.Create)
	g.GET("/sales-orders/:id", salesH.Get)

	g.GET("/service-orders", serviceH.List)
	g.POST("/service-orders", serviceH.Create)

	g.GET("/inventories", inventoryH.List)
	g.POST("/inventories/adjust", inventoryH.Adjust)

	g.GET("/purchase-orders", purchaseH.List)
	g.POST("/purchase-orders", purchaseH.Create)

	g.GET("/surveillance-devices", surveillanceH.List)
	g.POST("/surveillance-devices", surveillanceH.Create)
}
