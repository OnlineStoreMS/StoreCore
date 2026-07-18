package repo

import (
	"strings"

	"gorm.io/gorm"
)

func applyEq(q *gorm.DB, column, value string) *gorm.DB {
	v := strings.TrimSpace(value)
	if v == "" {
		return q
	}
	return q.Where(column+" = ?", v)
}

func applyOrderKeyword(q *gorm.DB, keyword string) *gorm.DB {
	kw := strings.TrimSpace(keyword)
	if kw == "" {
		return q
	}
	like := "%" + kw + "%"
	return q.Where("order_no LIKE ? OR customer_name LIKE ? OR customer_phone LIKE ?", like, like, like)
}
