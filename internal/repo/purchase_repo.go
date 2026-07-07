package repo

import (
	"storecore/internal/model"

	"gorm.io/gorm"
)

type PurchaseRepo struct {
	db       *gorm.DB
	tenantID uint64
}

func NewPurchaseRepo(db *gorm.DB) *PurchaseRepo {
	return &PurchaseRepo{db: db}
}

func (r *PurchaseRepo) ForTenant(tenantID uint64) *PurchaseRepo {
	return &PurchaseRepo{db: r.db, tenantID: NormalizeTenantID(tenantID)}
}

func (r *PurchaseRepo) List(storeID uint64, page, pageSize int) ([]model.StorePurchaseOrder, int64, error) {
	q := r.db.Model(&model.StorePurchaseOrder{}).Scopes(scopeTenant(r.tenantID))
	if storeID > 0 {
		q = q.Where("store_id = ?", storeID)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.StorePurchaseOrder
	offset := (page - 1) * pageSize
	err := q.Preload("Items").Order("id DESC").Offset(offset).Limit(pageSize).Find(&list).Error
	return list, total, err
}

func (r *PurchaseRepo) GetByID(id uint64) (*model.StorePurchaseOrder, error) {
	var item model.StorePurchaseOrder
	err := r.db.Scopes(scopeTenant(r.tenantID)).Preload("Items").First(&item, id).Error
	return &item, err
}

func (r *PurchaseRepo) Create(order *model.StorePurchaseOrder, items []model.StorePurchaseOrderItem) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		order.TenantID = r.tenantID
		if err := tx.Create(order).Error; err != nil {
			return err
		}
		for i := range items {
			items[i].TenantID = r.tenantID
			items[i].PurchaseOrderID = order.ID
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

func (r *PurchaseRepo) Save(order *model.StorePurchaseOrder) error {
	return r.db.Scopes(scopeTenant(r.tenantID)).Save(order).Error
}
