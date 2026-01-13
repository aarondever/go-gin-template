package user

import "github.com/aarondever/go-gin-template/pkg/pagination"

type CreateUserRequest struct {
	Name  string  `json:"name" binding:"required"`
	Email *string `json:"email" binding:"omitempty,email"`
}

type UpdateUserRequest struct {
	Name  string  `json:"name" binding:"omitempty"`
	Email *string `json:"email" binding:"omitempty,email"`
}

type UserResponse struct {
	ID    int64   `json:"id"`
	Name  string  `json:"name"`
	Email *string `json:"email"`
}

func UserToResponse(user *User) *UserResponse {
	return &UserResponse{
		ID:    user.ID,
		Name:  user.Name,
		Email: user.Email,
	}
}

type GetUserListRequest struct {
	Name string `form:"name"`
	pagination.Pagination
}

type UserListResponse struct {
	Users []*UserResponse `json:"users"`
	pagination.Pagination
}

func UserListToResponse(users []*User) []*UserResponse {
	responses := make([]*UserResponse, len(users))
	for i, user := range users {
		responses[i] = UserToResponse(user)
	}
	return responses
}
