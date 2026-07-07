package repo

import (
	"storecore/internal/model"

	"gorm.io/gorm"
)

type StoreRepo struct {
	db       *gorm.DB
	tenantID uint64
}

func NewStoreRepo(db *gorm.DB) *StoreRepo {
	return &StoreRepo{db: db}
}

func (r *StoreRepo) ForTenant(tenantID uint64) *StoreRepo {
	return &StoreRepo{db: r.db, tenantID: NormalizeTenantID(tenantID)}
}

func (r *StoreRepo) List(keyword string, page, pageSize int) ([]model.Store, int64, error) {
	q := r.db.Model(&model.Store{}).Scopes(scopeTenant(r.tenantID))
	if keyword != "" {
		like := "%" + keyword + "%"
		q = q.Where("name ILIKE ? OR code ILIKE ?", like, like)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.Store
	offset := (page - 1) * pageSize
	err := q.Order("id DESC").Offset(offset).Limit(pageSize).Find(&list).Error
	return list, total, err
}

func (r *StoreRepo) GetByID(id uint64) (*model.Store, error) {
	var item model.Store
	err := r.db.Scopes(scopeTenant(r.tenantID)).First(&item, id).Error
	return &item, err
}

func (r *StoreRepo) GetByCode(code string) (*model.Store, error) {
	var item model.Store
	err := r.db.Scopes(scopeTenant(r.tenantID)).Where("code = ?", code).First(&item).Error
	return &item, err
}

func (r *StoreRepo) Create(item *model.Store) error {
	item.TenantID = r.tenantID
	return r.db.Create(item).Error
}

func (r *StoreRepo) Update(item *model.Store) error {
	return r.db.Scopes(scopeTenant(r.tenantID)).Save(item).Error
}

func (r *StoreRepo) Delete(id uint64) error {
	return r.db.Scopes(scopeTenant(r.tenantID)).Delete(&model.Store{}, id).Error
}
