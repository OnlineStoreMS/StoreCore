package repo

import (
	"storecore/internal/model"

	"gorm.io/gorm"
)

type StockTransferRepo struct {
	db       *gorm.DB
	tenantID uint64
}

func NewStockTransferRepo(db *gorm.DB) *StockTransferRepo {
	return &StockTransferRepo{db: db}
}

func (r *StockTransferRepo) ForTenant(tenantID uint64) *StockTransferRepo {
	return &StockTransferRepo{db: r.db, tenantID: NormalizeTenantID(tenantID)}
}

func (r *StockTransferRepo) List(storeID uint64, page, pageSize int) ([]model.StockTransferOrder, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	q := r.db.Model(&model.StockTransferOrder{}).Scopes(scopeTenant(r.tenantID))
	if storeID > 0 {
		q = q.Where("store_id = ?", storeID)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.StockTransferOrder
	offset := (page - 1) * pageSize
	err := q.Preload("Items").Order("id DESC").Offset(offset).Limit(pageSize).Find(&list).Error
	return list, total, err
}

func (r *StockTransferRepo) GetByID(id uint64) (*model.StockTransferOrder, error) {
	var item model.StockTransferOrder
	err := r.db.Scopes(scopeTenant(r.tenantID)).Preload("Items").First(&item, id).Error
	return &item, err
}

func (r *StockTransferRepo) Create(order *model.StockTransferOrder, items []model.StockTransferOrderItem) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		order.TenantID = r.tenantID
		if err := tx.Create(order).Error; err != nil {
			return err
		}
		for i := range items {
			items[i].TenantID = r.tenantID
			items[i].TransferOrderID = order.ID
		}
		if len(items) > 0 {
			if err := tx.Create(&items).Error; err != nil {
				return err
			}
		}
		order.Items = items
		return nil
	})
}

func (r *StockTransferRepo) Save(order *model.StockTransferOrder) error {
	return r.db.Scopes(scopeTenant(r.tenantID)).Save(order).Error
}
