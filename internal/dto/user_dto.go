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
	Name  string  `json:"name"`
	Email *string `json:"email" binding:"omitempty,email"`
}

// User List
type GetUserListRequest struct {
	Name  string `form:"name"`
	Email string `form:"email"`
	pagination.Pagination
}

type UserListResponse struct {
	Users []*model.User `json:"users"`
	pagination.Pagination
}

type UserListFilter struct {
	Name   string
	Email  string
	Limit  int
	Offset int
}
