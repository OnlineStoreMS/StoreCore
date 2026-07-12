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

func (s *SalesService) buildSalesReceiptHTML(order *model.StoreSalesOrder, items []model.StoreSalesOrderItem, serviceItems []model.StoreSalesOrderServiceItem, store *model.Store, preview bool) string {
	storeName := "门店"
	storePhone := ""
	storeAddr := ""
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
	}

	title := "销售单"
	subtitle := "欢迎光临"
	footer := "谢谢惠顾"
	if preview {
		title = "销售预结算单"
		subtitle = "仅供确认明细与金额，非正式收款凭证"
		footer = "请确认以上信息后到店办理"
	}

	var b strings.Builder
	b.WriteString(`<div class="receipt-doc">`)
	b.WriteString(`<div class="receipt-header"><div class="receipt-title">` + htmlEscape(title) + `</div>`)
	b.WriteString(`<div class="receipt-sub">` + htmlEscape(subtitle) + `</div></div>`)
	b.WriteString(`<div class="receipt-store">` + htmlEscape(storeName) + `</div>`)
	if storePhone != "" {
		b.WriteString(`<div class="receipt-meta">电话：` + htmlEscape(storePhone) + `</div>`)
	}
	if storeAddr != "" {
		b.WriteString(`<div class="receipt-meta">地址：` + htmlEscape(storeAddr) + `</div>`)
	}
	b.WriteString(`<div class="receipt-meta">单号：` + htmlEscape(order.OrderNo) + `</div>`)
	b.WriteString(`<div class="receipt-meta">履约：` + htmlEscape(fulfillmentLabel(order.FulfillmentType)) + `</div>`)
	if order.CustomerName != "" || order.CustomerPhone != "" {
		b.WriteString(`<div class="receipt-meta">顾客：` + htmlEscape(strings.TrimSpace(order.CustomerName+" "+order.CustomerPhone)) + `</div>`)
	}
	if order.AppointmentAt != nil {
		b.WriteString(`<div class="receipt-meta">预约：` + order.AppointmentAt.Format("2006-01-02 15:04") + `</div>`)
	}
	if order.PickupPersonName != "" {
		b.WriteString(`<div class="receipt-meta">取件人：` + htmlEscape(strings.TrimSpace(order.PickupPersonName+" "+order.PickupPersonPhone)) + `</div>`)
	}
	if order.FulfillmentType == "delivery" {
		if order.DeliveryType != "" {
			b.WriteString(`<div class="receipt-meta">配送：` + htmlEscape(deliveryTypeLabel(order.DeliveryType)) + `</div>`)
		}
		if order.ExpectedDeliveryAt != nil {
			b.WriteString(`<div class="receipt-meta">期望配送：` + order.ExpectedDeliveryAt.Format("2006-01-02 15:04") + `</div>`)
		}
	}
	if order.ShippingAddress != "" {
		recv := strings.TrimSpace(order.ReceiverName + " " + order.ReceiverPhone)
		if recv != "" {
			b.WriteString(`<div class="receipt-meta">收货人：` + htmlEscape(recv) + `</div>`)
		}
		b.WriteString(`<div class="receipt-meta">地址：` + htmlEscape(order.ShippingAddress) + `</div>`)
	}
	if order.ExpressScheduledAt != nil {
		b.WriteString(`<div class="receipt-meta">预约快递：` + order.ExpressScheduledAt.Format("2006-01-02 15:04") + `</div>`)
	}
	b.WriteString(`<div class="receipt-divider"></div>`)
	b.WriteString(`<div class="receipt-section">商品</div>`)
	for _, it := range items {
		b.WriteString(`<div class="receipt-line"><div class="receipt-line-name">` + htmlEscape(it.ProductName))
		if it.SpecLabel != "" {
			b.WriteString(` <span class="receipt-spec">` + htmlEscape(it.SpecLabel) + `</span>`)
		}
		b.WriteString(`</div>`)
		b.WriteString(fmt.Sprintf(`<div class="receipt-line-amt">x%d  ¥%.2f</div></div>`, it.Quantity, it.TotalAmount))
		if it.OriginalPrice > 0 && it.UnitPrice < it.OriginalPrice {
			b.WriteString(fmt.Sprintf(`<div class="receipt-line-sub">原价¥%.2f · %g折 · 实付¥%.2f</div>`, it.OriginalPrice, it.Discount, it.UnitPrice))
		}
	}
	if len(serviceItems) > 0 {
		b.WriteString(`<div class="receipt-divider"></div>`)
		b.WriteString(`<div class="receipt-section">服务</div>`)
		for _, it := range serviceItems {
			b.WriteString(`<div class="receipt-line"><div class="receipt-line-name">` + htmlEscape(it.ServiceName) + `</div>`)
			b.WriteString(fmt.Sprintf(`<div class="receipt-line-amt">x%d  ¥%.2f</div></div>`, it.Quantity, it.TotalAmount))
		}
	}
	b.WriteString(`<div class="receipt-divider"></div>`)
	if order.OriginalAmount > 0 && order.DiscountAmount > 0 {
		b.WriteString(fmt.Sprintf(`<div class="receipt-sum"><span>原价合计</span><span>¥%.2f</span></div>`, order.OriginalAmount))
		b.WriteString(fmt.Sprintf(`<div class="receipt-sum"><span>优惠</span><span>-¥%.2f</span></div>`, order.DiscountAmount))
	}
	sumLabel := "应付合计"
	if !preview {
		sumLabel = "订单合计"
	}
	b.WriteString(fmt.Sprintf(`<div class="receipt-total"><span>%s</span><span>¥%.2f</span></div>`, sumLabel, order.TotalAmount))
	if order.Remark != "" {
		b.WriteString(`<div class="receipt-meta">备注：` + htmlEscape(order.Remark) + `</div>`)
	}
	b.WriteString(`<div class="receipt-footer">` + htmlEscape(footer) + `</div>`)
	b.WriteString(`</div>`)
	return b.String()
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
