package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/aarondever/go-gin-template/internal/dto"
	"github.com/aarondever/go-gin-template/internal/model"
	"github.com/aarondever/go-gin-template/internal/service"
	e "github.com/aarondever/go-gin-template/internal/shared/errors"
	"github.com/aarondever/go-gin-template/pkg/response"
	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	service service.UserService
}

func NewUserHandler(svc service.UserService) *UserHandler {
	return &UserHandler{service: svc}
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
	var req dto.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request", err)
		return
	}

	user, err := h.service.CreateUser(c.Request.Context(), &model.User{
		Name:  req.Name,
		Email: req.Email,
	})
	if err != nil {
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

	user, err := h.service.GetUserByID(c.Request.Context(), userID)
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
	var req dto.GetUserListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request", err)
		return
	}

	users, total, err := h.service.GetUserList(c.Request.Context(), dto.UserListFilter{
		Name:   req.Name,
		Limit:  req.GetLimit(),
		Offset: req.GetOffset(),
	})
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to list users", err)
		return
	}

	req.Pagination.SetTotal(total)
	response.Success(c, http.StatusOK, "users retrieved successfully", &dto.UserListResponse{
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

	var req dto.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request", err)
		return
	}

	// Check if user exists
	_, err = h.service.GetUserByID(c.Request.Context(), userID)
	if err != nil {
		if errors.Is(err, e.ErrNotFound) {
			response.Error(c, http.StatusNotFound, "user not found", err)
			return
		}

		response.Error(c, http.StatusInternalServerError, "failed to get user", err)
		return
	}

	// Update user
	user, err := h.service.UpdateUser(c.Request.Context(), &model.User{
		ID:    userID,
		Name:  req.Name,
		Email: req.Email,
	})
	if err != nil {
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

	// Check if user exists
	_, err = h.service.GetUserByID(c.Request.Context(), userID)
	if err != nil {
		if errors.Is(err, e.ErrNotFound) {
			response.Error(c, http.StatusNotFound, "user not found", err)
			return
		}

		response.Error(c, http.StatusInternalServerError, "failed to get user", err)
		return
	}

	// Delete user
	if err := h.service.DeleteUser(c.Request.Context(), userID); err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to delete user", err)
		return
	}

	response.Success(c, http.StatusNoContent, "user deleted successfully", nil)
}
