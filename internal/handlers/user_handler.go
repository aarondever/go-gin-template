package handlers

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/aarondever/go-gin-template/internal/database"
	"github.com/aarondever/go-gin-template/internal/models"
	"github.com/aarondever/go-gin-template/internal/services"
	"github.com/aarondever/go-gin-template/internal/utils"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type UserHandler struct {
	service *services.UserService
}

func NewUserHandler(service *services.UserService) *UserHandler {
	return &UserHandler{
		service: service,
	}
}

func (handler *UserHandler) SetupRouters(router *gin.Engine) {
	userV1 := router.Group("/users/v1")
	userV1.GET("/", handler.GetUsers)
	userV1.GET("/:id", handler.GetUser)
	userV1.POST("/", handler.CreateUser)
	userV1.PUT("/:id", handler.UpdateUser)
}

func (handler *UserHandler) GetUsers(c *gin.Context) {
	// Parse query parameters
	page := utils.DefaultToInt32(c.Query("page"), 1)
	pageSize := utils.DefaultToInt32(c.Query("page_size"), 10)

	// Fetch all users from the service layer
	userPagination, err := handler.service.GetUserPagination(c.Request.Context(), database.GetUsersParams{
		Limit:  pageSize,
		Offset: (page - 1) * pageSize,
	})
	if err != nil {
		slog.Error("Failed to get users", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	userPagination.CurrentPage = page
	userPagination.PageSize = pageSize

	c.JSON(http.StatusOK, userPagination)
}

func (handler *UserHandler) GetUser(c *gin.Context) {
	// Parse and validate user ID from URL parameter
	id := c.Param("id")
	userID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	// Fetch user from the service layer
	user, err := handler.service.GetUser(c.Request.Context(), userID)
	if err != nil {
		slog.Error("Failed to get user", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	if user == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, user)
}

func (handler *UserHandler) CreateUser(c *gin.Context) {
	// Parse and validate request body
	var params models.UserParams
	if err := c.ShouldBindJSON(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Create user via the service layer
	user, err := handler.service.CreateUser(c.Request.Context(), params)
	if err != nil {
		if errors.Is(err, models.ErrDuplicateUser) {
			c.JSON(http.StatusConflict, gin.H{"error": "Username already exists"})
			return
		}

		slog.Error("Failed to create user", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	c.JSON(http.StatusCreated, user)
}

func (handler *UserHandler) UpdateUser(c *gin.Context) {
	// Parse and validate user ID from URL parameter
	id := c.Param("id")
	userID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	// Parse and validate request body
	var params models.UpdateUserParams
	if err = c.ShouldBindJSON(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	params.ID = userID

	// Update user via the service layer
	user, err := handler.service.UpdateUser(c.Request.Context(), params)
	if err != nil {
		if errors.Is(err, models.ErrDuplicateUser) {
			c.JSON(http.StatusConflict, gin.H{"error": "Username already exists"})
			return
		}

		slog.Error("Failed to update user", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	if user == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, user)
}
