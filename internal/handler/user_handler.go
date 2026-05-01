package handler

import (
	"errors"
	"net/http"
	"strconv"

	e "github.com/aarondever/go-gin-template/errors"
	"github.com/aarondever/go-gin-template/internal/model"
	"github.com/aarondever/go-gin-template/internal/service"
	"github.com/aarondever/go-gin-template/pkg/pagination"
	"github.com/aarondever/go-gin-template/pkg/response"
	"github.com/gin-gonic/gin"
)

type createUserRequest struct {
	Name  string  `json:"name" binding:"required"`
	Email *string `json:"email" binding:"omitzero,email"`
}

type updateUserRequest struct {
	Name  string  `json:"name"`
	Email *string `json:"email" binding:"omitzero,email"`
}

type getUserListRequest struct {
	Name  string `form:"name"`
	Email string `form:"email"`
	pagination.Pagination
}

type userListResponse struct {
	Users []*model.User `json:"users"`
	pagination.Pagination
}

type UserHandler struct {
	svc service.UserService
}

func NewUserHandler(svc service.UserService) *UserHandler {
	return &UserHandler{svc: svc}
}

func (h *UserHandler) RegisterRoutes(router *gin.RouterGroup) {
	users := router.Group("/users")
	{
		users.POST("", h.CreateUser)
		users.GET("/:userID", h.GetUserByID)
		users.GET("", h.GetUserList)
		users.PUT("/:userID", h.UpdateUser)
		users.DELETE("/:userID", h.DeleteUser)
	}
}

func (h *UserHandler) CreateUser(c *gin.Context) {
	var req createUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request", err)
		return
	}

	user, err := h.svc.CreateUser(c.Request.Context(), &model.User{
		Name:  req.Name,
		Email: req.Email,
	})
	if err != nil {
		var valErr *e.ValidationError
		if errors.As(err, &valErr) {
			response.Error(c, http.StatusBadRequest, "validation failed", valErr.Err)
			return
		}
		if errors.Is(err, e.ErrConflict) {
			response.Error(c, http.StatusConflict, "user already exists", err)
			return
		}
		response.Error(c, http.StatusInternalServerError, "failed to create user", err)
		return
	}

	response.Success(c, http.StatusCreated, "user created successfully", user)
}

func (h *UserHandler) GetUserByID(c *gin.Context) {
	userID, err := strconv.ParseInt(c.Param("userID"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid id", err)
		return
	}

	user, err := h.svc.GetUserByID(c.Request.Context(), userID)
	if err != nil {
		if errors.Is(err, e.ErrNotFound) {
			response.Error(c, http.StatusNotFound, "user not found", err)
			return
		}
		response.Error(c, http.StatusInternalServerError, "failed to get user", err)
		return
	}

	response.Success(c, http.StatusOK, "user retrieved successfully", user)
}

func (h *UserHandler) GetUserList(c *gin.Context) {
	var req getUserListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request", err)
		return
	}

	users, err := h.svc.GetUserList(c.Request.Context(), &req.Pagination, &model.UserListFilter{
		Name:  req.Name,
		Email: req.Email,
	})
	if err != nil {
		var valErr *e.ValidationError
		if errors.As(err, &valErr) {
			response.Error(c, http.StatusBadRequest, "validation failed", valErr.Err)
			return
		}
		response.Error(c, http.StatusInternalServerError, "failed to list users", err)
		return
	}

	response.Success(c, http.StatusOK, "users retrieved successfully", &userListResponse{
		Users:      users,
		Pagination: req.Pagination,
	})
}

func (h *UserHandler) UpdateUser(c *gin.Context) {
	userID, err := strconv.ParseInt(c.Param("userID"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid id", err)
		return
	}

	var req updateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request", err)
		return
	}

	user, err := h.svc.UpdateUser(c.Request.Context(), &model.User{
		ID:    userID,
		Name:  req.Name,
		Email: req.Email,
	})
	if err != nil {
		var valErr *e.ValidationError
		if errors.As(err, &valErr) {
			response.Error(c, http.StatusBadRequest, "validation failed", valErr.Err)
			return
		}
		if errors.Is(err, e.ErrNotFound) {
			response.Error(c, http.StatusNotFound, "user not found", err)
			return
		}
		if errors.Is(err, e.ErrConflict) {
			response.Error(c, http.StatusConflict, "user already exists", err)
			return
		}
		response.Error(c, http.StatusInternalServerError, "failed to update user", err)
		return
	}

	response.Success(c, http.StatusOK, "user updated successfully", user)
}

func (h *UserHandler) DeleteUser(c *gin.Context) {
	userID, err := strconv.ParseInt(c.Param("userID"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid id", err)
		return
	}

	if err := h.svc.DeleteUser(c.Request.Context(), userID); err != nil {
		if errors.Is(err, e.ErrNotFound) {
			response.Error(c, http.StatusNotFound, "user not found", err)
			return
		}
		response.Error(c, http.StatusInternalServerError, "failed to delete user", err)
		return
	}

	response.Success(c, http.StatusNoContent, "user deleted successfully", nil)
}
