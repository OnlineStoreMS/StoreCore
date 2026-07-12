package repo

import (
	"storecore/internal/model"

	"gorm.io/gorm"
)

type ReceiptTemplateRepo struct {
	db       *gorm.DB
	tenantID uint64
}

func NewReceiptTemplateRepo(db *gorm.DB) *ReceiptTemplateRepo {
	return &ReceiptTemplateRepo{db: db}
}

func (r *ReceiptTemplateRepo) ForTenant(tenantID uint64) *ReceiptTemplateRepo {
	return &ReceiptTemplateRepo{db: r.db, tenantID: NormalizeTenantID(tenantID)}
}

func (r *ReceiptTemplateRepo) List(storeID uint64, receiptType string, page, pageSize int) ([]model.ReceiptTemplate, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	q := r.db.Model(&model.ReceiptTemplate{}).Scopes(scopeTenant(r.tenantID))
	if storeID > 0 {
		q = q.Where("store_id = ? OR store_id = 0", storeID)
	}
	if receiptType != "" {
		q = q.Where("receipt_type = ?", receiptType)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.ReceiptTemplate
	offset := (page - 1) * pageSize
	err := q.Order("is_default DESC, id DESC").Offset(offset).Limit(pageSize).Find(&list).Error
	return list, total, err
}

func (r *ReceiptTemplateRepo) GetByID(id uint64) (*model.ReceiptTemplate, error) {
	var item model.ReceiptTemplate
	err := r.db.Scopes(scopeTenant(r.tenantID)).First(&item, id).Error
	return &item, err
}

func (r *ReceiptTemplateRepo) FindDefault(storeID uint64, receiptType string) (*model.ReceiptTemplate, error) {
	var item model.ReceiptTemplate
	q := r.db.Scopes(scopeTenant(r.tenantID)).
		Where("status = 1 AND is_default = ? AND receipt_type = ?", true, receiptType)
	if storeID > 0 {
		q = q.Where("store_id = ? OR store_id = 0", storeID)
	}
	err := q.Order("store_id DESC, id DESC").First(&item).Error
	return &item, err
}

func (r *ReceiptTemplateRepo) Create(item *model.ReceiptTemplate) error {
	item.TenantID = r.tenantID
	return r.db.Transaction(func(tx *gorm.DB) error {
		if item.IsDefault {
			if err := clearDefaultTemplates(tx, r.tenantID, item.StoreID, item.ReceiptType); err != nil {
				return err
			}
		}
		return tx.Create(item).Error
	})
}

func (r *ReceiptTemplateRepo) Update(item *model.ReceiptTemplate) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if item.IsDefault {
			if err := clearDefaultTemplates(tx, r.tenantID, item.StoreID, item.ReceiptType); err != nil {
				return err
			}
		}
		return tx.Scopes(scopeTenant(r.tenantID)).Save(item).Error
	})
}

func (r *ReceiptTemplateRepo) Delete(id uint64) error {
	return r.db.Scopes(scopeTenant(r.tenantID)).Delete(&model.ReceiptTemplate{}, id).Error
}

func clearDefaultTemplates(tx *gorm.DB, tenantID, storeID uint64, receiptType string) error {
	q := tx.Model(&model.ReceiptTemplate{}).
		Where("tenant_id = ? AND receipt_type = ?", tenantID, receiptType)
	if storeID > 0 {
		q = q.Where("store_id = ? OR store_id = 0", storeID)
	} else {
		q = q.Where("store_id = 0")
	}
	return q.Update("is_default", false).Error
}
