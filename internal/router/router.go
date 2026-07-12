package router

import (
	"path/filepath"

	"storecore/admin"
	adminmw "storecore/admin/middleware"
	"storecore/internal/config"
	jwtmgr "storecore/internal/pkg/jwt"
	"storecore/internal/integrations/productcore"
	"storecore/internal/integrations/supplycore"
	"storecore/internal/repo"
	"storecore/internal/service"
	"storecore/internal/storage"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func Setup(db *gorm.DB, cfg *config.Config) *gin.Engine {
	if cfg.Server.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery(), corsMiddleware(cfg))

	if cfg.Storage.Driver == "local" || cfg.Storage.Driver == "" {
		uploadDir := filepath.Join(cfg.Storage.LocalPath, cfg.Storage.Prefix)
		r.Static("/uploads", uploadDir)
	}

	store, err := storage.New(&cfg.Storage)
	if err != nil {
		panic(err)
	}

	repos := repo.New(db)
	storeSvc := service.NewStoreService(repos)
	posSvc := service.NewPosService(repos)
	salesSvc := service.NewSalesService(repos)
	serviceSvc := service.NewServiceOrderService(repos)
	inventorySvc := service.NewInventoryService(repos)
	purchaseSvc := service.NewPurchaseService(repos)
	surveillanceSvc := service.NewSurveillanceService(repos)
	receiptTplSvc := service.NewReceiptTemplateService(repos)
	serviceCatalogSvc := service.NewServiceCatalogService(repos)
	transferSvc := service.NewStockTransferService(repos)
	pcClient := productcore.NewClient(cfg.Integrations.ProductCoreAPIURL)
	scClient := supplycore.NewClient(cfg.Integrations.SupplyCoreAPIURL)

	storeH := admin.NewStoreHandler(storeSvc)
	posH := admin.NewPosHandler(posSvc)
	salesH := admin.NewSalesHandler(salesSvc)
	serviceH := admin.NewServiceHandler(serviceSvc)
	inventoryH := admin.NewInventoryHandler(inventorySvc, pcClient)
	purchaseH := admin.NewPurchaseHandler(purchaseSvc)
	surveillanceH := admin.NewSurveillanceHandler(surveillanceSvc)
	skuH := admin.NewProductSkuHandler(pcClient)
	supplierH := admin.NewSupplierHandler(scClient)
	receiptH := admin.NewReceiptTemplateHandler(receiptTplSvc)
	catalogH := admin.NewServiceCatalogHandler(serviceCatalogSvc)
	uploadH := admin.NewUploadHandler(store)
	transferH := admin.NewStockTransferHandler(transferSvc)

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok", "service": "storecore"})
	})

	v1 := r.Group("/api/v1")
	adminGroup := v1.Group("/admin")
	jwtMgr := jwtmgr.NewManager(cfg.Auth.JWTSecret)
	adminGroup.Use(adminmw.AdminAuth(&cfg.Auth, jwtMgr))
	admin.RegisterRoutes(adminGroup, storeH, posH, salesH, serviceH, inventoryH, purchaseH, surveillanceH, skuH, supplierH, receiptH, catalogH, uploadH, transferH)

	return r
}

func corsMiddleware(cfg *config.Config) gin.HandlerFunc {
	origins := cfg.CORS.AllowOrigins
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		allowed := origin == ""
		for _, o := range origins {
			if o == origin || o == "*" {
				allowed = true
				break
			}
		}
		if allowed && origin != "" {
			c.Header("Access-Control-Allow-Origin", origin)
		}
		c.Header("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type,Authorization")
		c.Header("Access-Control-Allow-Credentials", "true")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}
