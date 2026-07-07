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
	err := q.Order("id DESC").Offset(offset).Limit(pageSize).Find(&list).Error
	return list, total, err
}

func (r *ServiceRepo) GetByID(id uint64) (*model.ServiceOrder, error) {
	var item model.ServiceOrder
	err := r.db.Scopes(scopeTenant(r.tenantID)).First(&item, id).Error
	return &item, err
}

func (r *ServiceRepo) Create(item *model.ServiceOrder) error {
	item.TenantID = r.tenantID
	return r.db.Create(item).Error
}

func (r *ServiceRepo) Update(item *model.ServiceOrder) error {
	return r.db.Scopes(scopeTenant(r.tenantID)).Save(item).Error
}
