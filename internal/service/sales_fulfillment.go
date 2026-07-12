package service

import (
	"fmt"
	"strings"
	"time"

	"storecore/internal/dto"
	"storecore/internal/model"
)

func normalizeFulfillmentType(v string) string {
	switch strings.TrimSpace(v) {
	case "install", "delivery", "express":
		return v
	default:
		return "pickup"
	}
}

func normalizeDeliveryType(v string) string {
	switch strings.TrimSpace(v) {
	case "huolala", "errand", "store_delivery":
		return v
	default:
		return ""
	}
}

func parseOptionalTime(p *string) (*time.Time, error) {
	if p == nil || strings.TrimSpace(*p) == "" {
		return nil, nil
	}
	t, err := parseFlexibleTime(*p)
	if err != nil {
		return nil, ErrBadRequest
	}
	return &t, nil
}

func (s *SalesService) applySalesDTO(order *model.StoreSalesOrder, in *dto.StoreSalesOrderDTO) error {
	ft := normalizeFulfillmentType(in.FulfillmentType)
	order.FulfillmentType = ft
	order.CustomerName = strings.TrimSpace(in.CustomerName)
	order.CustomerPhone = strings.TrimSpace(in.CustomerPhone)
	order.NeedProcurement = in.NeedProcurement
	order.Remark = strings.TrimSpace(in.Remark)

	appointmentAt, err := parseOptionalTime(in.AppointmentAt)
	if err != nil {
		return err
	}
	expectedDeliveryAt, err := parseOptionalTime(in.ExpectedDeliveryAt)
	if err != nil {
		return err
	}
	expressScheduledAt, err := parseOptionalTime(in.ExpressScheduledAt)
	if err != nil {
		return err
	}

	order.AppointmentAt = nil
	order.PickupPersonName = ""
	order.PickupPersonPhone = ""
	order.PickupCode = ""
	order.DeliveryType = ""
	order.ExpectedDeliveryAt = nil
	order.ReceiverName = ""
	order.ReceiverPhone = ""
	order.ShippingAddress = ""
	order.ExpressCompany = ""
	order.ExpressNo = ""
	order.ExpressScheduledAt = nil

	switch ft {
	case "pickup":
		order.AppointmentAt = appointmentAt
		order.PickupPersonName = strings.TrimSpace(in.PickupPersonName)
		if order.PickupPersonName == "" {
			order.PickupPersonName = order.CustomerName
		}
		order.PickupPersonPhone = strings.TrimSpace(in.PickupPersonPhone)
		if order.PickupPersonPhone == "" {
			order.PickupPersonPhone = order.CustomerPhone
		}
		order.PickupCode = strings.TrimSpace(in.PickupCode)
	case "install":
		order.AppointmentAt = appointmentAt
		order.PickupPersonName = strings.TrimSpace(in.PickupPersonName)
		order.PickupPersonPhone = strings.TrimSpace(in.PickupPersonPhone)
		if order.PickupPersonName == "" {
			order.PickupPersonName = order.CustomerName
		}
		if order.PickupPersonPhone == "" {
			order.PickupPersonPhone = order.CustomerPhone
		}
	case "delivery":
		order.DeliveryType = normalizeDeliveryType(in.DeliveryType)
		if order.DeliveryType == "" {
			order.DeliveryType = "store_delivery"
		}
		order.ExpectedDeliveryAt = expectedDeliveryAt
		order.ReceiverName = strings.TrimSpace(in.ReceiverName)
		if order.ReceiverName == "" {
			order.ReceiverName = order.CustomerName
		}
		order.ReceiverPhone = strings.TrimSpace(in.ReceiverPhone)
		if order.ReceiverPhone == "" {
			order.ReceiverPhone = order.CustomerPhone
		}
		order.ShippingAddress = strings.TrimSpace(in.ShippingAddress)
	case "express":
		order.ReceiverName = strings.TrimSpace(in.ReceiverName)
		if order.ReceiverName == "" {
			order.ReceiverName = order.CustomerName
		}
		order.ReceiverPhone = strings.TrimSpace(in.ReceiverPhone)
		if order.ReceiverPhone == "" {
			order.ReceiverPhone = order.CustomerPhone
		}
		order.ShippingAddress = strings.TrimSpace(in.ShippingAddress)
		order.ExpressCompany = strings.TrimSpace(in.ExpressCompany)
		order.ExpressNo = strings.TrimSpace(in.ExpressNo)
		order.ExpressScheduledAt = expressScheduledAt
	}

	if order.NeedProcurement {
		if order.PurchaseStatus == "" || order.PurchaseStatus == "none" {
			order.PurchaseStatus = "pending"
		}
	} else {
		order.PurchaseStatus = "none"
		order.PurchaseOrderID = 0
	}

	switch ft {
	case "install":
		if order.ServiceStatus == "" || order.ServiceStatus == "none" {
			order.ServiceStatus = "pending"
		}
	default:
		if order.ServiceOrderID == 0 {
			order.ServiceStatus = "none"
		}
	}
	return nil
}

func initSubStatusesOnConfirm(order *model.StoreSalesOrder) {
	if order.NeedProcurement {
		if order.PurchaseStatus == "none" || order.PurchaseStatus == "" {
			order.PurchaseStatus = "pending"
		}
	} else {
		order.PurchaseStatus = "none"
	}
	switch order.FulfillmentType {
	case "pickup":
		order.FulfillStatus = "awaiting_pickup"
		order.ServiceStatus = "none"
	case "install":
		order.FulfillStatus = "awaiting_pickup"
		if order.ServiceStatus == "none" || order.ServiceStatus == "" {
			order.ServiceStatus = "pending"
		}
	case "delivery":
		order.FulfillStatus = "awaiting_delivery"
		order.ServiceStatus = "none"
	case "express":
		order.FulfillStatus = "awaiting_express"
		order.ServiceStatus = "none"
	default:
		order.FulfillStatus = "none"
	}
}

func fulfillmentLabel(ft string) string {
	switch ft {
	case "pickup":
		return "到店提货"
	case "install":
		return "到店安装"
	case "delivery":
		return "送货上门"
	case "express":
		return "发快递"
	default:
		return ft
	}
}

func deliveryTypeLabel(v string) string {
	switch v {
	case "huolala":
		return "货拉拉"
	case "errand":
		return "跑腿"
	case "store_delivery":
		return "门店送货"
	default:
		return v
	}
}

func (s *SalesService) resolveSalesTemplate(storeID uint64) *model.ReceiptTemplate {
	tpl, err := s.repos.ReceiptTpl.ForTenant(s.tenantID).FindDefault(storeID, "sales")
	if err != nil || tpl == nil {
		return &model.ReceiptTemplate{
			Name:              "默认销售单",
			ReceiptType:       "sales",
			HeaderTitle:       "销售单据",
			HeaderSubtitle:    "正式单据",
			FooterThanks:      "客户签字确认：____________　　经办人：____________　　日期：____________",
			FooterExtra:       "",
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

func (s *SalesService) buildSalesReceiptHTML(order *model.StoreSalesOrder, items []model.StoreSalesOrderItem, serviceItems []model.StoreSalesOrderServiceItem, store *model.Store, preview bool) string {
	tpl := s.resolveSalesTemplate(order.StoreID)
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

	title := strings.TrimSpace(tpl.HeaderTitle)
	if title == "" {
		title = "销售单据"
	}
	badge := strings.TrimSpace(tpl.HeaderSubtitle)
	if badge == "" {
		badge = "正式单据"
	}
	footer := strings.TrimSpace(tpl.FooterThanks)
	if footer == "" {
		footer = "客户签字确认：____________　　经办人：____________　　日期：____________"
	}
	if preview {
		if strings.TrimSpace(tpl.HeaderTitle) == "" || tpl.HeaderTitle == "销售单据" {
			title = "销售预结算单"
		}
		badge = "预结算 · 非正式收款凭证"
		if strings.TrimSpace(tpl.FooterExtra) != "" {
			footer = strings.TrimSpace(tpl.FooterExtra)
		} else {
			footer = "以上金额仅供参考确认，请核对明细后到店办理"
		}
	}

	createdAt := order.CreatedAt.Format("2006-01-02 15:04")
	var b strings.Builder
	b.WriteString(`<div class="sales-doc">`)
	b.WriteString(`<div class="sales-doc-head">`)
	b.WriteString(`<div class="sales-doc-brand"><div class="sales-doc-store">` + htmlEscape(storeName) + `</div>`)
	if tpl.ShowStorePhone && storePhone != "" {
		b.WriteString(`<div class="sales-doc-muted">电话 ` + htmlEscape(storePhone) + `</div>`)
	}
	if tpl.ShowStoreAddress && storeAddr != "" {
		b.WriteString(`<div class="sales-doc-muted">地址 ` + htmlEscape(storeAddr) + `</div>`)
	}
	if tpl.ShowBusinessHours && businessHours != "" {
		b.WriteString(`<div class="sales-doc-muted">营业 ` + htmlEscape(businessHours) + `</div>`)
	}
	if strings.TrimSpace(tpl.HeaderExtra) != "" {
		b.WriteString(`<div class="sales-doc-muted">` + htmlEscape(tpl.HeaderExtra) + `</div>`)
	}
	b.WriteString(`</div>`)
	b.WriteString(`<div class="sales-doc-title-block"><div class="sales-doc-title">` + htmlEscape(title) + `</div>`)
	b.WriteString(`<div class="sales-doc-badge">` + htmlEscape(badge) + `</div></div></div>`)

	// 订单信息两列表格
	b.WriteString(`<table class="sales-doc-info"><tbody>`)
	writeInfoRow := func(k1, v1, k2, v2 string) {
		b.WriteString(`<tr>`)
		b.WriteString(`<th>` + htmlEscape(k1) + `</th><td>` + v1 + `</td>`)
		b.WriteString(`<th>` + htmlEscape(k2) + `</th><td>` + v2 + `</td>`)
		b.WriteString(`</tr>`)
	}
	writeInfoRow("销售单号", htmlEscape(order.OrderNo), "开单时间", htmlEscape(createdAt))
	writeInfoRow("履约方式", htmlEscape(fulfillmentLabel(order.FulfillmentType)), "需采购", map[bool]string{true: "是", false: "否"}[order.NeedProcurement])
	writeInfoRow("顾客姓名", htmlEscape(order.CustomerName), "顾客电话", htmlEscape(order.CustomerPhone))
	if order.AppointmentAt != nil {
		writeInfoRow("预约时间", order.AppointmentAt.Format("2006-01-02 15:04"), "取件人", htmlEscape(strings.TrimSpace(order.PickupPersonName+" "+order.PickupPersonPhone)))
	} else if order.PickupPersonName != "" {
		writeInfoRow("取件人", htmlEscape(order.PickupPersonName), "取件电话", htmlEscape(order.PickupPersonPhone))
	}
	if order.PickupCode != "" {
		writeInfoRow("取件码", htmlEscape(order.PickupCode), "", "")
	}
	if order.FulfillmentType == "delivery" {
		writeInfoRow("配送类型", htmlEscape(deliveryTypeLabel(order.DeliveryType)), "期望配送", func() string {
			if order.ExpectedDeliveryAt != nil {
				return order.ExpectedDeliveryAt.Format("2006-01-02 15:04")
			}
			return "-"
		}())
	}
	if order.ShippingAddress != "" {
		writeInfoRow("收货人", htmlEscape(strings.TrimSpace(order.ReceiverName+" "+order.ReceiverPhone)), "收货地址", htmlEscape(order.ShippingAddress))
	}
	if order.FulfillmentType == "express" {
		expAt := "-"
		if order.ExpressScheduledAt != nil {
			expAt = order.ExpressScheduledAt.Format("2006-01-02 15:04")
		}
		writeInfoRow("快递公司", htmlEscape(nz(order.ExpressCompany, "-")), "预约快递", expAt)
		if order.ExpressNo != "" {
			writeInfoRow("运单号", htmlEscape(order.ExpressNo), "", "")
		}
	}
	if order.ServiceOrderNo != "" {
		writeInfoRow("服务工单", htmlEscape(order.ServiceOrderNo), "备注", htmlEscape(nz(order.Remark, "-")))
	} else if order.Remark != "" {
		writeInfoRow("备注", htmlEscape(order.Remark), "", "")
	}
	b.WriteString(`</tbody></table>`)

	// 明细横向表格
	b.WriteString(`<div class="sales-doc-section">明细清单</div>`)
	b.WriteString(`<table class="sales-doc-table"><thead><tr>`)
	b.WriteString(`<th class="col-idx">#</th>`)
	if tpl.ShowSkuPic {
		b.WriteString(`<th class="col-pic">图</th>`)
	}
	b.WriteString(`<th class="col-name">品名 / 规格</th><th>类型</th><th>编码</th>`)
	b.WriteString(`<th class="num">数量</th><th class="num">原价</th><th class="num">折扣</th><th class="num">单价</th><th class="num">小计</th>`)
	b.WriteString(`</tr></thead><tbody>`)

	idx := 0
	colspan := 9
	if tpl.ShowSkuPic {
		colspan = 10
	}
	appendLine := func(pic, name, spec, typ, code string, qty int, orig, disc, unit, total float64) {
		idx++
		b.WriteString(`<tr>`)
		b.WriteString(fmt.Sprintf(`<td class="col-idx">%d</td>`, idx))
		if tpl.ShowSkuPic {
			b.WriteString(`<td class="col-pic">`)
			if strings.TrimSpace(pic) != "" {
				b.WriteString(`<img src="` + htmlEscape(pic) + `" alt="" />`)
			} else {
				b.WriteString(`<span class="pic-empty">无图</span>`)
			}
			b.WriteString(`</td>`)
		}
		b.WriteString(`<td class="col-name"><div class="name">` + htmlEscape(name) + `</div>`)
		if strings.TrimSpace(spec) != "" {
			b.WriteString(`<div class="spec">` + htmlEscape(spec) + `</div>`)
		}
		b.WriteString(`</td>`)
		b.WriteString(`<td>` + htmlEscape(typ) + `</td>`)
		b.WriteString(`<td>` + htmlEscape(nz(code, "-")) + `</td>`)
		b.WriteString(fmt.Sprintf(`<td class="num">%d</td>`, qty))
		b.WriteString(fmt.Sprintf(`<td class="num">%.2f</td>`, orig))
		b.WriteString(fmt.Sprintf(`<td class="num">%g折</td>`, disc))
		b.WriteString(fmt.Sprintf(`<td class="num">%.2f</td>`, unit))
		b.WriteString(fmt.Sprintf(`<td class="num strong">%.2f</td>`, total))
		b.WriteString(`</tr>`)
	}

	for _, it := range items {
		appendLine(it.Pic, it.ProductName, it.SpecLabel, "商品", it.SkuCode, it.Quantity, it.OriginalPrice, it.Discount, it.UnitPrice, it.TotalAmount)
	}
	for _, it := range serviceItems {
		appendLine(it.Pic, it.ServiceName, "", "服务", it.ServiceCode, it.Quantity, it.OriginalPrice, it.Discount, it.UnitPrice, it.TotalAmount)
	}
	if idx == 0 {
		b.WriteString(fmt.Sprintf(`<tr><td colspan="%d" class="empty">暂无明细</td></tr>`, colspan))
	}
	b.WriteString(`</tbody></table>`)

	// 汇总
	b.WriteString(`<table class="sales-doc-summary"><tbody>`)
	b.WriteString(fmt.Sprintf(`<tr><th>原价合计</th><td>¥%.2f</td><th>优惠金额</th><td>¥%.2f</td></tr>`, order.OriginalAmount, order.DiscountAmount))
	sumLabel := "应付合计"
	if !preview {
		sumLabel = "订单合计"
	}
	b.WriteString(fmt.Sprintf(`<tr class="total"><th colspan="2">%s</th><td colspan="2" class="total-amt">¥%.2f</td></tr>`, sumLabel, order.TotalAmount))
	b.WriteString(`</tbody></table>`)

	b.WriteString(`<div class="sales-doc-footer">` + htmlEscape(footer) + `</div>`)
	if !preview && strings.TrimSpace(tpl.FooterExtra) != "" {
		b.WriteString(`<div class="sales-doc-footer extra">` + htmlEscape(tpl.FooterExtra) + `</div>`)
	}
	b.WriteString(`</div>`)
	return b.String()
}

func nz(v, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return v
}

func htmlEscape(s string) string {
	r := strings.NewReplacer(
		`&`, "&amp;",
		`<`, "&lt;",
		`>`, "&gt;",
		`"`, "&quot;",
	)
	return r.Replace(s)
}

func (s *SalesService) attachReceipt(order *model.StoreSalesOrder, items []model.StoreSalesOrderItem, serviceItems []model.StoreSalesOrderServiceItem, preview bool) {
	store, _ := s.repos.Store.ForTenant(s.tenantID).GetByID(order.StoreID)
	order.ReceiptHTML = s.buildSalesReceiptHTML(order, items, serviceItems, store, preview)
}
