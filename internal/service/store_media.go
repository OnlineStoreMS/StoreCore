package service

import "storecore/internal/model"

func applyStorePublicURLs(item *model.Store, resolve func(string) string) {
	if item == nil || resolve == nil {
		return
	}
	item.BrandLogo = resolve(item.BrandLogo)
	item.WechatMpQrCode = resolve(item.WechatMpQrCode)
	item.GroupBuyQrCode = resolve(item.GroupBuyQrCode)
	item.CoverPic = resolve(item.CoverPic)
	item.Photos = resolveURLList(item.Photos, resolve)
	item.GuidePics = resolveURLList(item.GuidePics, resolve)
}

func applyStoreListPublicURLs(list []model.Store, resolve func(string) string) {
	if resolve == nil {
		return
	}
	for i := range list {
		applyStorePublicURLs(&list[i], resolve)
	}
}

func resolveURLList(list []string, resolve func(string) string) []string {
	if len(list) == 0 || resolve == nil {
		return list
	}
	out := make([]string, len(list))
	for i, u := range list {
		out[i] = resolve(u)
	}
	return out
}
