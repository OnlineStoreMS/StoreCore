package repo

import "gorm.io/gorm"

type Repos struct {
	Store          *StoreRepo
	Pos            *PosRepo
	Sales          *SalesRepo
	Service        *ServiceRepo
	Inventory      *InventoryRepo
	Purchase       *PurchaseRepo
	Surveillance   *SurveillanceRepo
	ReceiptTpl     *ReceiptTemplateRepo
	ServiceCatalog *ServiceCatalogRepo
	StockTransfer  *StockTransferRepo
}

func New(db *gorm.DB) *Repos {
	return &Repos{
		Store:          NewStoreRepo(db),
		Pos:            NewPosRepo(db),
		Sales:          NewSalesRepo(db),
		Service:        NewServiceRepo(db),
		Inventory:      NewInventoryRepo(db),
		Purchase:       NewPurchaseRepo(db),
		Surveillance:   NewSurveillanceRepo(db),
		ReceiptTpl:     NewReceiptTemplateRepo(db),
		ServiceCatalog: NewServiceCatalogRepo(db),
		StockTransfer:  NewStockTransferRepo(db),
	}
}
