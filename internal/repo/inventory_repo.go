package repo

import (
	"storecore/internal/model"

	"gorm.io/gorm"
)

type InventoryRepo struct {
	db       *gorm.DB
	tenantID uint64
}

func NewInventoryRepo(db *gorm.DB) *InventoryRepo {
	return &InventoryRepo{db: db}
}

func (r *InventoryRepo) ForTenant(tenantID uint64) *InventoryRepo {
	return &InventoryRepo{db: r.db, tenantID: NormalizeTenantID(tenantID)}
}

func (r *InventoryRepo) List(storeID uint64, keyword string, page, pageSize int) ([]model.StoreInventory, int64, error) {
	q := r.db.Model(&model.StoreInventory{}).Scopes(scopeTenant(r.tenantID))
	if storeID > 0 {
		q = q.Where("store_id = ?", storeID)
	}
	if keyword != "" {
		like := "%" + keyword + "%"
		q = q.Where("product_name ILIKE ? OR sku_code ILIKE ?", like, like)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.StoreInventory
	offset := (page - 1) * pageSize
	err := q.Order("id DESC").Offset(offset).Limit(pageSize).Find(&list).Error
	return list, total, err
}

func (r *InventoryRepo) Upsert(item *model.StoreInventory) error {
	item.TenantID = r.tenantID
	var existing model.StoreInventory
	err := r.db.Scopes(scopeTenant(r.tenantID)).
		Where("store_id = ? AND sku_id = ?", item.StoreID, item.SkuID).
		First(&existing).Error
	if err == gorm.ErrRecordNotFound {
		return r.db.Create(item).Error
	}
	if err != nil {
		return err
	}
	existing.Quantity = item.Quantity
	existing.ReservedQty = item.ReservedQty
	existing.SafetyStock = item.SafetyStock
	existing.ProductName = item.ProductName
	existing.SkuCode = item.SkuCode
	existing.SpecLabel = item.SpecLabel
	return r.db.Save(&existing).Error
}
