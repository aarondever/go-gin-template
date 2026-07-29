package model

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	ID        uint64         `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	Name      string         `json:"name" gorm:"column:name;not null" validate:"required"`
	Email     *string        `json:"email" gorm:"column:email;uniqueIndex" validate:"omitzero,email"`
	CreatedAt time.Time      `json:"created_at" gorm:"column:created_at"`
	UpdatedAt time.Time      `json:"updated_at" gorm:"column:updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"column:deleted_at;index"`
}

type UserListFilter struct {
	Name  string
	Email string `validate:"omitzero,email"`
}
