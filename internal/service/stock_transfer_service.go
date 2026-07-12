package service

import (
	"errors"
	"strings"
	"time"

	"storecore/internal/dto"
	"storecore/internal/model"
	"storecore/internal/repo"

	"gorm.io/gorm"
)

type StockTransferService struct {
	repos    *repo.Repos
	tenantID uint64
}

func NewStockTransferService(repos *repo.Repos) *StockTransferService {
	return &StockTransferService{repos: repos}
}

func (s *StockTransferService) ForTenant(tenantID uint64) *StockTransferService {
	return &StockTransferService{repos: s.repos, tenantID: repo.NormalizeTenantID(tenantID)}
}

func (s *StockTransferService) List(storeID uint64, page, pageSize int) ([]model.StockTransferOrder, int64, error) {
	return s.repos.StockTransfer.ForTenant(s.tenantID).List(storeID, page, pageSize)
}

func (s *StockTransferService) Get(id uint64) (*model.StockTransferOrder, error) {
	item, err := s.repos.StockTransfer.ForTenant(s.tenantID).GetByID(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return item, err
}

func (s *StockTransferService) Create(in *dto.StockTransferOrderDTO, userID uint64) (*model.StockTransferOrder, error) {
	if in.StoreID == 0 || len(in.Items) == 0 {
		return nil, ErrBadRequest
	}
	items := make([]model.StockTransferOrderItem, 0, len(in.Items))
	for _, line := range in.Items {
		if line.SkuID == 0 || line.Quantity <= 0 || strings.TrimSpace(line.ProductName) == "" {
			return nil, ErrBadRequest
		}
		items = append(items, model.StockTransferOrderItem{
			SkuID: line.SkuID, SkuCode: line.SkuCode,
			ProductName: line.ProductName, SpecLabel: line.SpecLabel,
			Pic: line.Pic, Quantity: line.Quantity,
		})
	}

	var expectedAt *time.Time
	if in.ExpectedAt != nil && strings.TrimSpace(*in.ExpectedAt) != "" {
		t, err := parseFlexibleTime(*in.ExpectedAt)
		if err != nil {
			return nil, ErrBadRequest
		}
		expectedAt = &t
	}

	reminderEnabled := in.ReminderEnabled
	reminderChannel := "wechat"
	reminderStatus := "none"
	var reminderAt *time.Time
	if reminderEnabled {
		// 平台消息提醒预留：写入待发送，暂不实际推送
		reminderStatus = "pending"
		if in.ReminderAt != nil && strings.TrimSpace(*in.ReminderAt) != "" {
			t, err := parseFlexibleTime(*in.ReminderAt)
			if err != nil {
				return nil, ErrBadRequest
			}
			reminderAt = &t
		} else if expectedAt != nil {
			t := *expectedAt
			reminderAt = &t
		}
	}

	order := &model.StockTransferOrder{
		StoreID:         in.StoreID,
		OrderNo:         genOrderNo("STF"),
		Status:          "pending",
		ExpectedAt:      expectedAt,
		Remark:          strings.TrimSpace(in.Remark),
		RefSalesID:      in.RefSalesID,
		ReminderEnabled: reminderEnabled,
		ReminderAt:      reminderAt,
		ReminderChannel: reminderChannel,
		ReminderStatus:  reminderStatus,
		CreatedBy:       userID,
	}
	if err := s.repos.StockTransfer.ForTenant(s.tenantID).Create(order, items); err != nil {
		return nil, err
	}
	return order, nil
}

func (s *StockTransferService) Cancel(id uint64) (*model.StockTransferOrder, error) {
	r := s.repos.StockTransfer.ForTenant(s.tenantID)
	order, err := r.GetByID(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if order.Status != "pending" {
		return nil, ErrInvalidStatus
	}
	order.Status = "cancelled"
	if err := r.Save(order); err != nil {
		return nil, err
	}
	return order, nil
}

func (s *StockTransferService) Confirm(id uint64) (*model.StockTransferOrder, error) {
	r := s.repos.StockTransfer.ForTenant(s.tenantID)
	order, err := r.GetByID(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if order.Status != "pending" {
		return nil, ErrInvalidStatus
	}
	inv := s.repos.Inventory.ForTenant(s.tenantID)
	for _, line := range order.Items {
		if err := inv.AddQuantity(order.StoreID, line.SkuID, line.SkuCode, line.ProductName, line.SpecLabel, line.Pic, line.Quantity); err != nil {
			return nil, err
		}
		// TODO: 扣减仓库/中央仓库存。当前库存系统未就绪，商品中央库存保持不变，仅完成门店分配记账。
	}
	now := time.Now()
	order.Status = "received"
	order.ReceivedAt = &now
	if err := r.Save(order); err != nil {
		return nil, err
	}
	return order, nil
}
