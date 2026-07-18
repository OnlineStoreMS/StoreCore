package repo

import (
	"storecore/internal/dto"
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

func (r *PosRepo) List(storeID uint64, f dto.PosOrderListFilter, page, pageSize int) ([]model.PosOrder, int64, error) {
	q := r.db.Model(&model.PosOrder{}).Scopes(scopeTenant(r.tenantID))
	if storeID > 0 {
		q = q.Where("store_id = ?", storeID)
	}
	q = applyEq(q, "status", f.Status)
	q = applyEq(q, "pay_status", f.PayStatus)
	q = applyEq(q, "payment_method", f.PaymentMethod)
	q = applyOrderKeyword(q, f.Keyword)
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

func (r *PosRepo) UpdateWithItems(order *model.PosOrder, items []model.PosOrderItem) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Scopes(scopeTenant(r.tenantID)).Save(order).Error; err != nil {
			return err
		}
		if err := tx.Where("tenant_id = ? AND pos_order_id = ?", r.tenantID, order.ID).
			Delete(&model.PosOrderItem{}).Error; err != nil {
			return err
		}
		for i := range items {
			items[i].ID = 0
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

func (r *PosRepo) Delete(id uint64) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("tenant_id = ? AND pos_order_id = ?", r.tenantID, id).
			Delete(&model.PosOrderItem{}).Error; err != nil {
			return err
		}
		res := tx.Scopes(scopeTenant(r.tenantID)).Delete(&model.PosOrder{}, id)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return nil
	})
}
