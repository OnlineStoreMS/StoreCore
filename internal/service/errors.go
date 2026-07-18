package service

import "errors"

var (
	ErrNotFound          = errors.New("记录不存在")
	ErrDuplicateCode     = errors.New("编码已存在")
	ErrBadRequest        = errors.New("请求参数无效")
	ErrInvalidStatus     = errors.New("状态不允许此操作")
	ErrInsufficientStock = errors.New("门店库存不足")
)
