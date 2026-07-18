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
	b.WriteString(`<div class="sales-doc price-list-doc price-list-poster">`)
	// 海报头带：大 Logo + 标题（品牌优先）
	b.WriteString(`<div class="pl-hero-band">`)
	b.WriteString(`<div class="pl-hero">`)
	if tpl.ShowBrandLogo && strings.TrimSpace(brandLogo) != "" {
		b.WriteString(`<div class="pl-logo"><img src="` + htmlEscape(brandLogo) + `" alt="logo" /></div>`)
	}
	b.WriteString(`<div class="pl-hero-text">`)
	b.WriteString(`<div class="pl-title">` + htmlEscape(title) + `</div>`)
	if storeName != "" {
		b.WriteString(`<div class="pl-store">` + htmlEscape(storeName) + `</div>`)
	}
	if subtitle != "" {
		b.WriteString(`<div class="pl-sub">` + htmlEscape(subtitle) + `</div>`)
	}
	b.WriteString(`</div></div>`)

	var chips []string
	if tpl.ShowStorePhone && strings.TrimSpace(storePhone) != "" {
		chips = append(chips, `<span class="pl-chip">☎ `+htmlEscape(storePhone)+`</span>`)
	}
	if tpl.ShowBusinessHours && strings.TrimSpace(businessHours) != "" {
		chips = append(chips, `<span class="pl-chip">营业 `+htmlEscape(businessHours)+`</span>`)
	}
	if len(chips) > 0 {
		b.WriteString(`<div class="pl-chips">` + strings.Join(chips, "") + `</div>`)
	}
	if tpl.ShowStoreAddress && strings.TrimSpace(storeAddr) != "" {
		b.WriteString(`<div class="pl-addr">` + htmlEscape(storeAddr) + `</div>`)
	}
	b.WriteString(`</div>`) // pl-hero-band

	if strings.TrimSpace(tpl.HeaderExtra) != "" {
		b.WriteString(`<div class="price-list-extra">` + nl2br(tpl.HeaderExtra) + `</div>`)
	}

	b.WriteString(`<div class="pl-body">`)

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
	iconImg := serviceToolsIconHTML()

	for _, g := range groups {
		sec := g.Name
		if sec == "" {
			sec = "服务项目"
		}
		b.WriteString(`<div class="pl-section">`)
		b.WriteString(`<div class="pl-section-title"><span>` + htmlEscape(sec) + `</span></div>`)
		b.WriteString(`<div class="pl-list">`)
		for _, it := range g.Items {
			b.WriteString(`<div class="pl-item">`)
			b.WriteString(`<div class="pl-item-main">`)
			b.WriteString(`<div class="svc-icon-wrap">` + iconImg + `</div>`)
			b.WriteString(`<div class="pl-item-body">`)
			b.WriteString(`<div class="pl-item-name">` + htmlEscape(it.Name) + `</div>`)
			if showDur && it.DurationMin > 0 {
				b.WriteString(`<div class="pl-item-meta">` + htmlEscape(formatServiceDuration(it.DurationMin)) + `</div>`)
			}
			if showDesc && strings.TrimSpace(it.Description) != "" {
				b.WriteString(`<div class="pl-item-desc">` + htmlEscape(compactPosterDesc(it.Description)) + `</div>`)
			}
			b.WriteString(`</div></div>`)
			b.WriteString(`<div class="pl-item-price">` + htmlEscape(formatPosterPrice(it.Price)) + `</div>`)
			b.WriteString(`</div>`)
		}
		b.WriteString(`</div></div>`)
	}
	b.WriteString(`</div>`) // pl-body

	footer := strings.TrimSpace(tpl.FooterThanks)
	if footer == "" {
		footer = "价格如有变动以到店确认为准"
	}
	b.WriteString(`<div class="pl-bottom">`)
	b.WriteString(`<div class="pl-footer">` + htmlEscape(footer) + `</div>`)
	if strings.TrimSpace(tpl.FooterExtra) != "" {
		b.WriteString(`<div class="pl-footer muted">` + nl2br(tpl.FooterExtra) + `</div>`)
	}

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
	b.WriteString(`</div>`) // pl-bottom

	b.WriteString(`<style>
.price-list-poster{display:flex;flex-direction:column;box-sizing:border-box;padding:0;color:#1f2937;background:#fff}
.price-list-poster .pl-hero-band{margin:0 0 16px;padding:20px 18px 16px;border-radius:0 0 20px 20px;background:linear-gradient(165deg,#fffaf0 0%,#fff7e6 42%,#ffffff 100%);border-bottom:1px solid #f3e7c9}
.price-list-poster .pl-hero{display:flex;align-items:center;gap:18px;margin-bottom:14px}
.price-list-poster .pl-logo{width:96px;height:96px;border-radius:20px;overflow:hidden;background:#fff;flex-shrink:0;box-shadow:0 4px 14px rgba(230,162,60,.18);border:1px solid rgba(230,162,60,.2)}
.price-list-poster .pl-logo img{width:100%;height:100%;object-fit:contain;display:block}
.price-list-poster .pl-hero-text{min-width:0;flex:1}
.price-list-poster .pl-title{font-size:34px;font-weight:800;letter-spacing:.1em;line-height:1.15;color:#111827}
.price-list-poster .pl-store{margin-top:8px;font-size:18px;font-weight:700;color:#303133}
.price-list-poster .pl-sub{margin-top:6px;font-size:14px;color:#909399}
.price-list-poster .pl-chips{display:flex;flex-wrap:wrap;gap:8px;margin:0 0 8px}
.price-list-poster .pl-chip{display:inline-block;padding:6px 12px;border-radius:999px;background:rgba(255,255,255,.9);border:1px solid #f0e0c0;color:#606266;font-size:14px;line-height:1.3}
.price-list-poster .pl-addr{font-size:14px;color:#606266;line-height:1.5}
.price-list-poster .price-list-extra{margin:0 16px 14px;padding:12px 14px;border-radius:10px;background:#fff7e6;color:#a16207;font-size:15px;line-height:1.55}
.price-list-poster .pl-body{flex:1 1 auto;padding:0 16px;display:flex;flex-direction:column;justify-content:flex-start}
.price-list-poster .pl-section{margin-top:4px}
.price-list-poster .pl-section+.pl-section{margin-top:18px}
.price-list-poster .pl-section-title{display:flex;align-items:center;gap:10px;margin-bottom:8px}
.price-list-poster .pl-section-title::before,.price-list-poster .pl-section-title::after{content:"";flex:1;height:1px;background:#f0e0c0}
.price-list-poster .pl-section-title span{flex:none;font-size:15px;font-weight:700;color:#e6a23c;letter-spacing:.08em}
.price-list-poster .pl-list{display:flex;flex-direction:column;gap:0}
.price-list-poster .pl-item{display:flex;align-items:flex-start;justify-content:space-between;gap:14px;padding:16px 0;border-bottom:1px solid #f3f4f6}
.price-list-poster .pl-item:last-child{border-bottom:none}
.price-list-poster .pl-item-main{display:flex;align-items:flex-start;gap:12px;min-width:0;flex:1}
.price-list-poster .svc-icon-wrap{width:48px;height:48px;border-radius:14px;background:#fff7e6;display:block;text-align:center;line-height:48px;flex-shrink:0;overflow:hidden}
.price-list-poster .svc-icon{display:inline-block;width:26px;height:26px;vertical-align:middle;border:0}
.price-list-poster .pl-item-body{min-width:0;flex:1}
.price-list-poster .pl-item-name{font-size:18px;font-weight:700;color:#111827;line-height:1.35;word-break:break-word}
.price-list-poster .pl-item-meta{margin-top:4px;font-size:14px;color:#909399}
.price-list-poster .pl-item-desc{margin-top:6px;font-size:14px;color:#606266;line-height:1.5;display:-webkit-box;-webkit-box-orient:vertical;-webkit-line-clamp:2;overflow:hidden;word-break:break-word}
.price-list-poster .pl-item-price{flex-shrink:0;font-size:26px;font-weight:800;color:#e6a23c;font-variant-numeric:tabular-nums;line-height:1.2;padding-top:2px;white-space:nowrap}
.price-list-poster .pl-bottom{margin-top:8px;padding:14px 16px 12px}
.price-list-poster .pl-footer{font-size:13px;color:#909399;line-height:1.55;text-align:center}
.price-list-poster .pl-footer.muted{margin-top:4px;font-size:12px}
.price-list-qr-row{display:flex;justify-content:flex-start;align-items:flex-start;gap:18px;margin:12px 0 0;padding:10px 0 0;border-top:1px dashed #eee}
.price-list-qr-row .qr-item{text-align:center;width:68px}
.price-list-qr-row .qr-item img{width:60px;height:60px;object-fit:contain;display:block;margin:0 auto;border-radius:4px;background:#fafafa}
.price-list-qr-row .qr-label{margin-top:4px;font-size:10px;color:#909399;line-height:1.3}
</style>`)
	b.WriteString(`</div>`)
	return b.String()
}

func formatPosterPrice(price float64) string {
	if price == float64(int64(price)) {
		return fmt.Sprintf("¥%.0f", price)
	}
	return fmt.Sprintf("¥%.2f", price)
}

// compactPosterDesc 海报说明压成单段短文，避免手机上大段挤占版面
func compactPosterDesc(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\n", " ")
	for strings.Contains(s, "  ") {
		s = strings.ReplaceAll(s, "  ", " ")
	}
	const maxRunes = 56
	r := []rune(s)
	if len(r) <= maxRunes {
		return s
	}
	return string(r[:maxRunes]) + "…"
}

// formatServiceDuration 分钟 → 易读时长（价目表表格内省略「约」以节省横向空间）
func formatServiceDuration(minutes int) string {
	if minutes <= 0 {
		return "-"
	}
	if minutes < 60 {
		return fmt.Sprintf("%d分钟", minutes)
	}
	h := minutes / 60
	rest := minutes % 60
	if rest == 0 {
		return fmt.Sprintf("%d小时", h)
	}
	return fmt.Sprintf("%d小时%d分", h, rest)
}
