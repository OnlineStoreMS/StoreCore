package database

import (
	"fmt"
	"os"
	"path/filepath"

	"storecore/internal/config"
	"storecore/internal/model"

	"github.com/glebarez/sqlite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func Connect(cfg *config.DatabaseConfig) (*gorm.DB, error) {
	var dialector gorm.Dialector
	switch cfg.Driver {
	case "postgres":
		dialector = postgres.Open(cfg.PostgresDSN)
	case "sqlite":
		if err := os.MkdirAll(filepath.Dir(cfg.SQLitePath), 0o755); err != nil {
			return nil, err
		}
		dialector = sqlite.Open(cfg.SQLitePath)
	default:
		return nil, fmt.Errorf("unsupported database driver: %s", cfg.Driver)
	}

	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, err
	}
	return db, nil
}

func AutoMigrate(db *gorm.DB) error {
	if err := db.AutoMigrate(
		&model.Store{},
		&model.PosOrder{},
		&model.PosOrderItem{},
		&model.StoreSalesOrder{},
		&model.StoreSalesOrderItem{},
		&model.ServiceOrder{},
		&model.ServiceOrderItem{},
		&model.StoreInventory{},
		&model.StockTransferOrder{},
		&model.StockTransferOrderItem{},
		&model.StorePurchaseOrder{},
		&model.StorePurchaseOrderItem{},
		&model.SurveillanceDevice{},
		&model.ReceiptTemplate{},
		&model.ServiceCategory{},
		&model.ServiceItem{},
	); err != nil {
		return err
	}
	return ensureIndexes(db)
}

func ensureIndexes(db *gorm.DB) error {
	switch db.Dialector.Name() {
	case "postgres":
		return db.Exec(`
			CREATE UNIQUE INDEX IF NOT EXISTS idx_stores_tenant_code ON stores (tenant_id, code);
			CREATE UNIQUE INDEX IF NOT EXISTS idx_pos_orders_tenant_no ON pos_orders (tenant_id, order_no);
			CREATE UNIQUE INDEX IF NOT EXISTS idx_store_sales_orders_tenant_no ON store_sales_orders (tenant_id, order_no);
			CREATE UNIQUE INDEX IF NOT EXISTS idx_service_orders_tenant_no ON service_orders (tenant_id, order_no);
			CREATE UNIQUE INDEX IF NOT EXISTS idx_store_inventories_tenant_store_sku ON store_inventories (tenant_id, store_id, sku_id);
			CREATE UNIQUE INDEX IF NOT EXISTS idx_store_purchase_orders_tenant_no ON store_purchase_orders (tenant_id, po_no);
			CREATE UNIQUE INDEX IF NOT EXISTS idx_stock_transfer_orders_tenant_no ON stock_transfer_orders (tenant_id, order_no);
		`).Error
	default:
		return nil
	}
}
