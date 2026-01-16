package dto

import (
	"github.com/aarondever/go-gin-template/internal/model"
	"github.com/aarondever/go-gin-template/pkg/pagination"
)

type CreateUserRequest struct {
	Name  string  `json:"name" binding:"required"`
	Email *string `json:"email" binding:"omitempty,email"`
}

type UpdateUserRequest struct {
	Name  string  `json:"name" binding:"omitempty"`
	Email *string `json:"email" binding:"omitempty,email"`
}

type GetUserListRequest struct {
	Name string `form:"name"`
	pagination.Pagination
}

type UserListResponse struct {
	Users []*model.User `json:"users"`
	pagination.Pagination
}

// Filters
type UserListFilter struct {
	Name   string
	Limit  int
	Offset int
}
