package repo

import (
	"storecore/internal/model"

	"gorm.io/gorm"
)

type SalesRepo struct {
	db       *gorm.DB
	tenantID uint64
}

func NewSalesRepo(db *gorm.DB) *SalesRepo {
	return &SalesRepo{db: db}
}

func (r *SalesRepo) ForTenant(tenantID uint64) *SalesRepo {
	return &SalesRepo{db: r.db, tenantID: NormalizeTenantID(tenantID)}
}

func (r *SalesRepo) List(storeID uint64, status string, page, pageSize int) ([]model.StoreSalesOrder, int64, error) {
	q := r.db.Model(&model.StoreSalesOrder{}).Scopes(scopeTenant(r.tenantID))
	if storeID > 0 {
		q = q.Where("store_id = ?", storeID)
	}
	if status != "" {
		q = q.Where("status = ?", status)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.StoreSalesOrder
	offset := (page - 1) * pageSize
	err := q.Preload("Items").Preload("ServiceItems").Order("id DESC").Offset(offset).Limit(pageSize).Find(&list).Error
	return list, total, err
}

func (r *SalesRepo) GetByID(id uint64) (*model.StoreSalesOrder, error) {
	var item model.StoreSalesOrder
	err := r.db.Scopes(scopeTenant(r.tenantID)).Preload("Items").Preload("ServiceItems").First(&item, id).Error
	return &item, err
}

func (r *SalesRepo) Create(order *model.StoreSalesOrder, items []model.StoreSalesOrderItem, serviceItems []model.StoreSalesOrderServiceItem) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		order.TenantID = r.tenantID
		if err := tx.Create(order).Error; err != nil {
			return err
		}
		for i := range items {
			items[i].TenantID = r.tenantID
			items[i].SalesOrderID = order.ID
		}
		if len(items) > 0 {
			if err := tx.Create(&items).Error; err != nil {
				return err
			}
		}
		for i := range serviceItems {
			serviceItems[i].TenantID = r.tenantID
			serviceItems[i].SalesOrderID = order.ID
		}
		if len(serviceItems) > 0 {
			if err := tx.Create(&serviceItems).Error; err != nil {
				return err
			}
		}
		order.Items = items
		order.ServiceItems = serviceItems
		return nil
	})
}

func (r *SalesRepo) Save(order *model.StoreSalesOrder) error {
	return r.db.Scopes(scopeTenant(r.tenantID)).Save(order).Error
}

func (r *SalesRepo) ReplaceItems(orderID uint64, items []model.StoreSalesOrderItem, serviceItems []model.StoreSalesOrderServiceItem) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Scopes(scopeTenant(r.tenantID)).Where("sales_order_id = ?", orderID).
			Delete(&model.StoreSalesOrderItem{}).Error; err != nil {
			return err
		}
		if err := tx.Scopes(scopeTenant(r.tenantID)).Where("sales_order_id = ?", orderID).
			Delete(&model.StoreSalesOrderServiceItem{}).Error; err != nil {
			return err
		}
		for i := range items {
			items[i].TenantID = r.tenantID
			items[i].SalesOrderID = orderID
		}
		if len(items) > 0 {
			if err := tx.Create(&items).Error; err != nil {
				return err
			}
		}
		for i := range serviceItems {
			serviceItems[i].TenantID = r.tenantID
			serviceItems[i].SalesOrderID = orderID
		}
		if len(serviceItems) > 0 {
			return tx.Create(&serviceItems).Error
		}
		return nil
	})
}
