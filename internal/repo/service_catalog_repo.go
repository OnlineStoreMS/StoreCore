package repo

import (
	"storecore/internal/model"

	"gorm.io/gorm"
)

type ServiceCatalogRepo struct {
	db       *gorm.DB
	tenantID uint64
}

func NewServiceCatalogRepo(db *gorm.DB) *ServiceCatalogRepo {
	return &ServiceCatalogRepo{db: db}
}

func (r *ServiceCatalogRepo) ForTenant(tenantID uint64) *ServiceCatalogRepo {
	return &ServiceCatalogRepo{db: r.db, tenantID: NormalizeTenantID(tenantID)}
}

func (r *ServiceCatalogRepo) ListCategories() ([]model.ServiceCategory, error) {
	var list []model.ServiceCategory
	err := r.db.Scopes(scopeTenant(r.tenantID)).Order("sort ASC, id ASC").Find(&list).Error
	return list, err
}

func (r *ServiceCatalogRepo) GetCategory(id uint64) (*model.ServiceCategory, error) {
	var item model.ServiceCategory
	err := r.db.Scopes(scopeTenant(r.tenantID)).First(&item, id).Error
	return &item, err
}

func (r *ServiceCatalogRepo) CreateCategory(item *model.ServiceCategory) error {
	item.TenantID = r.tenantID
	return r.db.Create(item).Error
}

func (r *ServiceCatalogRepo) UpdateCategory(item *model.ServiceCategory) error {
	return r.db.Scopes(scopeTenant(r.tenantID)).Save(item).Error
}

func (r *ServiceCatalogRepo) DeleteCategory(id uint64) error {
	var count int64
	if err := r.db.Model(&model.ServiceItem{}).Scopes(scopeTenant(r.tenantID)).
		Where("category_id = ?", id).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return gorm.ErrForeignKeyViolated
	}
	var childCount int64
	if err := r.db.Model(&model.ServiceCategory{}).Scopes(scopeTenant(r.tenantID)).
		Where("parent_id = ?", id).Count(&childCount).Error; err != nil {
		return err
	}
	if childCount > 0 {
		return gorm.ErrForeignKeyViolated
	}
	return r.db.Scopes(scopeTenant(r.tenantID)).Delete(&model.ServiceCategory{}, id).Error
}

func (r *ServiceCatalogRepo) CountItemsByCategory() (map[uint64]int64, error) {
	type row struct {
		CategoryID uint64
		Cnt        int64
	}
	var rows []row
	err := r.db.Model(&model.ServiceItem{}).Scopes(scopeTenant(r.tenantID)).
		Select("category_id, count(*) as cnt").Group("category_id").Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make(map[uint64]int64, len(rows))
	for _, x := range rows {
		out[x.CategoryID] = x.Cnt
	}
	return out, nil
}

func (r *ServiceCatalogRepo) ListItems(categoryID uint64, keyword string, status *int8, page, pageSize int) ([]model.ServiceItem, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	q := r.db.Model(&model.ServiceItem{}).Scopes(scopeTenant(r.tenantID))
	if categoryID > 0 {
		q = q.Where("category_id = ?", categoryID)
	}
	if status != nil {
		q = q.Where("status = ?", *status)
	}
	if keyword != "" {
		like := "%" + keyword + "%"
		q = q.Where("name ILIKE ? OR code ILIKE ?", like, like)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.ServiceItem
	offset := (page - 1) * pageSize
	err := q.Order("sort ASC, id DESC").Offset(offset).Limit(pageSize).Find(&list).Error
	return list, total, err
}

func (r *ServiceCatalogRepo) GetItem(id uint64) (*model.ServiceItem, error) {
	var item model.ServiceItem
	err := r.db.Scopes(scopeTenant(r.tenantID)).First(&item, id).Error
	return &item, err
}

func (r *ServiceCatalogRepo) CreateItem(item *model.ServiceItem) error {
	item.TenantID = r.tenantID
	return r.db.Create(item).Error
}

func (r *ServiceCatalogRepo) UpdateItem(item *model.ServiceItem) error {
	return r.db.Scopes(scopeTenant(r.tenantID)).Save(item).Error
}

func (r *ServiceCatalogRepo) DeleteItem(id uint64) error {
	return r.db.Scopes(scopeTenant(r.tenantID)).Delete(&model.ServiceItem{}, id).Error
}
