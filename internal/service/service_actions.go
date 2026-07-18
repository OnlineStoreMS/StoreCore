package service

import (
	"errors"
	"fmt"
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

// Delete 删除服务工单，并级联删除关联收银订单；若挂在销售单上则清除销售单上的服务关联。
func (s *ServiceOrderService) Delete(id uint64) error {
	return s.deleteWithCascade(id, true)
}

// deleteWithCascade 删除服务工单及关联收银单。
// clearSalesLink=true 时断开销售单上的 serviceOrder 引用（销售单自身删除时传 false）。
func (s *ServiceOrderService) deleteWithCascade(id uint64, clearSalesLink bool) error {
	r := s.repos.Service.ForTenant(s.tenantID)
	item, err := r.GetByID(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}

	// 先删关联收银订单
	if item.PosOrderID > 0 {
		pr := s.repos.Pos.ForTenant(s.tenantID)
		if err := pr.Delete(item.PosOrderID); err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
	}

	if clearSalesLink && item.SalesOrderID > 0 {
		sr := s.repos.Sales.ForTenant(s.tenantID)
		if sales, err := sr.GetByID(item.SalesOrderID); err == nil && sales != nil && sales.ServiceOrderID == id {
			sales.ServiceOrderID = 0
			sales.ServiceOrderNo = ""
			sales.ServiceStatus = "none"
			_ = sr.Save(sales)
		}
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
	// completed 可由待付款确认完成；亦可重开回进行中继续服务
	allowed := map[string][]string{
		"in_progress":      {"pending", "completed"}, // completed→in_progress 为重开
		"awaiting_payment": {"in_progress"},
		"completed":        {"awaiting_payment"},
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

	// 开始工单：必须已有「服务前」过程媒体
	if status == "in_progress" && item.Status == "pending" {
		if err := s.requireProcessPhase(id, "before", "开始工单前请先填写服务前过程纪录（至少一张图片或视频）"); err != nil {
			return nil, err
		}
	}
	// 完成服务：必须已有「服务后」过程媒体
	if status == "awaiting_payment" {
		if err := s.requireProcessPhase(id, "after", "完成服务前请先填写服务后过程纪录（至少一张图片或视频）"); err != nil {
			return nil, err
		}
	}

	if status == "awaiting_payment" && s.shouldSkipCashier(item) {
		// 服务已做完且已付款（或零元）：履约完成
		item.PayStatus = "paid"
		item.Status = "completed"
	} else if status == "completed" {
		// 待付款阶段确认履约完成（付款可已先完成，也可后续再补记）
		item.Status = "completed"
	} else {
		item.Status = status
	}

	s.attachServiceReceipt(item, item.Items)
	// 完成服务后生成/刷新服务报告
	if status == "awaiting_payment" || item.Status == "completed" {
		recs, _ := r.ListProcessRecords(item.ID)
		item.ProcessRecords = recs
		s.attachServiceReport(item)
	}
	if err := r.Update(item, nil); err != nil {
		return nil, err
	}
	if item.SalesOrderID > 0 {
		_ = NewSalesService(s.repos, nil).ForTenant(s.tenantID).SyncServiceStatus(item.SalesOrderID, item.Status)
	}
	return item, nil
}

// shouldSkipCashier 销售单已付款或金额为 0 时，完成服务后无需收银台收款。
func (s *ServiceOrderService) shouldSkipCashier(item *model.ServiceOrder) bool {
	if item == nil {
		return false
	}
	if item.PayStatus == "paid" {
		return true
	}
	if item.EstimatedAmount <= 0 {
		return true
	}
	if item.SalesOrderID == 0 {
		return false
	}
	so, err := s.repos.Sales.ForTenant(s.tenantID).GetByID(item.SalesOrderID)
	if err != nil {
		return false
	}
	return so.PayStatus == "paid"
}

// MarkPaidFromSales 销售单付款完成后，同步关联服务工单为已付款（不改变工单履约状态）。
func (s *ServiceOrderService) MarkPaidFromSales(serviceOrderID uint64) error {
	return s.MarkPaidFromSalesOrder(serviceOrderID, nil)
}

func (s *ServiceOrderService) MarkPaidFromSalesOrder(serviceOrderID uint64, sales *model.StoreSalesOrder) error {
	if serviceOrderID == 0 {
		return nil
	}
	r := s.repos.Service.ForTenant(s.tenantID)
	item, err := r.GetByID(serviceOrderID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	if item.PayStatus == "paid" {
		// 销售单修改付款信息时同步过来
		if sales != nil && item.Status != "completed" && item.Status != "cancelled" {
			if sales.PaymentMethod != "" {
				item.PaymentMethod = sales.PaymentMethod
			}
			if strings.TrimSpace(sales.PaymentProofURL) != "" {
				item.PaymentProofURL = sales.PaymentProofURL
			}
			if sales.PaidAt != nil {
				item.PaidAt = sales.PaidAt
			}
			s.attachServiceReceipt(item, item.Items)
			return r.Update(item, nil)
		}
		return nil
	}
	item.PayStatus = "paid"
	if item.PaymentMethod == "" {
		if sales != nil && sales.PaymentMethod != "" {
			item.PaymentMethod = sales.PaymentMethod
		} else {
			item.PaymentMethod = "sales"
		}
	}
	if sales != nil && strings.TrimSpace(sales.PaymentProofURL) != "" && item.PaymentProofURL == "" {
		item.PaymentProofURL = sales.PaymentProofURL
	}
	if sales != nil && sales.PaidAt != nil {
		item.PaidAt = sales.PaidAt
	} else if item.PaidAt == nil {
		now := time.Now()
		item.PaidAt = &now
	}
	s.attachServiceReceipt(item, item.Items)
	if err := r.Update(item, nil); err != nil {
		return err
	}
	return nil
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
	// 完成 / 取消前均可编辑
	switch existing.Status {
	case "pending", "in_progress", "awaiting_payment":
		// ok
	default:
		return nil, fmt.Errorf("%w：已完成或已取消的工单不可编辑", ErrInvalidStatus)
	}
	order, items, err := s.buildServiceOrder(in, existing.CreatedBy)
	if err != nil {
		return nil, err
	}
	order.ID = existing.ID
	order.OrderNo = existing.OrderNo
	order.Status = existing.Status
	order.PayStatus = existing.PayStatus
	order.PaymentMethod = existing.PaymentMethod
	order.PaymentProofURL = existing.PaymentProofURL
	order.PaidAt = existing.PaidAt
	order.PaidBy = existing.PaidBy
	if order.EstimatedAmount <= 0 {
		order.PayStatus = "paid"
	}
	order.PosOrderID = existing.PosOrderID
	order.PosOrderNo = existing.PosOrderNo
	order.ReceiptHTML = existing.ReceiptHTML
	order.CreatedAt = existing.CreatedAt
	order.TenantID = existing.TenantID
	order.SalesOrderID = existing.SalesOrderID
	order.SalesOrderNo = existing.SalesOrderNo
	s.attachServiceReceipt(order, items)
	if err := r.Update(order, items); err != nil {
		return nil, err
	}
	return order, nil
}

// MarkPaidByPos 收银结算成功后回写服务工单为已完成/已付款，并刷新工单明细票据。
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
	now := time.Now()
	item.Status = "completed"
	item.PayStatus = "paid"
	item.PosOrderID = posOrder.ID
	item.PosOrderNo = posOrder.OrderNo
	item.PaymentMethod = strings.TrimSpace(posOrder.PaymentMethod)
	if item.PaymentMethod == "" {
		item.PaymentMethod = "pos"
	}
	item.PaidAt = &now
	s.attachServiceReceipt(item, item.Items)
	if err := r.Update(item, nil); err != nil {
		return err
	}
	if item.SalesOrderID > 0 {
		_ = NewSalesService(s.repos, nil).ForTenant(s.tenantID).SyncServiceStatus(item.SalesOrderID, "completed")
	}
	return nil
}

// MarkPaid 线下确认收款（转账截图等），可上传付款截图；含商品时扣减门店库存。
func (s *ServiceOrderService) MarkPaid(id uint64, in *dto.ServiceMarkPaidDTO, userID uint64) (*model.ServiceOrder, error) {
	if in == nil {
		return nil, ErrBadRequest
	}
	r := s.repos.Service.ForTenant(s.tenantID)
	item, err := r.GetByID(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if item.PayStatus == "paid" {
		// 已付款且未完成：允许修改付款方式 / 截图 / 时间
		if item.Status == "completed" || item.Status == "cancelled" {
			return nil, fmt.Errorf("%w：已完成或已取消的工单不可修改付款信息", ErrInvalidStatus)
		}
		method, err := normalizeOfflinePayMethod(in.PaymentMethod)
		if err != nil {
			return nil, err
		}
		proof := strings.TrimSpace(in.PaymentProofURL)
		if method == "transfer" && proof == "" {
			return nil, fmt.Errorf("%w：转账收款请上传付款截图", ErrBadRequest)
		}
		item.PaymentMethod = method
		item.PaymentProofURL = proof
		if t := parseOptionalPaidAt(in.PaidAt); t != nil {
			item.PaidAt = t
		}
		s.attachServiceReceipt(item, item.Items)
		if err := r.Update(item, nil); err != nil {
			return nil, err
		}
		return item, nil
	}
	switch item.Status {
	case "pending", "in_progress", "awaiting_payment":
		// 付款与工单完成状态独立，顾客可先付款
	case "cancelled":
		return nil, fmt.Errorf("%w：已取消工单不可确认收款", ErrInvalidStatus)
	default:
		// completed 等其它状态若未付款也允许补记收款
		if item.Status != "completed" {
			return nil, fmt.Errorf("%w：当前状态不可确认收款", ErrInvalidStatus)
		}
	}

	method, err := normalizeOfflinePayMethod(in.PaymentMethod)
	if err != nil {
		return nil, err
	}
	proof := strings.TrimSpace(in.PaymentProofURL)
	if method == "transfer" && proof == "" {
		return nil, fmt.Errorf("%w：转账收款请上传付款截图", ErrBadRequest)
	}

	// 含商品：校验并扣减门店库存（与收银台结算一致）
	productLines := make([]model.ServiceOrderItem, 0)
	skuIDs := make([]uint64, 0)
	for _, it := range item.Items {
		t := strings.TrimSpace(it.ItemType)
		if t == "" && it.SkuID > 0 {
			t = "product"
		}
		if t == "product" && it.SkuID > 0 {
			productLines = append(productLines, it)
			skuIDs = append(skuIDs, it.SkuID)
		}
	}
	if len(productLines) > 0 {
		qtyMap, err := s.repos.Inventory.ForTenant(s.tenantID).MapQtyBySkuIDs(item.StoreID, skuIDs)
		if err != nil {
			return nil, err
		}
		for _, line := range productLines {
			if qtyMap[line.SkuID] < line.Quantity {
				name := strings.TrimSpace(line.ProductName)
				if name == "" {
					name = line.SkuCode
				}
				if name == "" {
					name = fmt.Sprintf("SKU#%d", line.SkuID)
				}
				return nil, fmt.Errorf("%w（%s），请先调货入库", ErrInsufficientStock, name)
			}
		}
		inv := s.repos.Inventory.ForTenant(s.tenantID)
		for _, line := range productLines {
			if err := inv.AddQuantity(item.StoreID, line.SkuID, line.SkuCode, line.ProductName, line.SpecLabel, line.Pic, -line.Quantity); err != nil {
				return nil, err
			}
		}
	}

	now := time.Now()
	item.PayStatus = "paid"
	item.PaymentMethod = method
	item.PaymentProofURL = proof
	if t := parseOptionalPaidAt(in.PaidAt); t != nil {
		item.PaidAt = t
	} else {
		item.PaidAt = &now
	}
	item.PaidBy = userID
	// 不改动工单 status：付款与完成状态相互独立
	s.attachServiceReceipt(item, item.Items)
	if err := r.Update(item, nil); err != nil {
		return nil, err
	}
	return item, nil
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

func normalizeServiceLineType(line dto.ServiceOrderLineDTO) string {
	t := strings.TrimSpace(line.ItemType)
	switch t {
	case "product", "service":
		return t
	}
	if line.SkuID > 0 {
		return "product"
	}
	return "service"
}

func (s *ServiceOrderService) buildServiceOrder(in *dto.ServiceOrderDTO, userID uint64) (*model.ServiceOrder, []model.ServiceOrderItem, error) {
	if in.StoreID == 0 || len(in.Items) == 0 {
		return nil, nil, ErrBadRequest
	}
	mode := normalizeOrderMode(in.OrderMode)

	serviceIDs := make([]uint64, 0)
	serviceLines := make([]dto.ServiceOrderLineDTO, 0)
	productLines := make([]dto.ServiceOrderLineDTO, 0)
	seenSvc := map[uint64]bool{}
	for _, line := range in.Items {
		itemType := normalizeServiceLineType(line)
		q := line.Quantity
		if q <= 0 {
			q = 1
		}
		switch itemType {
		case "product":
			if line.SkuID == 0 || strings.TrimSpace(line.ProductName) == "" {
				return nil, nil, ErrBadRequest
			}
			line.Quantity = q
			productLines = append(productLines, line)
		default:
			if line.ServiceItemID == 0 {
				return nil, nil, ErrBadRequest
			}
			line.Quantity = q
			serviceLines = append(serviceLines, line)
			if !seenSvc[line.ServiceItemID] {
				seenSvc[line.ServiceItemID] = true
				serviceIDs = append(serviceIDs, line.ServiceItemID)
			}
		}
	}

	byID := map[uint64]model.ServiceItem{}
	if len(serviceIDs) > 0 {
		catalog := s.repos.ServiceCatalog.ForTenant(s.tenantID)
		catalogItems, err := catalog.ListItemsByIDs(serviceIDs)
		if err != nil {
			return nil, nil, err
		}
		if len(catalogItems) != len(serviceIDs) {
			return nil, nil, ErrBadRequest
		}
		for _, it := range catalogItems {
			byID[it.ID] = it
		}
	}

	items := make([]model.ServiceOrderItem, 0, len(serviceLines)+len(productLines))
	estimated := 0.0
	for _, line := range serviceLines {
		src := byID[line.ServiceItemID]
		if src.Status == 0 {
			return nil, nil, ErrBadRequest
		}
		catalogPrice := src.Price
		unit := line.UnitPrice
		if unit <= 0 {
			unit = catalogPrice
		}
		orig := line.OriginalPrice
		if orig <= 0 {
			orig = catalogPrice
		}
		orig, disc, unit := normalizeLinePrices(orig, line.Discount, unit)
		lineTotal := roundMoney(unit * float64(line.Quantity))
		estimated += lineTotal
		items = append(items, model.ServiceOrderItem{
			ItemType:      "service",
			ServiceItemID: src.ID,
			ServiceName:   src.Name,
			ServiceCode:   src.Code,
			Quantity:      line.Quantity,
			OriginalPrice: orig,
			Discount:      disc,
			UnitPrice:     unit,
			TotalAmount:   lineTotal,
			DurationMin:   src.DurationMin,
			Pic:           src.Pic,
		})
	}
	for _, line := range productLines {
		orig, disc, unit := normalizeLinePrices(line.OriginalPrice, line.Discount, line.UnitPrice)
		lineTotal := roundMoney(unit * float64(line.Quantity))
		estimated += lineTotal
		items = append(items, model.ServiceOrderItem{
			ItemType:      "product",
			SkuID:         line.SkuID,
			SkuCode:       strings.TrimSpace(line.SkuCode),
			ProductName:   strings.TrimSpace(line.ProductName),
			SpecLabel:     strings.TrimSpace(line.SpecLabel),
			Quantity:      line.Quantity,
			OriginalPrice: orig,
			Discount:      disc,
			UnitPrice:     unit,
			TotalAmount:   lineTotal,
			Pic:           strings.TrimSpace(line.Pic),
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
	if estimated <= 0 {
		order.PayStatus = "paid"
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

func parseOptionalPaidAt(v string) *time.Time {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil
	}
	t, err := parseFlexibleTime(v)
	if err != nil {
		return nil
	}
	return &t
}

func normalizeOfflinePayMethod(method string) (string, error) {
	method = strings.TrimSpace(method)
	if method == "" {
		method = "transfer"
	}
	switch method {
	case "transfer", "wechat_transfer", "cash", "other":
		if method == "wechat_transfer" {
			method = "transfer"
		}
		return method, nil
	default:
		return "", fmt.Errorf("%w：不支持的付款方式", ErrBadRequest)
	}
}
