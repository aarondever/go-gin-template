package model

import (
	sharedmodel "github.com/aarondever/go-gin-template/internal/shared/model"
)

type User struct {
	ID    int64   `gorm:"primaryKey;autoIncrement" json:"id"`
	Name  string  `gorm:"not null" json:"name"`
	Email *string `gorm:"uniqueIndex" json:"email"`
	sharedmodel.BaseModel
}
