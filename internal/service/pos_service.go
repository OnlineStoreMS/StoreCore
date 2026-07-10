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

func (s *PosService) Create(in *dto.PosOrderDTO, cashierUserID uint64) (*model.PosOrder, error) {
	if in.StoreID == 0 || len(in.Items) == 0 {
		return nil, ErrBadRequest
	}
	store, err := s.repos.Store.ForTenant(s.tenantID).GetByID(in.StoreID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrBadRequest
	}
	if err != nil {
		return nil, err
	}
	total := 0.0
	items := make([]model.PosOrderItem, 0, len(in.Items))
	for _, line := range in.Items {
		if line.SkuID == 0 || line.Quantity <= 0 {
			return nil, ErrBadRequest
		}
		lineTotal := line.UnitPrice * float64(line.Quantity)
		total += lineTotal
		items = append(items, model.PosOrderItem{
			SkuID: line.SkuID, ProductName: line.ProductName, SkuCode: line.SkuCode,
			SpecLabel: line.SpecLabel, Pic: strings.TrimSpace(line.Pic),
			Quantity: line.Quantity, UnitPrice: line.UnitPrice, TotalAmount: lineTotal,
		})
	}
	payStatus := "unpaid"
	status := "pending"
	paidAmount := 0.0
	if in.PaymentMethod == "cash" || in.PaymentMethod == "static_qr" {
		payStatus = "paid"
		status = "completed"
		paidAmount = total
	}
	now := time.Now()
	order := &model.PosOrder{
		StoreID:       in.StoreID,
		OrderNo:       genOrderNo("POS"),
		Status:        status,
		PaymentMethod: in.PaymentMethod,
		PayStatus:     payStatus,
		TotalAmount:   total,
		PaidAmount:    paidAmount,
		CustomerName:  in.CustomerName,
		CustomerPhone: in.CustomerPhone,
		CashierUserID: cashierUserID,
		ReceiptType:   defaultReceiptType(in.ReceiptType),
		Remark:        in.Remark,
	}
	if payStatus == "paid" {
		order.PaidAt = &now
		order.ReceiptHTML = s.buildReceiptHTML(order, items, store)
	}
	if err := s.repos.Pos.ForTenant(s.tenantID).Create(order, items); err != nil {
		return nil, err
	}
	if payStatus == "paid" {
		inv := s.repos.Inventory.ForTenant(s.tenantID)
		for _, line := range items {
			_ = inv.AddQuantity(in.StoreID, line.SkuID, line.SkuCode, line.ProductName, line.SpecLabel, -line.Quantity)
		}
	}
	return order, nil
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
		_ = inv.AddQuantity(order.StoreID, line.SkuID, line.SkuCode, line.ProductName, line.SpecLabel, -line.Quantity)
	}
	return order, nil
}

func genOrderNo(prefix string) string {
	return fmt.Sprintf("%s-%s-%s", prefix, time.Now().Format("20060102"), uuid.New().String()[:8])
}

func defaultReceiptType(t string) string {
	if t == "large" {
		return "large"
	}
	return "small"
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
	}

	headerTitle := tpl.HeaderTitle
	if headerTitle == "" {
		if storeName != "" {
			headerTitle = storeName
		} else {
			headerTitle = "门店收银小票"
		}
	}
	headerSubtitle := tpl.HeaderSubtitle
	if headerSubtitle == "" {
		headerSubtitle = "欢迎光临"
	}
	footerThanks := tpl.FooterThanks
	if footerThanks == "" {
		footerThanks = "谢谢惠顾，欢迎再次光临"
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
	b.WriteString(fmt.Sprintf(`<div class="receipt-title">%s</div>`, escapeReceipt(headerTitle)))
	b.WriteString(fmt.Sprintf(`<div class="receipt-subtitle">%s</div>`, escapeReceipt(headerSubtitle)))
	if storeName != "" && storeName != headerTitle {
		b.WriteString(fmt.Sprintf(`<div class="receipt-store">%s</div>`, escapeReceipt(storeName)))
	}
	if storePhone != "" {
		b.WriteString(fmt.Sprintf(`<div class="receipt-meta-line">电话：%s</div>`, escapeReceipt(storePhone)))
	}
	if storeAddr != "" {
		b.WriteString(fmt.Sprintf(`<div class="receipt-meta-line">地址：%s</div>`, escapeReceipt(storeAddr)))
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
		b.WriteString(fmt.Sprintf(`<div class="receipt-item-name">%s</div>`, escapeReceipt(it.ProductName)))
		if it.SpecLabel != "" {
			b.WriteString(fmt.Sprintf(`<div class="receipt-item-spec">%s</div>`, escapeReceipt(it.SpecLabel)))
		}
		if it.SkuCode != "" {
			b.WriteString(fmt.Sprintf(`<div class="receipt-item-code">编码 %s</div>`, escapeReceipt(it.SkuCode)))
		}
		b.WriteString(`<div class="receipt-item-row">`)
		b.WriteString(fmt.Sprintf(`<span>¥%.2f × %d</span>`, it.UnitPrice, it.Quantity))
		b.WriteString(fmt.Sprintf(`<strong>¥%.2f</strong>`, it.TotalAmount))
		b.WriteString(`</div></div></div>`)
	}
	b.WriteString(`</div>`)

	b.WriteString(`<div class="receipt-divider"></div>`)
	b.WriteString(`<div class="receipt-summary">`)
	b.WriteString(fmt.Sprintf(`<div><span>件数</span><b>%d</b></div>`, totalQty))
	b.WriteString(fmt.Sprintf(`<div class="receipt-total"><span>合计</span><b>¥%.2f</b></div>`, order.TotalAmount))
	b.WriteString(fmt.Sprintf(`<div><span>实收</span><b>¥%.2f</b></div>`, order.PaidAmount))
	b.WriteString(`</div>`)

	b.WriteString(`<div class="receipt-divider"></div>`)
	b.WriteString(`<div class="receipt-footer">`)
	b.WriteString(fmt.Sprintf(`<div class="receipt-thanks">%s</div>`, escapeReceipt(footerThanks)))
	if tpl.FooterExtra != "" {
		b.WriteString(fmt.Sprintf(`<div class="receipt-extra">%s</div>`, nl2br(tpl.FooterExtra)))
	}
	b.WriteString(`</div></div>`)
	return b.String()
}
