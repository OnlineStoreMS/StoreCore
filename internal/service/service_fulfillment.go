package service

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"storecore/internal/model"

	"gorm.io/gorm"
)

func (s *ServiceOrderService) attachServiceReceipt(order *model.ServiceOrder, items []model.ServiceOrderItem) {
	if order == nil {
		return
	}
	if items == nil {
		items = order.Items
	}
	store, _ := s.repos.Store.ForTenant(s.tenantID).GetByID(order.StoreID)
	order.ReceiptHTML = s.buildServiceReceiptHTML(order, items, store, nil)
}

func serviceDocIsPreview(order *model.ServiceOrder) bool {
	if order == nil {
		return true
	}
	// 已付款一律正式单据，不再展示预结算
	if order.PayStatus == "paid" {
		return false
	}
	return order.Status == "pending" || order.Status == "in_progress"
}

func serviceDocBadge(order *model.ServiceOrder) string {
	paid := order.PayStatus == "paid"
	switch order.Status {
	case "pending":
		if paid {
			return "待处理 · 已付款"
		}
		return "待处理"
	case "in_progress":
		if paid {
			return "进行中 · 已付款"
		}
		return "进行中"
	case "awaiting_payment":
		if paid {
			return "待完工 · 已付款"
		}
		return "待付款"
	case "completed":
		if paid {
			return "已完成 · 已付款"
		}
		return "已完成"
	case "cancelled":
		return "已取消"
	default:
		if paid {
			return "已付款"
		}
		return "服务工单"
	}
}

func (s *ServiceOrderService) resolveServiceDocTemplate(storeID uint64) *model.ReceiptTemplate {
	tpl, err := s.repos.ReceiptTpl.ForTenant(s.tenantID).FindDefault(storeID, "service")
	if err != nil || tpl == nil {
		return &model.ReceiptTemplate{
			Name:              "默认服务工单",
			ReceiptType:       "service",
			HeaderTitle:       "服务工单明细",
			HeaderSubtitle:    "正式单据",
			FooterThanks:      "客户签字确认：____________　　经办人：____________　　日期：____________",
			FooterExtra:       "以上金额仅供参考确认，服务完成后到店结算",
			ShowSkuPic:        true,
			ShowStorePhone:    true,
			ShowStoreAddress:  true,
			ShowBusinessHours: true,
			ShowBrandLogo:     true,
			IsDefault:         true,
			Status:            1,
		}
	}
	return tpl
}

type serviceReceiptLine struct {
	OrderNo    string
	DeviceInfo string
	FaultDesc  string
	Item       model.ServiceOrderItem
}

func (s *ServiceOrderService) buildServiceReceiptHTML(order *model.ServiceOrder, items []model.ServiceOrderItem, store *model.Store, extraOrders []model.ServiceOrder) string {
	tpl := s.resolveServiceDocTemplate(order.StoreID)
	storeName := "门店"
	storePhone := ""
	storeAddr := ""
	businessHours := ""
	if store != nil {
		if store.Name != "" {
			storeName = store.Name
		}
		storePhone = store.Phone
		parts := []string{store.Province, store.City, store.District, store.Address}
		var addr []string
		for _, p := range parts {
			if strings.TrimSpace(p) != "" {
				addr = append(addr, strings.TrimSpace(p))
			}
		}
		storeAddr = strings.Join(addr, "")
		businessHours = strings.TrimSpace(store.BusinessHours)
	}

	isMerge := len(extraOrders) > 0
	isPreviewDoc := !isMerge && serviceDocIsPreview(order)
	title := strings.TrimSpace(tpl.HeaderTitle)
	if title == "" {
		title = "服务工单明细"
	}
	badge := strings.TrimSpace(tpl.HeaderSubtitle)
	footer := strings.TrimSpace(tpl.FooterThanks)
	if footer == "" {
		footer = "客户签字确认：____________　　经办人：____________　　日期：____________"
	}
	if isMerge {
		title = "服务工单合并明细"
		badge = "合并打印 · 汇总"
	} else if isPreviewDoc {
		if title == "服务工单明细" {
			title = "服务工单预结算"
		}
		badge = "预结算 · 非正式收款凭证"
		if fe := strings.TrimSpace(tpl.FooterExtra); fe != "" {
			footer = fe
		} else {
			footer = "以上金额仅供参考确认，服务完成后到店结算"
		}
	} else {
		if title == "" || title == "服务工单预结算" || strings.Contains(title, "预结算") {
			title = "服务工单明细"
		}
		if badge == "" || badge == "正式单据" || strings.Contains(badge, "预结算") {
			badge = serviceDocBadge(order)
		}
	}

	createdAt := order.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now()
	}
	createdAtStr := createdAt.Format("2006-01-02 15:04")
	brandLogo := ""
	if store != nil {
		brandLogo = strings.TrimSpace(store.BrandLogo)
	}

	var b strings.Builder
	b.WriteString(`<div class="sales-doc">`)
	b.WriteString(`<div class="sales-doc-head">`)
	b.WriteString(`<div class="sales-doc-brand">`)
	if tpl.ShowBrandLogo && brandLogo != "" {
		b.WriteString(`<div class="sales-doc-logo"><img src="` + htmlEscape(brandLogo) + `" alt="" /></div>`)
	}
	b.WriteString(`<div class="sales-doc-store">` + htmlEscape(storeName) + `</div>`)
	if tpl.ShowStorePhone && storePhone != "" {
		b.WriteString(`<div class="sales-doc-muted">电话 ` + htmlEscape(storePhone) + `</div>`)
	}
	if tpl.ShowStoreAddress && storeAddr != "" {
		b.WriteString(`<div class="sales-doc-muted">地址 ` + htmlEscape(storeAddr) + `</div>`)
	}
	if tpl.ShowBusinessHours && businessHours != "" {
		b.WriteString(`<div class="sales-doc-muted">营业 ` + htmlEscape(businessHours) + `</div>`)
	}
	b.WriteString(`</div>`)
	b.WriteString(`<div class="sales-doc-title-block"><div class="sales-doc-title">` + htmlEscape(title) + `</div>`)
	b.WriteString(`<div class="sales-doc-badge">` + htmlEscape(badge) + `</div></div></div>`)

	b.WriteString(`<table class="sales-doc-info"><tbody>`)
	writeInfoRow := func(k1, v1, k2, v2 string) {
		b.WriteString(`<tr>`)
		b.WriteString(`<th>` + htmlEscape(k1) + `</th><td>` + v1 + `</td>`)
		b.WriteString(`<th>` + htmlEscape(k2) + `</th><td>` + v2 + `</td>`)
		b.WriteString(`</tr>`)
	}
	payLabel := map[string]string{"paid": "已付款", "unpaid": "未付款"}[order.PayStatus]
	if payLabel == "" {
		payLabel = order.PayStatus
	}
	if order.PayStatus == "paid" {
		switch order.PaymentMethod {
		case "transfer", "wechat_transfer":
			payLabel = "已付款 · 转账"
		case "cash":
			payLabel = "已付款 · 现金"
		case "static_qr":
			payLabel = "已付款 · 收款码"
		case "pos":
			payLabel = "已付款 · 收银台"
		case "sales":
			payLabel = "已付款 · 销售单"
		case "other":
			payLabel = "已付款 · 其他"
		}
	}
	if isMerge {
		orderNos := []string{order.OrderNo}
		for _, o := range extraOrders {
			orderNos = append(orderNos, o.OrderNo)
		}
		writeInfoRow("合并工单", htmlEscape(strings.Join(orderNos, "、")), "开单时间", htmlEscape(createdAtStr))
	} else {
		writeInfoRow("工单号", htmlEscape(order.OrderNo), "开单时间", htmlEscape(createdAtStr))
	}
	writeInfoRow("工单状态", htmlEscape(order.Status), "付款状态", htmlEscape(payLabel))
	if order.PayStatus == "paid" {
		paidAtStr := "-"
		if order.PaidAt != nil && !order.PaidAt.IsZero() {
			paidAtStr = order.PaidAt.Format("2006-01-02 15:04:05")
		}
		writeInfoRow("付款时间", htmlEscape(paidAtStr), "顾客姓名", htmlEscape(order.CustomerName))
		writeInfoRow("顾客电话", htmlEscape(order.CustomerPhone), "", "")
	} else {
		writeInfoRow("顾客姓名", htmlEscape(order.CustomerName), "顾客电话", htmlEscape(order.CustomerPhone))
	}
	if !isMerge {
		if order.AppointmentAt != nil {
			writeInfoRow("预约时间", order.AppointmentAt.Format("2006-01-02 15:04"), "工程师", htmlEscape(nz(order.EngineerName, "-")))
		} else if order.EngineerName != "" {
			writeInfoRow("工程师", htmlEscape(order.EngineerName), "", "")
		}
		if order.DeviceInfo != "" || order.FaultDesc != "" {
			writeInfoRow("设备", htmlEscape(nz(order.DeviceInfo, "-")), "说明", htmlEscape(nz(order.FaultDesc, "-")))
		}
		if order.Remark != "" {
			writeInfoRow("备注", htmlEscape(order.Remark), "", "")
		}
	}
	b.WriteString(`</tbody></table>`)

	allOrders := make([]model.ServiceOrder, 0, 1+len(extraOrders))
	allOrders = append(allOrders, *order)
	allOrders = append(allOrders, extraOrders...)

	if isMerge {
		b.WriteString(`<div class="sales-doc-section">设备与说明</div>`)
		b.WriteString(`<table class="sales-doc-table"><thead><tr>`)
		b.WriteString(`<th>设备</th><th>说明</th><th>备注</th><th class="num">金额</th>`)
		b.WriteString(`</tr></thead><tbody>`)
		for _, o := range allOrders {
			b.WriteString(`<tr>`)
			b.WriteString(`<td class="col-name"><div class="name">` + htmlEscape(nz(strings.TrimSpace(o.DeviceInfo), "-")) + `</div>`)
			if o.EngineerName != "" {
				b.WriteString(`<div class="spec">工程师 ` + htmlEscape(o.EngineerName) + `</div>`)
			}
			b.WriteString(`</td>`)
			b.WriteString(`<td>` + htmlEscape(nz(strings.TrimSpace(o.FaultDesc), "-")) + `</td>`)
			b.WriteString(`<td>` + htmlEscape(nz(strings.TrimSpace(o.Remark), "-")) + `</td>`)
			b.WriteString(fmt.Sprintf(`<td class="num strong">%.2f</td>`, o.EstimatedAmount))
			b.WriteString(`</tr>`)
		}
		b.WriteString(`</tbody></table>`)
	}

	lines := make([]serviceReceiptLine, 0)
	appendOrderLines := func(o model.ServiceOrder) {
		its := o.Items
		if o.ID == order.ID && items != nil {
			its = items
		}
		for _, it := range its {
			lines = append(lines, serviceReceiptLine{
				OrderNo:    o.OrderNo,
				DeviceInfo: strings.TrimSpace(o.DeviceInfo),
				FaultDesc:  strings.TrimSpace(o.FaultDesc),
				Item:       it,
			})
		}
	}
	for _, o := range allOrders {
		appendOrderLines(o)
	}

	b.WriteString(`<div class="sales-doc-section">明细清单</div>`)
	b.WriteString(`<table class="sales-doc-table"><thead><tr>`)
	b.WriteString(`<th class="col-idx">#</th>`)
	if tpl.ShowSkuPic {
		b.WriteString(`<th class="col-pic">图</th>`)
	}
	b.WriteString(`<th class="col-name">品名 / 规格</th><th>类型</th><th>编码</th>`)
	b.WriteString(`<th class="num">数量</th><th class="num">原价</th><th class="num">折扣</th><th class="num">单价</th><th class="num">小计</th>`)
	b.WriteString(`</tr></thead><tbody>`)

	colspan := 9
	if tpl.ShowSkuPic {
		colspan++
	}
	total := 0.0
	originalTotal := 0.0
	lineNo := 0
	for _, row := range lines {
		it := row.Item
		itemType := strings.TrimSpace(it.ItemType)
		if itemType == "" {
			if it.SkuID > 0 {
				itemType = "product"
			} else {
				itemType = "service"
			}
		}
		name := it.ServiceName
		code := it.ServiceCode
		spec := ""
		typLabel := "服务"
		if itemType == "product" {
			name = it.ProductName
			code = it.SkuCode
			spec = it.SpecLabel
			typLabel = "商品"
		}
		orig := it.OriginalPrice
		disc := it.Discount
		if orig <= 0 {
			orig = it.UnitPrice
		}
		if disc <= 0 {
			disc = 10
		}
		originalTotal = roundMoney(originalTotal + orig*float64(it.Quantity))
		total = roundMoney(total + it.TotalAmount)
		lineNo++
		b.WriteString(`<tr>`)
		b.WriteString(fmt.Sprintf(`<td class="col-idx">%d</td>`, lineNo))
		if tpl.ShowSkuPic {
			b.WriteString(`<td class="col-pic">`)
			if strings.TrimSpace(it.Pic) != "" {
				b.WriteString(`<img src="` + htmlEscape(it.Pic) + `" alt="" />`)
			} else {
				b.WriteString(`<span class="pic-empty">无图</span>`)
			}
			b.WriteString(`</td>`)
		}
		b.WriteString(`<td class="col-name"><div class="name">` + htmlEscape(name) + `</div>`)
		if strings.TrimSpace(spec) != "" {
			b.WriteString(`<div class="spec">` + htmlEscape(spec) + `</div>`)
		}
		// 合并打印：设备信息已在上方「设备与说明」列出，明细里用小字标注归属即可
		if isMerge {
			device := nz(row.DeviceInfo, "未填设备")
			b.WriteString(`<div class="spec device-tag">设备 ` + htmlEscape(device) + `</div>`)
		}
		b.WriteString(`</td>`)
		b.WriteString(`<td>` + htmlEscape(typLabel) + `</td>`)
		b.WriteString(`<td>` + htmlEscape(nz(code, "-")) + `</td>`)
		b.WriteString(fmt.Sprintf(`<td class="num">%d</td>`, it.Quantity))
		b.WriteString(fmt.Sprintf(`<td class="num">%.2f</td>`, orig))
		b.WriteString(fmt.Sprintf(`<td class="num">%g折</td>`, disc))
		b.WriteString(fmt.Sprintf(`<td class="num">%.2f</td>`, it.UnitPrice))
		b.WriteString(fmt.Sprintf(`<td class="num strong">%.2f</td>`, it.TotalAmount))
		b.WriteString(`</tr>`)
	}
	if len(lines) == 0 {
		b.WriteString(fmt.Sprintf(`<tr><td colspan="%d" class="empty">暂无明细</td></tr>`, colspan))
	}
	b.WriteString(`</tbody></table>`)

	if !isMerge {
		total = order.EstimatedAmount
		// 单工单原价合计按明细重算（兼容旧数据无原价）
		if originalTotal <= 0 {
			originalTotal = total
		}
	} else {
		total = order.EstimatedAmount
		for _, o := range extraOrders {
			total = roundMoney(total + o.EstimatedAmount)
		}
	}
	discountAmount := roundMoney(originalTotal - total)
	if discountAmount < 0 {
		discountAmount = 0
	}

	b.WriteString(`<table class="sales-doc-summary"><tbody>`)
	if originalTotal > total+0.001 {
		b.WriteString(fmt.Sprintf(`<tr><th>原价合计</th><td>¥%.2f</td><th>优惠金额</th><td>¥%.2f</td></tr>`, originalTotal, discountAmount))
	}
	sumLabel := "应付合计"
	if !isPreviewDoc {
		sumLabel = "合计金额"
	}
	b.WriteString(fmt.Sprintf(`<tr class="total"><th colspan="2">%s</th><td colspan="2" class="total-amt">¥%.2f</td></tr>`, sumLabel, total))
	b.WriteString(`</tbody></table>`)
	b.WriteString(`<div class="sales-doc-footer">` + htmlEscape(footer) + `</div>`)
	b.WriteString(`</div>`)
	return b.String()
}

func (s *ServiceOrderService) RefreshReceipt(id uint64) (*model.ServiceOrder, error) {
	r := s.repos.Service.ForTenant(s.tenantID)
	order, err := r.GetByID(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	s.attachServiceReceipt(order, order.Items)
	if err := r.Update(order, nil); err != nil {
		return nil, err
	}
	return order, nil
}

type ServiceMergeReceiptResult struct {
	HTML        string   `json:"html"`
	TotalAmount float64  `json:"totalAmount"`
	OrderNos    []string `json:"orderNos"`
}

func (s *ServiceOrderService) MergeReceipt(ids []uint64) (*ServiceMergeReceiptResult, error) {
	if len(ids) < 2 {
		return nil, fmt.Errorf("%w：请至少选择两个服务工单", ErrBadRequest)
	}
	list, err := s.repos.Service.ForTenant(s.tenantID).GetByIDs(ids)
	if err != nil {
		return nil, err
	}
	if len(list) != len(ids) {
		return nil, fmt.Errorf("%w：部分服务工单不存在", ErrBadRequest)
	}
	base := list[0]
	name := strings.TrimSpace(base.CustomerName)
	phone := strings.TrimSpace(base.CustomerPhone)
	if name == "" || phone == "" {
		return nil, fmt.Errorf("%w：合并打印要求客户姓名与电话均已填写", ErrBadRequest)
	}
	for i := 1; i < len(list); i++ {
		o := list[i]
		if o.StoreID != base.StoreID {
			return nil, fmt.Errorf("%w：仅同门店工单可合并打印", ErrBadRequest)
		}
		if strings.TrimSpace(o.CustomerName) != name || strings.TrimSpace(o.CustomerPhone) != phone {
			return nil, fmt.Errorf("%w：仅同客户姓名与电话的工单可合并打印", ErrBadRequest)
		}
	}
	store, _ := s.repos.Store.ForTenant(s.tenantID).GetByID(base.StoreID)
	extra := list[1:]
	html := s.buildServiceReceiptHTML(&base, base.Items, store, extra)
	total := base.EstimatedAmount
	orderNos := make([]string, 0, len(list))
	orderNos = append(orderNos, base.OrderNo)
	for _, o := range extra {
		total = roundMoney(total + o.EstimatedAmount)
		orderNos = append(orderNos, o.OrderNo)
	}
	return &ServiceMergeReceiptResult{
		HTML:        html,
		TotalAmount: total,
		OrderNos:    orderNos,
	}, nil
}
