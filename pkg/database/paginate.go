package database

import (
	"github.com/aarondever/go-gin-template/pkg/pagination"
	"gorm.io/gorm"
)

func Paginate(p *pagination.Pagination) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if p == nil {
			return db
		}
		db.Session(&gorm.Session{}).Count(&p.Total)
		return db.Offset(p.Offset()).Limit(p.Limit())
	}
}
