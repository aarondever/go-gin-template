package repository

import (
	"github.com/aarondever/go-gin-template/pkg/pagination"
	"gorm.io/gorm"
)

func paginate(p *pagination.Pagination) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if p == nil {
			return db
		}
		var total int64
		db.Session(&gorm.Session{}).Count(&total)
		p.Total = total
		return db.Offset(p.GetOffset()).Limit(p.GetLimit())
	}
}
