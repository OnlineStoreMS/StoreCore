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

func normalizeProcessPhase(phase string) (string, error) {
	switch strings.TrimSpace(strings.ToLower(phase)) {
	case "before":
		return "before", nil
	case "after":
		return "after", nil
	default:
		return "", fmt.Errorf("%w：过程阶段须为 before（服务前）或 after（服务后）", ErrBadRequest)
	}
}

func normalizeProcessMedia(items []dto.ServiceProcessMediaDTO) ([]model.ServiceProcessMediaItem, error) {
	out := make([]model.ServiceProcessMediaItem, 0, len(items))
	for _, it := range items {
		url := strings.TrimSpace(it.URL)
		if url == "" {
			continue
		}
		mt := strings.TrimSpace(strings.ToLower(it.MediaType))
		if mt == "" {
			mt = guessMediaType(url)
		}
		if mt != "image" && mt != "video" {
			return nil, fmt.Errorf("%w：媒体类型仅支持 image / video", ErrBadRequest)
		}
		out = append(out, model.ServiceProcessMediaItem{URL: url, MediaType: mt})
	}
	return out, nil
}

func guessMediaType(url string) string {
	u := strings.ToLower(url)
	for _, ext := range []string{".mp4", ".mov", ".webm", ".m4v", ".avi", ".mkv"} {
		if strings.Contains(u, ext) {
			return "video"
		}
	}
	return "image"
}

func (s *ServiceOrderService) CreateProcessRecord(orderID uint64, in *dto.ServiceProcessRecordDTO, userID uint64) (*model.ServiceOrder, error) {
	r := s.repos.Service.ForTenant(s.tenantID)
	order, err := r.GetByID(orderID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if order.Status == "cancelled" {
		return nil, fmt.Errorf("%w：已取消工单不可添加过程纪录", ErrBadRequest)
	}
	phase, err := normalizeProcessPhase(in.Phase)
	if err != nil {
		return nil, err
	}
	media, err := normalizeProcessMedia(in.Media)
	if err != nil {
		return nil, err
	}
	note := strings.TrimSpace(in.Note)
	if len(media) == 0 && note == "" {
		return nil, fmt.Errorf("%w：请上传图片/视频或填写说明", ErrBadRequest)
	}
	rec := &model.ServiceProcessRecord{
		ServiceOrderID: orderID,
		Phase:          phase,
		Note:           note,
		Media:          media,
		CreatedBy:      userID,
	}
	if err := r.CreateProcessRecord(rec); err != nil {
		return nil, err
	}
	return r.GetByID(orderID)
}

func (s *ServiceOrderService) UpdateProcessRecord(orderID, recordID uint64, in *dto.ServiceProcessRecordDTO) (*model.ServiceOrder, error) {
	r := s.repos.Service.ForTenant(s.tenantID)
	order, err := r.GetByID(orderID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if order.Status == "cancelled" {
		return nil, fmt.Errorf("%w：已取消工单不可修改过程纪录", ErrBadRequest)
	}
	rec, err := r.GetProcessRecord(recordID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if rec.ServiceOrderID != orderID {
		return nil, ErrNotFound
	}
	phase, err := normalizeProcessPhase(in.Phase)
	if err != nil {
		return nil, err
	}
	media, err := normalizeProcessMedia(in.Media)
	if err != nil {
		return nil, err
	}
	note := strings.TrimSpace(in.Note)
	if len(media) == 0 && note == "" {
		return nil, fmt.Errorf("%w：请上传图片/视频或填写说明", ErrBadRequest)
	}
	rec.Phase = phase
	rec.Note = note
	rec.Media = media
	if err := r.UpdateProcessRecord(rec); err != nil {
		return nil, err
	}
	return r.GetByID(orderID)
}

func (s *ServiceOrderService) DeleteProcessRecord(orderID, recordID uint64) (*model.ServiceOrder, error) {
	r := s.repos.Service.ForTenant(s.tenantID)
	if _, err := r.GetByID(orderID); errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	} else if err != nil {
		return nil, err
	}
	rec, err := r.GetProcessRecord(recordID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if rec.ServiceOrderID != orderID {
		return nil, ErrNotFound
	}
	if err := r.DeleteProcessRecord(recordID); err != nil {
		return nil, err
	}
	return r.GetByID(orderID)
}

func (s *ServiceOrderService) requireProcessPhase(orderID uint64, phase, tip string) error {
	n, err := s.repos.Service.ForTenant(s.tenantID).CountProcessMediaByPhase(orderID, phase)
	if err != nil {
		return err
	}
	if n < 1 {
		return fmt.Errorf("%w：%s", ErrBadRequest, tip)
	}
	return nil
}

func (s *ServiceOrderService) attachServiceReport(order *model.ServiceOrder) {
	if order == nil {
		return
	}
	records := order.ProcessRecords
	if len(records) == 0 {
		records, _ = s.repos.Service.ForTenant(s.tenantID).ListProcessRecords(order.ID)
	}
	store, _ := s.repos.Store.ForTenant(s.tenantID).GetByID(order.StoreID)
	order.ReportHTML = s.buildServiceReportHTML(order, records, store)
}

func (s *ServiceOrderService) RefreshReport(id uint64) (*model.ServiceOrder, error) {
	r := s.repos.Service.ForTenant(s.tenantID)
	order, err := r.GetByID(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	s.attachServiceReport(order)
	if err := r.Update(order, nil); err != nil {
		return nil, err
	}
	return order, nil
}

type ServiceDocBundleResult struct {
	HTML string `json:"html"`
}

func (s *ServiceOrderService) DocBundle(id uint64, includeReceipt, includeReport bool) (*ServiceDocBundleResult, error) {
	if !includeReceipt && !includeReport {
		includeReceipt = true
		includeReport = true
	}
	r := s.repos.Service.ForTenant(s.tenantID)
	order, err := r.GetByID(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	store, _ := s.repos.Store.ForTenant(s.tenantID).GetByID(order.StoreID)
	var parts []string
	if includeReceipt {
		if strings.TrimSpace(order.ReceiptHTML) == "" {
			s.attachServiceReceipt(order, order.Items)
		}
		if html := strings.TrimSpace(order.ReceiptHTML); html != "" {
			parts = append(parts, html)
		}
	}
	if includeReport {
		if strings.TrimSpace(order.ReportHTML) == "" {
			s.attachServiceReport(order)
		}
		if html := strings.TrimSpace(order.ReportHTML); html != "" {
			parts = append(parts, html)
		} else {
			// 即使尚无报告也生成一份（可能无过程纪录）
			html := s.buildServiceReportHTML(order, order.ProcessRecords, store)
			if strings.TrimSpace(html) != "" {
				parts = append(parts, html)
			}
		}
	}
	if len(parts) == 0 {
		return nil, fmt.Errorf("%w：无可合并的单据内容", ErrBadRequest)
	}
	return &ServiceDocBundleResult{HTML: joinDocHTML(parts)}, nil
}

func joinDocHTML(parts []string) string {
	if len(parts) == 1 {
		return parts[0]
	}
	var b strings.Builder
	b.WriteString(`<div class="service-doc-bundle">`)
	for i, p := range parts {
		if i > 0 {
			b.WriteString(`<div class="service-doc-page-break" style="page-break-before:always;border-top:2px dashed #ccc;margin:24px 0;padding-top:16px;"></div>`)
		}
		b.WriteString(p)
	}
	b.WriteString(`</div>`)
	return b.String()
}

func (s *ServiceOrderService) buildServiceReportHTML(order *model.ServiceOrder, records []model.ServiceProcessRecord, store *model.Store) string {
	if order == nil {
		return ""
	}
	storeName := ""
	storePhone := ""
	if store != nil {
		storeName = store.Name
		storePhone = store.Phone
	}
	var b strings.Builder
	b.WriteString(`<div class="sales-doc service-report">`)
	b.WriteString(`<div class="sales-doc-header">`)
	b.WriteString(`<div class="sales-doc-title">服务报告</div>`)
	b.WriteString(`<div class="sales-doc-sub">` + htmlEscape(nz(storeName, "门店")) + ` · ` + htmlEscape(order.OrderNo) + `</div>`)
	b.WriteString(`</div>`)

	b.WriteString(`<table class="sales-doc-info"><tbody>`)
	writeInfo := func(k1, v1, k2, v2 string) {
		b.WriteString(`<tr>`)
		b.WriteString(`<th>` + htmlEscape(k1) + `</th><td>` + v1 + `</td>`)
		b.WriteString(`<th>` + htmlEscape(k2) + `</th><td>` + v2 + `</td>`)
		b.WriteString(`</tr>`)
	}
	writeInfo("客户", htmlEscape(nz(order.CustomerName, "-")), "电话", htmlEscape(nz(order.CustomerPhone, "-")))
	writeInfo("设备", htmlEscape(nz(order.DeviceInfo, "-")), "工程师", htmlEscape(nz(order.EngineerName, "-")))
	writeInfo("说明", htmlEscape(nz(order.FaultDesc, "-")), "门店电话", htmlEscape(nz(storePhone, "-")))
	b.WriteString(`</tbody></table>`)

	phaseLabel := map[string]string{"before": "服务前", "after": "服务后"}
	renderPhase := func(phase string) {
		b.WriteString(`<div class="sales-doc-section">` + phaseLabel[phase] + `</div>`)
		found := false
		for _, rec := range records {
			if rec.Phase != phase {
				continue
			}
			found = true
			b.WriteString(`<div class="process-block">`)
			if strings.TrimSpace(rec.Note) != "" {
				b.WriteString(`<div class="process-note">` + htmlEscape(rec.Note) + `</div>`)
			}
			if len(rec.Media) > 0 {
				b.WriteString(`<div class="process-media">`)
				for _, m := range rec.Media {
					if m.MediaType == "video" {
						b.WriteString(`<div class="media-item video"><a href="` + htmlEscape(m.URL) + `" target="_blank" rel="noopener">视频</a></div>`)
					} else {
						b.WriteString(`<div class="media-item"><img src="` + htmlEscape(m.URL) + `" alt="" /></div>`)
					}
				}
				b.WriteString(`</div>`)
			}
			if !rec.CreatedAt.IsZero() {
				b.WriteString(`<div class="process-time">` + htmlEscape(rec.CreatedAt.Format("2006-01-02 15:04")) + `</div>`)
			}
			b.WriteString(`</div>`)
		}
		if !found {
			b.WriteString(`<div class="process-empty">暂无纪录</div>`)
		}
	}
	renderPhase("before")
	renderPhase("after")

	b.WriteString(`<div class="sales-doc-footer">客户确认：____________　　工程师：____________　　日期：` +
		htmlEscape(time.Now().Format("2006-01-02")) + `</div>`)
	b.WriteString(`<style>
.service-report .process-block{margin:8px 0 14px;padding:10px;border:1px solid #eee;border-radius:6px}
.service-report .process-note{font-size:13px;line-height:1.5;margin-bottom:8px;white-space:pre-wrap}
.service-report .process-media{display:flex;flex-wrap:wrap;gap:8px}
.service-report .media-item{width:120px;height:120px;border:1px solid #eee;border-radius:4px;overflow:hidden;background:#fafafa;display:flex;align-items:center;justify-content:center}
.service-report .media-item img{width:100%;height:100%;object-fit:cover}
.service-report .media-item.video a{font-size:13px;color:#409eff}
.service-report .process-time{margin-top:6px;font-size:11px;color:#999}
.service-report .process-empty{font-size:12px;color:#999;padding:8px 0}
</style>`)
	b.WriteString(`</div>`)
	return b.String()
}
