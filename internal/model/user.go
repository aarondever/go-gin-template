package model

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	ID        int64          `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Name      string         `gorm:"column:name;not null" json:"name" validate:"required"`
	Email     *string        `gorm:"column:email;uniqueIndex" json:"email" validate:"omitzero,email"`
	CreatedAt time.Time      `gorm:"column:created_at" json:"created_at"`
	UpdatedAt time.Time      `gorm:"column:updated_at" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"column:deleted_at;index" json:"-"`
}

type UserListFilter struct {
	Name  string
	Email string `validate:"omitzero,email"`
}
