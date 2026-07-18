package service

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"storecore/internal/dto"
	"storecore/internal/model"
	"storecore/internal/repo"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PosService struct {
	repos    *repo.Repos
	tenantID uint64
}

func NewPosService(repos *repo.Repos) *PosService {
	return &PosService{repos: repos}
}

func (s *PosService) ForTenant(tenantID uint64) *PosService {
	return &PosService{repos: s.repos, tenantID: repo.NormalizeTenantID(tenantID)}
}

func (s *PosService) List(storeID uint64, page, pageSize int) ([]model.PosOrder, int64, error) {
	return s.repos.Pos.ForTenant(s.tenantID).List(storeID, page, pageSize)
}

func (s *PosService) Get(id uint64) (*model.PosOrder, error) {
	item, err := s.repos.Pos.ForTenant(s.tenantID).GetByID(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return item, err
}

func (s *PosService) Delete(id uint64) error {
	r := s.repos.Pos.ForTenant(s.tenantID)
	order, err := r.GetByID(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound
	} else if err != nil {
		return err
	}
	// 断开服务工单上的收银关联，避免留下脏指针
	if order.ServiceOrderID > 0 {
		sr := s.repos.Service.ForTenant(s.tenantID)
		if so, err := sr.GetByID(order.ServiceOrderID); err == nil && so != nil && so.PosOrderID == id {
			so.PosOrderID = 0
			so.PosOrderNo = ""
			_ = sr.Update(so, nil)
		}
	}
	if err := r.Delete(id); errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound
	} else if err != nil {
		return err
	}
	return nil
}

func (s *PosService) Create(in *dto.PosOrderDTO, cashierUserID uint64) (*model.PosOrder, error) {
	if in.StoreID == 0 || len(in.Items) == 0 {
		return nil, ErrBadRequest
	}
	if !in.IsPreview && strings.TrimSpace(in.PaymentMethod) == "" {
		return nil, ErrBadRequest
	}
	store, err := s.repos.Store.ForTenant(s.tenantID).GetByID(in.StoreID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrBadRequest
	}
	if err != nil {
		return nil, err
	}

	var linkedService *model.ServiceOrder
	if in.ServiceOrderID > 0 && !in.IsPreview {
		so, err := s.repos.Service.ForTenant(s.tenantID).GetByID(in.ServiceOrderID)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrBadRequest
		}
		if err != nil {
			return nil, err
		}
		if so.StoreID != in.StoreID {
			return nil, ErrBadRequest
		}
		if so.Status != "awaiting_payment" && so.Status != "in_progress" {
			return nil, ErrInvalidStatus
		}
		if so.PosOrderID > 0 || so.PayStatus == "paid" {
			return nil, ErrInvalidStatus
		}
		linkedService = so
	}

	originalTotal := 0.0
	payableTotal := 0.0
	items := make([]model.PosOrderItem, 0, len(in.Items))
	productSkuIDs := make([]uint64, 0)
	for _, line := range in.Items {
		if normalizePosItemType(line.ItemType) == "product" && line.SkuID > 0 {
			productSkuIDs = append(productSkuIDs, line.SkuID)
		}
	}
	storeQty := map[uint64]int{}
	if !in.IsPreview && len(productSkuIDs) > 0 {
		m, err := s.repos.Inventory.ForTenant(s.tenantID).MapQtyBySkuIDs(in.StoreID, productSkuIDs)
		if err != nil {
			return nil, err
		}
		storeQty = m
	}
	for _, line := range in.Items {
		itemType := normalizePosItemType(line.ItemType)
		if line.Quantity <= 0 || strings.TrimSpace(line.ProductName) == "" {
			return nil, ErrBadRequest
		}
		if itemType == "product" && line.SkuID == 0 {
			return nil, ErrBadRequest
		}
		if itemType == "product" && !in.IsPreview {
			if storeQty[line.SkuID] < line.Quantity {
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
		if itemType == "service" && line.ServiceItemID == 0 {
			return nil, ErrBadRequest
		}
		orig, disc, unit := normalizeLinePrices(line.OriginalPrice, line.Discount, line.UnitPrice)
		lineOrigTotal := roundMoney(orig * float64(line.Quantity))
		lineTotal := roundMoney(unit * float64(line.Quantity))
		originalTotal += lineOrigTotal
		payableTotal += lineTotal
		items = append(items, model.PosOrderItem{
			ItemType: itemType, SkuID: line.SkuID, ServiceItemID: line.ServiceItemID,
			ProductName: line.ProductName, SkuCode: line.SkuCode,
			SpecLabel: line.SpecLabel, Pic: strings.TrimSpace(line.Pic),
			Quantity: line.Quantity, OriginalPrice: orig, Discount: disc,
			UnitPrice: unit, TotalAmount: lineTotal,
		})
	}
	originalTotal = roundMoney(originalTotal)
	payableTotal = roundMoney(payableTotal)
	discountTotal := roundMoney(originalTotal - payableTotal)
	if discountTotal < 0 {
		discountTotal = 0
	}

	payStatus := "unpaid"
	status := "pending"
	paidAmount := 0.0
	paymentMethod := strings.TrimSpace(in.PaymentMethod)
	orderPrefix := "POS"
	if in.IsPreview {
		status = "preview"
		payStatus = "unpaid"
		paymentMethod = "preview"
		orderPrefix = "PRE"
	} else if paymentMethod == "cash" || paymentMethod == "static_qr" {
		payStatus = "paid"
		status = "completed"
		paidAmount = payableTotal
	}

	now := time.Now()
	order := &model.PosOrder{
		StoreID:        in.StoreID,
		OrderNo:        genOrderNo(orderPrefix),
		Status:         status,
		PaymentMethod:  paymentMethod,
		PayStatus:      payStatus,
		OriginalAmount: originalTotal,
		DiscountAmount: discountTotal,
		TotalAmount:    payableTotal,
		PaidAmount:     paidAmount,
		CustomerName:   in.CustomerName,
		CustomerPhone:  in.CustomerPhone,
		CashierUserID:  cashierUserID,
		ReceiptType:    defaultReceiptType(in.ReceiptType),
		Remark:         in.Remark,
	}
	if linkedService != nil {
		order.ServiceOrderID = linkedService.ID
		order.ServiceOrderNo = linkedService.OrderNo
		if strings.TrimSpace(order.CustomerName) == "" {
			order.CustomerName = linkedService.CustomerName
		}
		if strings.TrimSpace(order.CustomerPhone) == "" {
			order.CustomerPhone = linkedService.CustomerPhone
		}
	}
	if payStatus == "paid" || in.IsPreview {
		if payStatus == "paid" {
			order.PaidAt = &now
		}
		order.ReceiptHTML = s.buildReceiptHTML(order, items, store)
	}
	if err := s.repos.Pos.ForTenant(s.tenantID).Create(order, items); err != nil {
		return nil, err
	}
	if linkedService != nil {
		if payStatus == "paid" {
			_ = s.syncServiceOrderPaid(linkedService.ID, order)
		} else {
			_ = s.linkServiceOrderPos(linkedService.ID, order)
		}
	}
	if payStatus == "paid" {
		inv := s.repos.Inventory.ForTenant(s.tenantID)
		for _, line := range items {
			if line.ItemType == "service" || line.SkuID == 0 {
				continue
			}
			_ = inv.AddQuantity(in.StoreID, line.SkuID, line.SkuCode, line.ProductName, line.SpecLabel, line.Pic, -line.Quantity)
		}
	}
	return order, nil
}

func (s *PosService) syncServiceOrderPaid(serviceOrderID uint64, posOrder *model.PosOrder) error {
	svc := NewServiceOrderService(s.repos).ForTenant(s.tenantID)
	return svc.MarkPaidByPos(serviceOrderID, posOrder)
}

func (s *PosService) linkServiceOrderPos(serviceOrderID uint64, posOrder *model.PosOrder) error {
	svc := NewServiceOrderService(s.repos).ForTenant(s.tenantID)
	return svc.LinkPosOrder(serviceOrderID, posOrder)
}

func (s *PosService) MarkPaid(id uint64) (*model.PosOrder, error) {
	r := s.repos.Pos.ForTenant(s.tenantID)
	order, err := r.GetByID(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if order.PayStatus == "paid" {
		return order, nil
	}
	store, err := s.repos.Store.ForTenant(s.tenantID).GetByID(order.StoreID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	now := time.Now()
	order.PayStatus = "paid"
	order.Status = "completed"
	order.PaidAmount = order.TotalAmount
	order.PaidAt = &now
	order.ReceiptHTML = s.buildReceiptHTML(order, order.Items, store)
	if err := r.Update(order); err != nil {
		return nil, err
	}
	inv := s.repos.Inventory.ForTenant(s.tenantID)
	for _, line := range order.Items {
		if line.ItemType == "service" || line.SkuID == 0 {
			continue
		}
		_ = inv.AddQuantity(order.StoreID, line.SkuID, line.SkuCode, line.ProductName, line.SpecLabel, line.Pic, -line.Quantity)
	}
	if order.ServiceOrderID > 0 {
		_ = s.syncServiceOrderPaid(order.ServiceOrderID, order)
	}
	return order, nil
}

func genOrderNo(prefix string) string {
	return fmt.Sprintf("%s-%s-%s", prefix, time.Now().Format("20060102"), uuid.New().String()[:8])
}

func roundMoney(v float64) float64 {
	return float64(int64(v*100+0.5)) / 100
}

// normalizeLinePrices 统一原价/折扣/实付价。折扣单位为「折」：10=原价，8=八折。
// 实付价以 unit 为准；若未传折扣则按原价推算。
func normalizeLinePrices(original, discount, unit float64) (orig, disc, final float64) {
	final = unit
	if final < 0 {
		final = 0
	}
	orig = original
	if orig <= 0 {
		orig = final
	}
	if orig < 0 {
		orig = 0
	}
	if discount > 0 {
		disc = discount
		if disc > 10 {
			disc = 10
		}
	} else if orig > 0 {
		disc = roundMoney(final / orig * 10)
		if disc <= 0 {
			disc = 10
		}
		if disc > 10 {
			disc = 10
		}
	} else {
		disc = 10
	}
	return roundMoney(orig), roundMoney(disc), roundMoney(final)
}

func normalizePosItemType(t string) string {
	if strings.TrimSpace(t) == "service" {
		return "service"
	}
	return "product"
}

func defaultReceiptType(t string) string {
	switch strings.TrimSpace(t) {
	case "large":
		return "large"
	case "sales":
		return "sales"
	case "service":
		return "service"
	default:
		return "small"
	}
}

func (s *PosService) resolveTemplate(storeID uint64, receiptType string) *model.ReceiptTemplate {
	tpl, err := s.repos.ReceiptTpl.ForTenant(s.tenantID).FindDefault(storeID, receiptType)
	if err != nil || tpl == nil {
		return defaultReceiptTemplate()
	}
	return tpl
}

func paymentMethodLabel(method string) string {
	switch method {
	case "cash":
		return "现金"
	case "static_qr":
		return "静态二维码"
	case "wechat":
		return "微信支付"
	case "alipay":
		return "支付宝"
	case "card":
		return "银行卡"
	case "mixed":
		return "组合支付"
	case "preview":
		return "预结算（未收款）"
	default:
		if method == "" {
			return "-"
		}
		return method
	}
}

func (s *PosService) buildReceiptHTML(order *model.PosOrder, items []model.PosOrderItem, store *model.Store) string {
	tpl := s.resolveTemplate(order.StoreID, order.ReceiptType)
	storeName := ""
	storePhone := ""
	storeAddr := ""
	businessHours := ""
	brandLogo := ""
	coverPic := ""
	guideText := ""
	mapLabel := ""
	if store != nil {
		storeName = store.Name
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
		brandLogo = strings.TrimSpace(store.BrandLogo)
		coverPic = strings.TrimSpace(store.CoverPic)
		guideText = strings.TrimSpace(store.GuideText)
		mapLabel = strings.TrimSpace(store.MapLabel)
	}

	headerTitle := tpl.HeaderTitle
	if headerTitle == "" {
		if storeName != "" {
			headerTitle = storeName
		} else {
			headerTitle = "门店收银小票"
		}
	}
	isPreview := order.Status == "preview" || order.PaymentMethod == "preview"
	if isPreview {
		if tpl.HeaderTitle == "" {
			headerTitle = "预结算单"
		}
	}
	headerSubtitle := tpl.HeaderSubtitle
	if headerSubtitle == "" {
		if isPreview {
			headerSubtitle = "仅供确认明细，非正式收款凭证"
		} else {
			headerSubtitle = "欢迎光临"
		}
	}
	footerThanks := tpl.FooterThanks
	if footerThanks == "" {
		if isPreview {
			footerThanks = "请确认以上明细与金额后到店结算"
		} else {
			footerThanks = "谢谢惠顾，欢迎再次光临"
		}
	}

	paidAt := ""
	if order.PaidAt != nil {
		paidAt = order.PaidAt.Format("2006-01-02 15:04:05")
	} else {
		paidAt = order.CreatedAt.Format("2006-01-02 15:04:05")
	}

	var b strings.Builder
	b.WriteString(`<div class="receipt-doc">`)
	b.WriteString(`<div class="receipt-header">`)
	if tpl.ShowBrandLogo && brandLogo != "" {
		b.WriteString(fmt.Sprintf(`<div class="receipt-logo"><img src="%s" alt="" /></div>`, escapeReceipt(brandLogo)))
	}
	if tpl.ShowCoverPic && coverPic != "" {
		b.WriteString(fmt.Sprintf(`<div class="receipt-cover"><img src="%s" alt="" /></div>`, escapeReceipt(coverPic)))
	}
	b.WriteString(fmt.Sprintf(`<div class="receipt-title">%s</div>`, escapeReceipt(headerTitle)))
	b.WriteString(fmt.Sprintf(`<div class="receipt-subtitle">%s</div>`, escapeReceipt(headerSubtitle)))
	if storeName != "" && storeName != headerTitle {
		b.WriteString(fmt.Sprintf(`<div class="receipt-store">%s</div>`, escapeReceipt(storeName)))
	}
	if tpl.ShowStorePhone && storePhone != "" {
		b.WriteString(fmt.Sprintf(`<div class="receipt-meta-line">电话：%s</div>`, escapeReceipt(storePhone)))
	}
	if tpl.ShowStoreAddress && storeAddr != "" {
		b.WriteString(fmt.Sprintf(`<div class="receipt-meta-line">地址：%s</div>`, escapeReceipt(storeAddr)))
	}
	if tpl.ShowMapLabel && mapLabel != "" {
		b.WriteString(fmt.Sprintf(`<div class="receipt-meta-line">位置：%s</div>`, escapeReceipt(mapLabel)))
	}
	if tpl.ShowBusinessHours && businessHours != "" {
		b.WriteString(fmt.Sprintf(`<div class="receipt-meta-line">营业时间：%s</div>`, escapeReceipt(businessHours)))
	}
	if tpl.HeaderExtra != "" {
		b.WriteString(fmt.Sprintf(`<div class="receipt-extra">%s</div>`, nl2br(tpl.HeaderExtra)))
	}
	b.WriteString(`</div>`)

	b.WriteString(`<div class="receipt-divider"></div>`)
	b.WriteString(`<div class="receipt-meta">`)
	b.WriteString(fmt.Sprintf(`<div><span>单号</span><b>%s</b></div>`, escapeReceipt(order.OrderNo)))
	b.WriteString(fmt.Sprintf(`<div><span>时间</span><b>%s</b></div>`, escapeReceipt(paidAt)))
	b.WriteString(fmt.Sprintf(`<div><span>支付</span><b>%s</b></div>`, escapeReceipt(paymentMethodLabel(order.PaymentMethod))))
	if order.CustomerName != "" {
		b.WriteString(fmt.Sprintf(`<div><span>顾客</span><b>%s</b></div>`, escapeReceipt(order.CustomerName)))
	}
	b.WriteString(`</div>`)
	b.WriteString(`<div class="receipt-divider"></div>`)

	b.WriteString(`<div class="receipt-items">`)
	totalQty := 0
	for _, it := range items {
		totalQty += it.Quantity
		b.WriteString(`<div class="receipt-item">`)
		if tpl.ShowSkuPic {
			b.WriteString(`<div class="receipt-item-pic">`)
			if strings.TrimSpace(it.Pic) != "" {
				b.WriteString(fmt.Sprintf(`<img src="%s" alt="" />`, escapeReceipt(it.Pic)))
			} else {
				b.WriteString(`<div class="receipt-item-pic-empty">无图</div>`)
			}
			b.WriteString(`</div>`)
		}
		b.WriteString(`<div class="receipt-item-body">`)
		typeLabel := "商品"
		if it.ItemType == "service" {
			typeLabel = "服务"
		}
		b.WriteString(fmt.Sprintf(`<div class="receipt-item-name"><span class="receipt-item-type">%s</span> %s</div>`, typeLabel, escapeReceipt(it.ProductName)))
		if it.SpecLabel != "" {
			b.WriteString(fmt.Sprintf(`<div class="receipt-item-spec">%s</div>`, escapeReceipt(it.SpecLabel)))
		}
		if it.SkuCode != "" {
			label := "编码"
			if it.ItemType == "service" {
				label = "服务编码"
			}
			b.WriteString(fmt.Sprintf(`<div class="receipt-item-code">%s %s</div>`, label, escapeReceipt(it.SkuCode)))
		}
		b.WriteString(`<div class="receipt-item-row">`)
		if it.OriginalPrice > 0 && it.OriginalPrice > it.UnitPrice+0.001 {
			b.WriteString(fmt.Sprintf(`<span><span class="receipt-orig">原价 ¥%.2f</span> · 实付 ¥%.2f × %d`, it.OriginalPrice, it.UnitPrice, it.Quantity))
			if it.Discount > 0 && it.Discount < 10 {
				b.WriteString(fmt.Sprintf(` · %g折`, it.Discount))
			}
			b.WriteString(`</span>`)
		} else {
			b.WriteString(fmt.Sprintf(`<span>¥%.2f × %d</span>`, it.UnitPrice, it.Quantity))
		}
		b.WriteString(fmt.Sprintf(`<strong>¥%.2f</strong>`, it.TotalAmount))
		b.WriteString(`</div></div></div>`)
	}
	b.WriteString(`</div>`)

	b.WriteString(`<div class="receipt-divider"></div>`)
	b.WriteString(`<div class="receipt-summary">`)
	b.WriteString(fmt.Sprintf(`<div><span>件数</span><b>%d</b></div>`, totalQty))
	if order.OriginalAmount > 0 && order.OriginalAmount > order.TotalAmount+0.001 {
		b.WriteString(fmt.Sprintf(`<div><span>原价合计</span><b class="receipt-orig-sum">¥%.2f</b></div>`, order.OriginalAmount))
		b.WriteString(fmt.Sprintf(`<div><span>优惠</span><b>-¥%.2f</b></div>`, order.DiscountAmount))
	}
	if isPreview {
		b.WriteString(fmt.Sprintf(`<div class="receipt-total"><span>应付合计</span><b>¥%.2f</b></div>`, order.TotalAmount))
	} else {
		b.WriteString(fmt.Sprintf(`<div class="receipt-total"><span>实付合计</span><b>¥%.2f</b></div>`, order.TotalAmount))
		b.WriteString(fmt.Sprintf(`<div><span>实收</span><b>¥%.2f</b></div>`, order.PaidAmount))
	}
	b.WriteString(`</div>`)

	b.WriteString(`<div class="receipt-divider"></div>`)
	b.WriteString(`<div class="receipt-footer">`)
	b.WriteString(fmt.Sprintf(`<div class="receipt-thanks">%s</div>`, escapeReceipt(footerThanks)))
	if tpl.ShowGuideText && guideText != "" {
		b.WriteString(fmt.Sprintf(`<div class="receipt-guide"><div class="receipt-guide-title">到店指引</div><div class="receipt-guide-body">%s</div></div>`, nl2br(guideText)))
	}
	if tpl.FooterExtra != "" {
		b.WriteString(fmt.Sprintf(`<div class="receipt-extra">%s</div>`, nl2br(tpl.FooterExtra)))
	}
	b.WriteString(`</div></div>`)
	return b.String()
}
