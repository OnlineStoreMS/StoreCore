package service

import (
	"errors"
	"html"
	"strings"

	"storecore/internal/dto"
	"storecore/internal/model"
	"storecore/internal/repo"

	"gorm.io/gorm"
)

type ReceiptTemplateService struct {
	repos    *repo.Repos
	tenantID uint64
}

func NewReceiptTemplateService(repos *repo.Repos) *ReceiptTemplateService {
	return &ReceiptTemplateService{repos: repos}
}

func (s *ReceiptTemplateService) ForTenant(tenantID uint64) *ReceiptTemplateService {
	return &ReceiptTemplateService{repos: s.repos, tenantID: repo.NormalizeTenantID(tenantID)}
}

func (s *ReceiptTemplateService) List(storeID uint64, receiptType string, page, pageSize int) ([]model.ReceiptTemplate, int64, error) {
	return s.repos.ReceiptTpl.ForTenant(s.tenantID).List(storeID, receiptType, page, pageSize)
}

func (s *ReceiptTemplateService) Get(id uint64) (*model.ReceiptTemplate, error) {
	item, err := s.repos.ReceiptTpl.ForTenant(s.tenantID).GetByID(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return item, err
}

func (s *ReceiptTemplateService) Create(in *dto.ReceiptTemplateDTO) (*model.ReceiptTemplate, error) {
	if strings.TrimSpace(in.Name) == "" {
		return nil, ErrBadRequest
	}
	item := applyReceiptTemplateDTO(&model.ReceiptTemplate{}, in)
	if err := s.repos.ReceiptTpl.ForTenant(s.tenantID).Create(item); err != nil {
		return nil, err
	}
	return item, nil
}

func (s *ReceiptTemplateService) Update(id uint64, in *dto.ReceiptTemplateDTO) (*model.ReceiptTemplate, error) {
	r := s.repos.ReceiptTpl.ForTenant(s.tenantID)
	item, err := r.GetByID(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	item = applyReceiptTemplateDTO(item, in)
	if err := r.Update(item); err != nil {
		return nil, err
	}
	return item, nil
}

func (s *ReceiptTemplateService) Delete(id uint64) error {
	r := s.repos.ReceiptTpl.ForTenant(s.tenantID)
	if _, err := r.GetByID(id); errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound
	} else if err != nil {
		return err
	}
	return r.Delete(id)
}

func applyReceiptTemplateDTO(item *model.ReceiptTemplate, in *dto.ReceiptTemplateDTO) *model.ReceiptTemplate {
	item.StoreID = in.StoreID
	item.Name = strings.TrimSpace(in.Name)
	item.ReceiptType = defaultReceiptType(in.ReceiptType)
	item.HeaderTitle = strings.TrimSpace(in.HeaderTitle)
	item.HeaderSubtitle = strings.TrimSpace(in.HeaderSubtitle)
	item.HeaderExtra = strings.TrimSpace(in.HeaderExtra)
	item.FooterThanks = strings.TrimSpace(in.FooterThanks)
	item.FooterExtra = strings.TrimSpace(in.FooterExtra)
	item.IsDefault = in.IsDefault
	applyBoolPtr(&item.ShowSkuPic, in.ShowSkuPic, item.ID == 0, true)
	applyBoolPtr(&item.ShowStorePhone, in.ShowStorePhone, item.ID == 0, true)
	applyBoolPtr(&item.ShowStoreAddress, in.ShowStoreAddress, item.ID == 0, true)
	applyBoolPtr(&item.ShowBusinessHours, in.ShowBusinessHours, item.ID == 0, true)
	applyBoolPtr(&item.ShowBrandLogo, in.ShowBrandLogo, item.ID == 0, true)
	applyBoolPtr(&item.ShowCoverPic, in.ShowCoverPic, item.ID == 0, false)
	applyBoolPtr(&item.ShowGuideText, in.ShowGuideText, item.ID == 0, false)
	applyBoolPtr(&item.ShowMapLabel, in.ShowMapLabel, item.ID == 0, false)
	// 价目表默认展示说明、时长、小程序码、团购码
	descDefault := item.ReceiptType == "price_list"
	applyBoolPtr(&item.ShowDescription, in.ShowDescription, item.ID == 0, descDefault)
	applyBoolPtr(&item.ShowDuration, in.ShowDuration, item.ID == 0, descDefault)
	applyBoolPtr(&item.ShowWechatMpQr, in.ShowWechatMpQr, item.ID == 0, descDefault)
	applyBoolPtr(&item.ShowGroupBuyQr, in.ShowGroupBuyQr, item.ID == 0, descDefault)
	if in.Status != 0 {
		item.Status = in.Status
	} else if item.ID == 0 {
		item.Status = 1
	}
	return item
}

func applyBoolPtr(dst *bool, src *bool, isCreate bool, createDefault bool) {
	if src != nil {
		*dst = *src
	} else if isCreate {
		*dst = createDefault
	}
}

func defaultReceiptTemplate() *model.ReceiptTemplate {
	return &model.ReceiptTemplate{
		Name:              "默认小票",
		ReceiptType:       "small",
		HeaderTitle:       "门店收银小票",
		HeaderSubtitle:    "欢迎光临",
		FooterThanks:      "谢谢惠顾，欢迎再次光临",
		FooterExtra:       "商品如有质量问题，请凭小票在7日内联系门店处理",
		ShowSkuPic:        true,
		ShowStorePhone:    true,
		ShowStoreAddress:  true,
		ShowBusinessHours: true,
		ShowBrandLogo:     true,
		ShowCoverPic:      false,
		ShowGuideText:     false,
		ShowMapLabel:      false,
		IsDefault:         true,
		Status:            1,
	}
}

func escapeReceipt(s string) string {
	return html.EscapeString(s)
}

func nl2br(s string) string {
	return strings.ReplaceAll(escapeReceipt(s), "\n", "<br/>")
}
