package repo

import (
	"storecore/internal/model"

	"gorm.io/gorm"
)

type PosRepo struct {
	db       *gorm.DB
	tenantID uint64
}

func NewPosRepo(db *gorm.DB) *PosRepo {
	return &PosRepo{db: db}
}

func (r *PosRepo) ForTenant(tenantID uint64) *PosRepo {
	return &PosRepo{db: r.db, tenantID: NormalizeTenantID(tenantID)}
}

func (r *PosRepo) List(storeID uint64, page, pageSize int) ([]model.PosOrder, int64, error) {
	q := r.db.Model(&model.PosOrder{}).Scopes(scopeTenant(r.tenantID))
	if storeID > 0 {
		q = q.Where("store_id = ?", storeID)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.PosOrder
	offset := (page - 1) * pageSize
	err := q.Preload("Items").Order("id DESC").Offset(offset).Limit(pageSize).Find(&list).Error
	return list, total, err
}

func (r *PosRepo) GetByID(id uint64) (*model.PosOrder, error) {
	var item model.PosOrder
	err := r.db.Scopes(scopeTenant(r.tenantID)).Preload("Items").First(&item, id).Error
	return &item, err
}

func (r *PosRepo) Create(order *model.PosOrder, items []model.PosOrderItem) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		order.TenantID = r.tenantID
		if err := tx.Create(order).Error; err != nil {
			return err
		}
		for i := range items {
			items[i].TenantID = r.tenantID
			items[i].PosOrderID = order.ID
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

func (r *PosRepo) Update(order *model.PosOrder) error {
	return r.db.Scopes(scopeTenant(r.tenantID)).Save(order).Error
}
