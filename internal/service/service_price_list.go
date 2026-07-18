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
	// 服务项目暂无图片：统一用齿轮图标（视觉上接近收银台 Tools 占位）
	iconSVG := `<svg class="svc-icon" viewBox="0 0 1024 1024" width="20" height="20" aria-hidden="true"><path fill="#909399" d="M512 704a192 192 0 1 0 0-384 192 192 0 0 0 0 384zm0-64a128 128 0 1 1 0-256 128 128 0 0 1 0 256zm418.8-194.4l-65.7-37.9a286.1 286.1 0 0 0 0-145.4l65.7-37.9a32 32 0 0 0 11.7-43.7l-64-110.8a32 32 0 0 0-43.7-11.7l-65.7 37.9a286.9 286.9 0 0 0-125.9-72.7V48a32 32 0 0 0-32-32h-128a32 32 0 0 0-32 32v75.8a286.9 286.9 0 0 0-125.9 72.7l-65.7-37.9a32 32 0 0 0-43.7 11.7l-64 110.8a32 32 0 0 0 11.7 43.7l65.7 37.9a286.1 286.1 0 0 0 0 145.4l-65.7 37.9a32 32 0 0 0-11.7 43.7l64 110.8a32 32 0 0 0 43.7 11.7l65.7-37.9a286.9 286.9 0 0 0 125.9 72.7V976a32 32 0 0 0 32 32h128a32 32 0 0 0 32-32v-75.8a286.9 286.9 0 0 0 125.9-72.7l65.7 37.9a32 32 0 0 0 43.7-11.7l64-110.8a32 32 0 0 0-11.7-43.7zM832 512a320 320 0 1 1-640 0 320 320 0 0 1 640 0z"/></svg>`

	for _, g := range groups {
		if g.Name != "" {
			b.WriteString(`<div class="sales-doc-section">` + htmlEscape(g.Name) + `</div>`)
		} else {
			b.WriteString(`<div class="sales-doc-section">服务项目</div>`)
		}
		b.WriteString(`<table class="sales-doc-table price-list-table"><thead><tr>`)
		b.WriteString(`<th class="col-idx">#</th>`)
		b.WriteString(`<th class="col-pic"> </th>`)
		b.WriteString(`<th class="col-name">服务项目</th>`)
		if showDur {
			b.WriteString(`<th class="num">参考时长</th>`)
		}
		b.WriteString(`<th class="num">价格</th>`)
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
				b.WriteString(`<td class="num">` + htmlEscape(formatServiceDuration(it.DurationMin)) + `</td>`)
			}
			b.WriteString(fmt.Sprintf(`<td class="num strong">¥%.2f</td>`, it.Price))
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
	b.WriteString(`<style>
.price-list-doc .price-list-extra{margin:8px 0 12px;font-size:13px;color:#606266;line-height:1.5}
.price-list-doc .spec.desc{margin-top:4px;color:#606266;white-space:pre-wrap}
.price-list-doc .sales-doc-footer.muted{color:#909399;font-size:12px;margin-top:6px}
.price-list-table .col-name{min-width:180px}
.price-list-table .col-pic{width:40px}
.price-list-table .svc-icon-wrap{width:32px;height:32px;border-radius:6px;background:#f0f2f5;display:flex;align-items:center;justify-content:center}
.price-list-table .svc-icon{display:block}
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
