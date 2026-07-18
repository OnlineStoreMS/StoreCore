package repo

import (
	"storecore/internal/dto"
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

func (r *SalesRepo) List(storeID uint64, f dto.SalesOrderListFilter, page, pageSize int) ([]model.StoreSalesOrder, int64, error) {
	q := r.db.Model(&model.StoreSalesOrder{}).Scopes(scopeTenant(r.tenantID))
	if storeID > 0 {
		q = q.Where("store_id = ?", storeID)
	}
	q = applyEq(q, "status", f.Status)
	q = applyEq(q, "pay_status", f.PayStatus)
	q = applyEq(q, "fulfillment_type", f.FulfillmentType)
	q = applyEq(q, "purchase_status", f.PurchaseStatus)
	q = applyOrderKeyword(q, f.Keyword)
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
	// 明细由 ReplaceItems 单独维护；Save 若带上 Preload 的 Items/ServiceItems，
	// GORM 会 ON CONFLICT 把已删除的旧明细再插回来，导致编辑保存后商品翻倍。
	return r.db.Scopes(scopeTenant(r.tenantID)).
		Omit("Items", "ServiceItems").
		Save(order).Error
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

func (r *SalesRepo) Delete(id uint64) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("tenant_id = ? AND sales_order_id = ?", r.tenantID, id).
			Delete(&model.StoreSalesOrderItem{}).Error; err != nil {
			return err
		}
		if err := tx.Where("tenant_id = ? AND sales_order_id = ?", r.tenantID, id).
			Delete(&model.StoreSalesOrderServiceItem{}).Error; err != nil {
			return err
		}
		res := tx.Scopes(scopeTenant(r.tenantID)).Delete(&model.StoreSalesOrder{}, id)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return nil
	})
}
