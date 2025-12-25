package user

import (
	"errors"
	"net/http"
	"strconv"

	e "github.com/aarondever/go-gin-template/internal/shared/errors"
	"github.com/aarondever/go-gin-template/pkg/response"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(router *gin.RouterGroup) {
	users := router.Group("/users")
	{
		users.POST("", h.CreateUser)
		users.GET("/:id", h.GetUserByID)
		users.GET("", h.ListUsers)
		users.PUT("/:id", h.UpdateUser)
		users.DELETE("/:id", h.DeleteUser)
	}
}

func (h *Handler) CreateUser(c *gin.Context) {
	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request", err)
		return
	}

	user, err := h.service.CreateUser(c.Request.Context(), req)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to create user", err)
		return
	}

	response.Success(c, http.StatusCreated, "user created successfully", user)
}

func (h *Handler) GetUserByID(c *gin.Context) {
	userID, err := strconv.ParseInt(c.Param("id"), 10, 64)
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

func (h *Handler) ListUsers(c *gin.Context) {
	var req ListUsersRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request", err)
		return
	}

	users, p, err := h.service.ListUsers(c.Request.Context(), req)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to list users", err)
		return
	}

	response.Success(c, http.StatusOK, "users retrieved successfully", &ListUsersResponse{
		Users:      users,
		Pagination: p,
	})
}

func (h *Handler) UpdateUser(c *gin.Context) {
	userID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid id", err)
		return
	}

	var req UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request", err)
		return
	}

	user, err := h.service.UpdateUser(c.Request.Context(), userID, req)
	if err != nil {
		if errors.Is(err, e.ErrNotFound) {
			response.Error(c, http.StatusNotFound, "user not found", err)
			return
		}

		response.Error(c, http.StatusInternalServerError, "failed to update user", err)
		return
	}

	response.Success(c, http.StatusOK, "user updated successfully", user)
}

func (h *Handler) DeleteUser(c *gin.Context) {
	userID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid id", err)
		return
	}

	if err := h.service.DeleteUser(c.Request.Context(), userID); err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to delete user", err)
		return
	}

	response.Success(c, http.StatusNoContent, "user deleted successfully", nil)
}
