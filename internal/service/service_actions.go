package service

import (
	"errors"
	"strings"
	"time"

	"storecore/internal/dto"
	"storecore/internal/model"

	"gorm.io/gorm"
)

func (s *ServiceOrderService) Get(id uint64) (*model.ServiceOrder, error) {
	item, err := s.repos.Service.ForTenant(s.tenantID).GetByID(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return item, err
}

func (s *ServiceOrderService) Delete(id uint64) error {
	r := s.repos.Service.ForTenant(s.tenantID)
	item, err := r.GetByID(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	// 已关联收银或已付款不可删
	if item.PosOrderID > 0 || item.PayStatus == "paid" || item.Status == "completed" {
		return ErrInvalidStatus
	}
	if item.Status != "pending" && item.Status != "cancelled" {
		return ErrInvalidStatus
	}
	if err := r.Delete(id); errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound
	} else if err != nil {
		return err
	}
	return nil
}

func (s *ServiceOrderService) UpdateStatus(id uint64, status string) (*model.ServiceOrder, error) {
	r := s.repos.Service.ForTenant(s.tenantID)
	item, err := r.GetByID(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	// completed（已付款）仅由收银结算回写，禁止手工跳到 completed
	allowed := map[string][]string{
		"in_progress":      {"pending"},
		"awaiting_payment": {"in_progress"},
		"cancelled":        {"pending", "in_progress", "awaiting_payment"},
	}
	from, ok := allowed[status]
	if !ok {
		return nil, ErrBadRequest
	}
	valid := false
	for _, st := range from {
		if item.Status == st {
			valid = true
			break
		}
	}
	if !valid {
		return nil, ErrInvalidStatus
	}
	item.Status = status
	if err := r.Update(item, nil); err != nil {
		return nil, err
	}
	if item.SalesOrderID > 0 {
		_ = NewSalesService(s.repos).ForTenant(s.tenantID).SyncServiceStatus(item.SalesOrderID, status)
	}
	return item, nil
}

func (s *ServiceOrderService) Update(id uint64, in *dto.ServiceOrderDTO) (*model.ServiceOrder, error) {
	r := s.repos.Service.ForTenant(s.tenantID)
	existing, err := r.GetByID(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	// 仅待处理可改明细；进行中仅允许改备注类字段也可放宽为 pending
	if existing.Status != "pending" {
		return nil, ErrInvalidStatus
	}
	if existing.PosOrderID > 0 {
		return nil, ErrInvalidStatus
	}
	order, items, err := s.buildServiceOrder(in, existing.CreatedBy)
	if err != nil {
		return nil, err
	}
	order.ID = existing.ID
	order.OrderNo = existing.OrderNo
	order.Status = existing.Status
	order.PayStatus = existing.PayStatus
	order.PosOrderID = existing.PosOrderID
	order.PosOrderNo = existing.PosOrderNo
	order.ReceiptHTML = existing.ReceiptHTML
	order.CreatedAt = existing.CreatedAt
	order.TenantID = existing.TenantID
	if err := r.Update(order, items); err != nil {
		return nil, err
	}
	return order, nil
}

// MarkPaidByPos 收银结算成功后回写服务工单为已完成/已付款，并保存小票。
func (s *ServiceOrderService) MarkPaidByPos(serviceOrderID uint64, posOrder *model.PosOrder) error {
	if serviceOrderID == 0 || posOrder == nil {
		return nil
	}
	r := s.repos.Service.ForTenant(s.tenantID)
	item, err := r.GetByID(serviceOrderID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if item.Status != "awaiting_payment" && item.Status != "in_progress" && item.Status != "completed" {
		return ErrInvalidStatus
	}
	item.Status = "completed"
	item.PayStatus = "paid"
	item.PosOrderID = posOrder.ID
	item.PosOrderNo = posOrder.OrderNo
	item.ReceiptHTML = posOrder.ReceiptHTML
	if err := r.Update(item, nil); err != nil {
		return err
	}
	if item.SalesOrderID > 0 {
		_ = NewSalesService(s.repos).ForTenant(s.tenantID).SyncServiceStatus(item.SalesOrderID, "completed")
	}
	return nil
}

// LinkPosOrder 未收款收银单先关联工单，避免重复结算。
func (s *ServiceOrderService) LinkPosOrder(serviceOrderID uint64, posOrder *model.PosOrder) error {
	if serviceOrderID == 0 || posOrder == nil {
		return nil
	}
	r := s.repos.Service.ForTenant(s.tenantID)
	item, err := r.GetByID(serviceOrderID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	item.PosOrderID = posOrder.ID
	item.PosOrderNo = posOrder.OrderNo
	if item.Status == "in_progress" {
		item.Status = "awaiting_payment"
	}
	return r.Update(item, nil)
}

func (s *ServiceOrderService) buildServiceOrder(in *dto.ServiceOrderDTO, userID uint64) (*model.ServiceOrder, []model.ServiceOrderItem, error) {
	if in.StoreID == 0 || len(in.Items) == 0 {
		return nil, nil, ErrBadRequest
	}
	mode := normalizeOrderMode(in.OrderMode)
	ids := make([]uint64, 0, len(in.Items))
	qtyByID := map[uint64]int{}
	for _, line := range in.Items {
		if line.ServiceItemID == 0 {
			return nil, nil, ErrBadRequest
		}
		q := line.Quantity
		if q <= 0 {
			q = 1
		}
		if _, ok := qtyByID[line.ServiceItemID]; !ok {
			ids = append(ids, line.ServiceItemID)
		}
		qtyByID[line.ServiceItemID] += q
	}
	catalog := s.repos.ServiceCatalog.ForTenant(s.tenantID)
	catalogItems, err := catalog.ListItemsByIDs(ids)
	if err != nil {
		return nil, nil, err
	}
	if len(catalogItems) != len(ids) {
		return nil, nil, ErrBadRequest
	}
	byID := map[uint64]model.ServiceItem{}
	for _, it := range catalogItems {
		byID[it.ID] = it
	}

	items := make([]model.ServiceOrderItem, 0, len(ids))
	estimated := 0.0
	for _, id := range ids {
		src := byID[id]
		if src.Status == 0 {
			return nil, nil, ErrBadRequest
		}
		qty := qtyByID[id]
		lineTotal := roundMoney(src.Price * float64(qty))
		estimated += lineTotal
		items = append(items, model.ServiceOrderItem{
			ServiceItemID: src.ID,
			ServiceName:   src.Name,
			ServiceCode:   src.Code,
			Quantity:      qty,
			UnitPrice:     src.Price,
			TotalAmount:   lineTotal,
			DurationMin:   src.DurationMin,
			Pic:           src.Pic,
		})
	}
	estimated = roundMoney(estimated)

	var appointmentAt *time.Time
	if in.AppointmentAt != nil && strings.TrimSpace(*in.AppointmentAt) != "" {
		t, err := parseFlexibleTime(*in.AppointmentAt)
		if err != nil {
			return nil, nil, ErrBadRequest
		}
		appointmentAt = &t
	}
	if mode == "appointment" && appointmentAt == nil {
		return nil, nil, ErrBadRequest
	}

	reminderEnabled := in.ReminderEnabled
	reminderChannel := "wechat"
	reminderStatus := "none"
	var reminderAt *time.Time
	if reminderEnabled {
		reminderStatus = "pending"
		if in.ReminderAt != nil && strings.TrimSpace(*in.ReminderAt) != "" {
			t, err := parseFlexibleTime(*in.ReminderAt)
			if err != nil {
				return nil, nil, ErrBadRequest
			}
			reminderAt = &t
		} else if appointmentAt != nil {
			t := appointmentAt.Add(-30 * time.Minute)
			reminderAt = &t
		} else {
			t := time.Now().Add(30 * time.Minute)
			reminderAt = &t
		}
	}

	order := &model.ServiceOrder{
		StoreID:         in.StoreID,
		OrderNo:         genOrderNo("SRV"),
		OrderMode:       mode,
		ServiceType:     mode,
		Status:          "pending",
		PayStatus:       "unpaid",
		CustomerName:    strings.TrimSpace(in.CustomerName),
		CustomerPhone:   strings.TrimSpace(in.CustomerPhone),
		DeviceInfo:      strings.TrimSpace(in.DeviceInfo),
		FaultDesc:       strings.TrimSpace(in.FaultDesc),
		AppointmentAt:   appointmentAt,
		EngineerName:    strings.TrimSpace(in.EngineerName),
		EstimatedAmount: estimated,
		ReminderEnabled: reminderEnabled,
		ReminderAt:      reminderAt,
		ReminderChannel: reminderChannel,
		ReminderStatus:  reminderStatus,
		Remark:          strings.TrimSpace(in.Remark),
		CreatedBy:       userID,
	}
	return order, items, nil
}

func normalizeOrderMode(mode string) string {
	switch strings.TrimSpace(mode) {
	case "instant":
		return "instant"
	default:
		return "appointment"
	}
}

func parseFlexibleTime(v string) (time.Time, error) {
	v = strings.TrimSpace(v)
	layouts := []string{
		time.RFC3339,
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
		"2006-01-02T15:04:05",
		"2006-01-02T15:04",
	}
	var lastErr error
	for _, layout := range layouts {
		t, err := time.ParseInLocation(layout, v, time.Local)
		if err == nil {
			return t, nil
		}
		lastErr = err
	}
	return time.Time{}, lastErr
}
