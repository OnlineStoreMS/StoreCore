package service

import (
	"fmt"
	"sort"
	"strings"

	"storecore/internal/dto"
	"storecore/internal/model"
)

type ServicePriceListResult struct {
	HTML     string   `json:"html"`
	ItemCount int     `json:"itemCount"`
	StoreName string  `json:"storeName"`
}

func (s *ServiceCatalogService) GeneratePriceList(in *dto.ServicePriceListDTO) (*ServicePriceListResult, error) {
	if in.StoreID == 0 || len(in.ServiceItemIDs) == 0 {
		return nil, fmt.Errorf("%w：请选择门店与服务项目", ErrBadRequest)
	}
	store, err := s.repos.Store.ForTenant(s.tenantID).GetByID(in.StoreID)
	if err != nil {
		return nil, fmt.Errorf("%w：门店不存在", ErrBadRequest)
	}

	tpl, err := s.resolvePriceListTemplate(in.StoreID, in.TemplateID)
	if err != nil {
		return nil, err
	}

	items, err := s.repos.ServiceCatalog.ForTenant(s.tenantID).ListItemsByIDs(in.ServiceItemIDs)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("%w：未找到所选服务", ErrBadRequest)
	}

	// 保持勾选顺序
	byID := make(map[uint64]model.ServiceItem, len(items))
	for _, it := range items {
		byID[it.ID] = it
	}
	ordered := make([]model.ServiceItem, 0, len(in.ServiceItemIDs))
	for _, id := range in.ServiceItemIDs {
		if it, ok := byID[id]; ok {
			ordered = append(ordered, it)
		}
	}

	cats, _ := s.repos.ServiceCatalog.ForTenant(s.tenantID).ListCategories()
	catName := map[uint64]string{}
	for _, c := range cats {
		catName[c.ID] = c.Name
	}

	html := buildServicePriceListHTML(store, tpl, ordered, catName, true)
	return &ServicePriceListResult{
		HTML:      html,
		ItemCount: len(ordered),
		StoreName: store.Name,
	}, nil
}

func (s *ServiceCatalogService) resolvePriceListTemplate(storeID, templateID uint64) (*model.ReceiptTemplate, error) {
	r := s.repos.ReceiptTpl.ForTenant(s.tenantID)
	if templateID > 0 {
		tpl, err := r.GetByID(templateID)
		if err != nil {
			return nil, fmt.Errorf("%w：价目表模板不存在", ErrBadRequest)
		}
		if tpl.ReceiptType != "price_list" {
			return nil, fmt.Errorf("%w：请选择价目表模板", ErrBadRequest)
		}
		return tpl, nil
	}
	tpl, err := r.FindDefault(storeID, "price_list")
	if err != nil || tpl == nil {
		return &model.ReceiptTemplate{
			Name:              "默认服务价目表",
			ReceiptType:       "price_list",
			HeaderTitle:       "服务价目表",
			HeaderSubtitle:    "到店服务报价参考",
			FooterThanks:      "价格如有变动以到店确认为准，欢迎咨询门店",
			FooterExtra:       "",
			ShowSkuPic:        false, // 暂不展示图片，统一工具图标
			ShowStorePhone:    true,
			ShowStoreAddress:  true,
			ShowBusinessHours: true,
			ShowBrandLogo:     true,
			ShowDescription:   true,
			ShowDuration:      true,
			ShowWechatMpQr:    true,
			ShowGroupBuyQr:    true,
			Status:            1,
		}, nil
	}
	return tpl, nil
}

func buildServicePriceListHTML(
	store *model.Store,
	tpl *model.ReceiptTemplate,
	items []model.ServiceItem,
	catName map[uint64]string,
	groupByCategory bool,
) string {
	title := strings.TrimSpace(tpl.HeaderTitle)
	if title == "" {
		title = "服务价目表"
	}
	subtitle := strings.TrimSpace(tpl.HeaderSubtitle)
	storeName := ""
	storePhone := ""
	storeAddr := ""
	businessHours := ""
	brandLogo := ""
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
		businessHours = store.BusinessHours
		brandLogo = store.BrandLogo
	}

	var b strings.Builder
	b.WriteString(`<div class="sales-doc price-list-doc">`)
	b.WriteString(`<div class="sales-doc-header">`)
	if tpl.ShowBrandLogo && strings.TrimSpace(brandLogo) != "" {
		b.WriteString(`<div class="sales-doc-logo"><img src="` + htmlEscape(brandLogo) + `" alt="logo" /></div>`)
	}
	b.WriteString(`<div class="sales-doc-title-block">`)
	b.WriteString(`<div class="sales-doc-title">` + htmlEscape(title) + `</div>`)
	if subtitle != "" {
		b.WriteString(`<div class="sales-doc-badge">` + htmlEscape(subtitle) + `</div>`)
	}
	b.WriteString(`</div></div>`)

	b.WriteString(`<table class="sales-doc-info"><tbody>`)
	writeRow := func(k1, v1, k2, v2 string) {
		b.WriteString(`<tr>`)
		b.WriteString(`<th>` + htmlEscape(k1) + `</th><td>` + v1 + `</td>`)
		b.WriteString(`<th>` + htmlEscape(k2) + `</th><td>` + v2 + `</td>`)
		b.WriteString(`</tr>`)
	}
	phoneCell := "-"
	if tpl.ShowStorePhone {
		phoneCell = nz(storePhone, "-")
	}
	writeRow("门店", htmlEscape(nz(storeName, "-")), "电话", htmlEscape(phoneCell))
	if tpl.ShowStoreAddress {
		writeRow("地址", htmlEscape(nz(storeAddr, "-")), "", "")
	}
	if tpl.ShowBusinessHours {
		writeRow("营业时间", htmlEscape(nz(businessHours, "-")), "", "")
	}
	b.WriteString(`</tbody></table>`)
	if strings.TrimSpace(tpl.HeaderExtra) != "" {
		b.WriteString(`<div class="price-list-extra">` + nl2br(tpl.HeaderExtra) + `</div>`)
	}

	type group struct {
		Name  string
		Items []model.ServiceItem
	}
	var groups []group
	if groupByCategory {
		orderKeys := make([]uint64, 0)
		idx := map[uint64]int{}
		for _, it := range items {
			cid := it.CategoryID
			if _, ok := idx[cid]; !ok {
				idx[cid] = len(orderKeys)
				orderKeys = append(orderKeys, cid)
				name := catName[cid]
				if name == "" {
					name = "其他服务"
				}
				groups = append(groups, group{Name: name})
			}
			groups[idx[cid]].Items = append(groups[idx[cid]].Items, it)
		}
		for i := range groups {
			sort.SliceStable(groups[i].Items, func(a, b int) bool {
				if groups[i].Items[a].Sort != groups[i].Items[b].Sort {
					return groups[i].Items[a].Sort < groups[i].Items[b].Sort
				}
				return groups[i].Items[a].ID < groups[i].Items[b].ID
			})
		}
	} else {
		groups = []group{{Name: "", Items: items}}
	}

	showDesc := tpl.ShowDescription
	showDur := tpl.ShowDuration
	// 与收银台一致：Element Plus Tools 齿轮图标 + 暖橙底
	iconSVG := `<svg class="svc-icon" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1024 1024" width="20" height="20" aria-hidden="true"><path fill="currentColor" d="M764.416 254.72a351.7 351.7 0 0 1 86.336 149.184H960v192.064H850.752a351.7 351.7 0 0 1-86.336 149.312l54.72 94.72-166.272 96-54.592-94.72a352.64 352.64 0 0 1-172.48 0L371.136 936l-166.272-96 54.72-94.72a351.7 351.7 0 0 1-86.336-149.312H64v-192h109.248a351.7 351.7 0 0 1 86.336-149.312L204.8 160l166.208-96h.192l54.656 94.592a352.64 352.64 0 0 1 172.48 0L652.8 64h.128L819.2 160l-54.72 94.72zM704 499.968a192 192 0 1 0-384 0 192 192 0 0 0 384 0"/></svg>`

	for _, g := range groups {
		if g.Name != "" {
			b.WriteString(`<div class="sales-doc-section">` + htmlEscape(g.Name) + `</div>`)
		} else {
			b.WriteString(`<div class="sales-doc-section">服务项目</div>`)
		}
		b.WriteString(`<table class="sales-doc-table price-list-table"><colgroup>`)
		b.WriteString(`<col class="col-idx" /><col class="col-pic" /><col class="col-name" />`)
		if showDur {
			b.WriteString(`<col class="col-dur" />`)
		}
		b.WriteString(`<col class="col-price" /></colgroup>`)
		b.WriteString(`<thead><tr>`)
		b.WriteString(`<th class="col-idx">#</th>`)
		b.WriteString(`<th class="col-pic"> </th>`)
		b.WriteString(`<th class="col-name">服务项目</th>`)
		if showDur {
			b.WriteString(`<th class="num col-dur">参考时长</th>`)
		}
		b.WriteString(`<th class="num col-price">价格</th>`)
		b.WriteString(`</tr></thead><tbody>`)
		for i, it := range g.Items {
			b.WriteString(`<tr>`)
			b.WriteString(fmt.Sprintf(`<td class="col-idx">%d</td>`, i+1))
			b.WriteString(`<td class="col-pic"><div class="svc-icon-wrap">` + iconSVG + `</div></td>`)
			b.WriteString(`<td class="col-name"><div class="name">` + htmlEscape(it.Name) + `</div>`)
			if strings.TrimSpace(it.Code) != "" {
				b.WriteString(`<div class="spec">编码 ` + htmlEscape(it.Code) + `</div>`)
			}
			if showDesc && strings.TrimSpace(it.Description) != "" {
				b.WriteString(`<div class="spec desc">` + htmlEscape(it.Description) + `</div>`)
			}
			b.WriteString(`</td>`)
			if showDur {
				b.WriteString(`<td class="num col-dur">` + htmlEscape(formatServiceDuration(it.DurationMin)) + `</td>`)
			}
			b.WriteString(fmt.Sprintf(`<td class="num col-price strong">¥%.2f</td>`, it.Price))
			b.WriteString(`</tr>`)
		}
		b.WriteString(`</tbody></table>`)
	}

	footer := strings.TrimSpace(tpl.FooterThanks)
	if footer == "" {
		footer = "价格如有变动以到店确认为准"
	}
	b.WriteString(`<div class="sales-doc-footer">` + htmlEscape(footer) + `</div>`)
	if strings.TrimSpace(tpl.FooterExtra) != "" {
		b.WriteString(`<div class="sales-doc-footer muted">` + nl2br(tpl.FooterExtra) + `</div>`)
	}

	// 门店二维码放最底部：靠左、小尺寸、低调
	mpQr := ""
	groupQr := ""
	if store != nil {
		mpQr = strings.TrimSpace(store.WechatMpQrCode)
		groupQr = strings.TrimSpace(store.GroupBuyQrCode)
	}
	showMp := tpl.ShowWechatMpQr && mpQr != ""
	showGroup := tpl.ShowGroupBuyQr && groupQr != ""
	if showMp || showGroup {
		b.WriteString(`<div class="price-list-qr-row">`)
		if showMp {
			b.WriteString(`<div class="qr-item"><img src="` + htmlEscape(mpQr) + `" alt="微信小程序" /><div class="qr-label">微信小程序</div></div>`)
		}
		if showGroup {
			b.WriteString(`<div class="qr-item"><img src="` + htmlEscape(groupQr) + `" alt="门店团购" /><div class="qr-label">门店团购</div></div>`)
		}
		b.WriteString(`</div>`)
	}

	b.WriteString(`<style>
.price-list-doc .price-list-extra{margin:8px 0 12px;font-size:13px;color:#606266;line-height:1.5}
.price-list-doc .spec.desc{margin-top:4px;color:#606266;white-space:pre-wrap}
.price-list-doc .sales-doc-footer.muted{color:#909399;font-size:12px;margin-top:6px}
.price-list-qr-row{display:flex;justify-content:flex-start;align-items:flex-start;gap:20px;margin:14px 0 0;padding:10px 0 0;border-top:1px solid #f0f0f0}
.price-list-qr-row .qr-item{text-align:center;width:72px}
.price-list-qr-row .qr-item img{width:64px;height:64px;object-fit:contain;display:block;margin:0 auto;border-radius:4px;background:#fafafa;opacity:.92}
.price-list-qr-row .qr-label{margin-top:4px;font-size:11px;color:#909399;line-height:1.3}
/* 各分类独立表格：固定列宽，保证跨类别预览列对齐 */
.price-list-table{table-layout:fixed;width:100%}
.price-list-table col.col-idx{width:36px}
.price-list-table col.col-pic{width:52px}
.price-list-table col.col-dur{width:112px}
.price-list-table col.col-price{width:88px}
.price-list-table .col-name .name,.price-list-table .col-name .spec{overflow:hidden;word-break:break-word}
.price-list-table td.col-pic,.price-list-table th.col-pic{text-align:center;vertical-align:middle;padding:6px 4px}
.price-list-table .svc-icon-wrap{display:inline-flex;align-items:center;justify-content:center;width:36px;height:36px;border-radius:8px;background:#fff7e6;color:#e6a23c;line-height:0;vertical-align:middle}
.price-list-table .svc-icon{display:block;width:20px;height:20px;flex:none}
</style>`)
	b.WriteString(`</div>`)
	return b.String()
}

// formatServiceDuration 分钟 → 易读时长：不足 1 小时「约 xx分钟」，否则「约 x小时」/「约 x小时xx分」
func formatServiceDuration(minutes int) string {
	if minutes <= 0 {
		return "-"
	}
	if minutes < 60 {
		return fmt.Sprintf("约 %d分钟", minutes)
	}
	h := minutes / 60
	rest := minutes % 60
	if rest == 0 {
		return fmt.Sprintf("约 %d小时", h)
	}
	return fmt.Sprintf("约 %d小时%d分", h, rest)
}
