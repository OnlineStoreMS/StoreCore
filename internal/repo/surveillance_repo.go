package repo

import (
	"storecore/internal/model"

	"gorm.io/gorm"
)

type SurveillanceRepo struct {
	db       *gorm.DB
	tenantID uint64
}

func NewSurveillanceRepo(db *gorm.DB) *SurveillanceRepo {
	return &SurveillanceRepo{db: db}
}

func (r *SurveillanceRepo) ForTenant(tenantID uint64) *SurveillanceRepo {
	return &SurveillanceRepo{db: r.db, tenantID: NormalizeTenantID(tenantID)}
}

func (r *SurveillanceRepo) List(storeID uint64, page, pageSize int) ([]model.SurveillanceDevice, int64, error) {
	q := r.db.Model(&model.SurveillanceDevice{}).Scopes(scopeTenant(r.tenantID))
	if storeID > 0 {
		q = q.Where("store_id = ?", storeID)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.SurveillanceDevice
	offset := (page - 1) * pageSize
	err := q.Order("id DESC").Offset(offset).Limit(pageSize).Find(&list).Error
	return list, total, err
}

func (r *SurveillanceRepo) Create(item *model.SurveillanceDevice) error {
	item.TenantID = r.tenantID
	return r.db.Create(item).Error
}

func (r *SurveillanceRepo) Update(item *model.SurveillanceDevice) error {
	return r.db.Scopes(scopeTenant(r.tenantID)).Save(item).Error
}

func (r *SurveillanceRepo) Delete(id uint64) error {
	return r.db.Scopes(scopeTenant(r.tenantID)).Delete(&model.SurveillanceDevice{}, id).Error
}
