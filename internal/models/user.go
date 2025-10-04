package models

import (
	"github.com/google/uuid"
)

type UserParams struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type UpdateUserParams struct {
	ID       uuid.UUID
	Username string `json:"username"`
}
