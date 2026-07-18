package repo

import (
	"strings"

	"storecore/internal/dto"
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

func (r *ServiceRepo) List(storeID uint64, f dto.ServiceOrderListFilter, page, pageSize int) ([]model.ServiceOrder, int64, error) {
	q := r.db.Model(&model.ServiceOrder{}).Scopes(scopeTenant(r.tenantID))
	if storeID > 0 {
		q = q.Where("store_id = ?", storeID)
	}
	q = applyEq(q, "status", f.Status)
	q = applyEq(q, "pay_status", f.PayStatus)
	if mode := strings.TrimSpace(f.OrderMode); mode != "" {
		// 兼容旧数据：order_mode 为空时看 service_type
		q = q.Where("(order_mode = ? OR (order_mode = '' AND service_type = ?))", mode, mode)
	}
	q = applyOrderKeyword(q, f.Keyword)
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
	err := r.db.Scopes(scopeTenant(r.tenantID)).
		Preload("Items").
		Preload("ProcessRecords", func(db *gorm.DB) *gorm.DB {
			return db.Order("id ASC")
		}).
		First(&item, id).Error
	return &item, err
}

func (r *ServiceRepo) GetByIDs(ids []uint64) ([]model.ServiceOrder, error) {
	if len(ids) == 0 {
		return []model.ServiceOrder{}, nil
	}
	var list []model.ServiceOrder
	err := r.db.Scopes(scopeTenant(r.tenantID)).
		Preload("Items").
		Preload("ProcessRecords", func(db *gorm.DB) *gorm.DB {
			return db.Order("id ASC")
		}).
		Where("id IN ?", ids).Order("id ASC").Find(&list).Error
	return list, err
}

func (r *ServiceRepo) ListProcessRecords(orderID uint64) ([]model.ServiceProcessRecord, error) {
	var list []model.ServiceProcessRecord
	err := r.db.Scopes(scopeTenant(r.tenantID)).
		Where("service_order_id = ?", orderID).
		Order("id ASC").Find(&list).Error
	return list, err
}

func (r *ServiceRepo) CreateProcessRecord(rec *model.ServiceProcessRecord) error {
	rec.TenantID = r.tenantID
	return r.db.Create(rec).Error
}

func (r *ServiceRepo) UpdateProcessRecord(rec *model.ServiceProcessRecord) error {
	return r.db.Scopes(scopeTenant(r.tenantID)).Save(rec).Error
}

func (r *ServiceRepo) GetProcessRecord(id uint64) (*model.ServiceProcessRecord, error) {
	var rec model.ServiceProcessRecord
	err := r.db.Scopes(scopeTenant(r.tenantID)).First(&rec, id).Error
	return &rec, err
}

func (r *ServiceRepo) DeleteProcessRecord(id uint64) error {
	res := r.db.Scopes(scopeTenant(r.tenantID)).Delete(&model.ServiceProcessRecord{}, id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *ServiceRepo) CountProcessMediaByPhase(orderID uint64, phase string) (int, error) {
	var list []model.ServiceProcessRecord
	err := r.db.Scopes(scopeTenant(r.tenantID)).
		Where("service_order_id = ? AND phase = ?", orderID, phase).
		Find(&list).Error
	if err != nil {
		return 0, err
	}
	n := 0
	for _, rec := range list {
		n += len(rec.Media)
	}
	return n, nil
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

func (r *ServiceRepo) Delete(id uint64) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("tenant_id = ? AND service_order_id = ?", r.tenantID, id).
			Delete(&model.ServiceProcessRecord{}).Error; err != nil {
			return err
		}
		if err := tx.Where("tenant_id = ? AND service_order_id = ?", r.tenantID, id).
			Delete(&model.ServiceOrderItem{}).Error; err != nil {
			return err
		}
		res := tx.Scopes(scopeTenant(r.tenantID)).Delete(&model.ServiceOrder{}, id)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return nil
	})
}
