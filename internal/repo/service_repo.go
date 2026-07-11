package repo

import (
	"storecore/internal/model"

	"gorm.io/gorm"
)

type ServiceRepo struct {
	db       *gorm.DB
	tenantID uint64
}

func NewServiceRepo(db *gorm.DB) *ServiceRepo {
	return &ServiceRepo{db: db}
}

func (r *ServiceRepo) ForTenant(tenantID uint64) *ServiceRepo {
	return &ServiceRepo{db: r.db, tenantID: NormalizeTenantID(tenantID)}
}

func (r *ServiceRepo) List(storeID uint64, page, pageSize int) ([]model.ServiceOrder, int64, error) {
	q := r.db.Model(&model.ServiceOrder{}).Scopes(scopeTenant(r.tenantID))
	if storeID > 0 {
		q = q.Where("store_id = ?", storeID)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.ServiceOrder
	offset := (page - 1) * pageSize
	err := q.Preload("Items").Order("id DESC").Offset(offset).Limit(pageSize).Find(&list).Error
	return list, total, err
}

func (r *ServiceRepo) GetByID(id uint64) (*model.ServiceOrder, error) {
	var item model.ServiceOrder
	err := r.db.Scopes(scopeTenant(r.tenantID)).Preload("Items").First(&item, id).Error
	return &item, err
}

func (r *ServiceRepo) Create(order *model.ServiceOrder, items []model.ServiceOrderItem) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		order.TenantID = r.tenantID
		if err := tx.Create(order).Error; err != nil {
			return err
		}
		for i := range items {
			items[i].TenantID = r.tenantID
			items[i].ServiceOrderID = order.ID
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

func (r *ServiceRepo) Update(order *model.ServiceOrder, items []model.ServiceOrderItem) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Scopes(scopeTenant(r.tenantID)).Save(order).Error; err != nil {
			return err
		}
		if items != nil {
			if err := tx.Where("tenant_id = ? AND service_order_id = ?", r.tenantID, order.ID).
				Delete(&model.ServiceOrderItem{}).Error; err != nil {
				return err
			}
			for i := range items {
				items[i].ID = 0
				items[i].TenantID = r.tenantID
				items[i].ServiceOrderID = order.ID
			}
			if len(items) > 0 {
				if err := tx.Create(&items).Error; err != nil {
					return err
				}
			}
			order.Items = items
		}
		return nil
	})
}
