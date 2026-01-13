package user

import (
	"github.com/aarondever/go-gin-template/internal/shared/model"
)

type User struct {
	model.BaseModel
	ID    int64   `gorm:"primaryKey;autoIncrement"`
	Name  string  `gorm:"not null"`
	Email *string `gorm:"uniqueIndex"`
}

type UserListFilter struct {
	Name   string
	Limit  int
	Offset int
}
