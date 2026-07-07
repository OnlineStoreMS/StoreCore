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
	if _, err := s.repos.Store.ForTenant(s.tenantID).GetByID(in.StoreID); errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrBadRequest
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
			SpecLabel: line.SpecLabel, Quantity: line.Quantity,
			UnitPrice: line.UnitPrice, TotalAmount: lineTotal,
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
		StoreID: in.StoreID,
		OrderNo: genOrderNo("POS"),
		Status: status,
		PaymentMethod: in.PaymentMethod,
		PayStatus: payStatus,
		TotalAmount: total,
		PaidAmount: paidAmount,
		CustomerName: in.CustomerName,
		CustomerPhone: in.CustomerPhone,
		CashierUserID: cashierUserID,
		ReceiptType: defaultReceiptType(in.ReceiptType),
		Remark: in.Remark,
	}
	if payStatus == "paid" {
		order.PaidAt = &now
		order.ReceiptHTML = buildReceiptHTML(order, items)
	}
	if err := s.repos.Pos.ForTenant(s.tenantID).Create(order, items); err != nil {
		return nil, err
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
	now := time.Now()
	order.PayStatus = "paid"
	order.Status = "completed"
	order.PaidAmount = order.TotalAmount
	order.PaidAt = &now
	order.ReceiptHTML = buildReceiptHTML(order, order.Items)
	if err := r.Update(order); err != nil {
		return nil, err
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

func buildReceiptHTML(order *model.PosOrder, items []model.PosOrderItem) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("<h3>门店收银小票</h3><p>单号：%s</p>", order.OrderNo))
	for _, it := range items {
		b.WriteString(fmt.Sprintf("<p>%s x%d = %.2f</p>", it.ProductName, it.Quantity, it.TotalAmount))
	}
	b.WriteString(fmt.Sprintf("<p><strong>合计：%.2f</strong></p>", order.TotalAmount))
	return b.String()
}
