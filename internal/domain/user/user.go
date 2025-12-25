package user

import (
	"github.com/aarondever/go-gin-template/internal/shared/model"
	"github.com/aarondever/go-gin-template/pkg/pagination"
)

type User struct {
	ID    int64   `json:"id" gorm:"primaryKey;autoIncrement"`
	Name  string  `json:"name" gorm:"not null"`
	Email *string `json:"email" gorm:"uniqueIndex"`
	model.BaseModel
}

type CreateUserRequest struct {
	Name  string  `json:"name" binding:"required"`
	Email *string `json:"email" binding:"omitempty,email"`
}

type UpdateUserRequest struct {
	Name  string  `json:"name" binding:"omitempty"`
	Email *string `json:"email" binding:"omitempty,email"`
}

type UserSearchFields struct {
	Name string `form:"name"`
}

type ListUsersRequest struct {
	UserSearchFields
	pagination.Pagination
}

type ListUsersFilter struct {
	UserSearchFields
	pagination.PaginationFilter
}

type ListUsersResponse struct {
	Users []*User `json:"users"`
	pagination.Pagination
}
