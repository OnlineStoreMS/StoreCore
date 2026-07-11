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

func (s *ServiceOrderService) UpdateStatus(id uint64, status string) (*model.ServiceOrder, error) {
	r := s.repos.Service.ForTenant(s.tenantID)
	item, err := r.GetByID(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	allowed := map[string][]string{
		"in_progress": {"pending"},
		"completed":   {"pending", "in_progress"},
		"cancelled":   {"pending", "in_progress"},
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
	if existing.Status != "pending" {
		return nil, ErrInvalidStatus
	}
	order, items, err := s.buildServiceOrder(in, existing.CreatedBy)
	if err != nil {
		return nil, err
	}
	order.ID = existing.ID
	order.OrderNo = existing.OrderNo
	order.Status = existing.Status
	order.CreatedAt = existing.CreatedAt
	order.TenantID = existing.TenantID
	if err := r.Update(order, items); err != nil {
		return nil, err
	}
	return order, nil
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
		// 微信消息提醒预留：写入待发送状态，当前不实际推送
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
		ServiceType:     mode, // 兼容旧列表字段
		Status:          "pending",
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
