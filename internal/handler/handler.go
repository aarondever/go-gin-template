package handler

import (
	"net/http"
	"strconv"

	"github.com/aarondever/go-gin-template/internal/model"
	p "github.com/aarondever/go-gin-template/internal/pagination"
	"github.com/aarondever/go-gin-template/internal/response"
	"github.com/aarondever/go-gin-template/internal/service"
	"github.com/aarondever/go-gin-template/internal/util"
	"github.com/aarondever/go-gin-template/internal/validation"
	"github.com/gin-gonic/gin"
)

type createUserRequest struct {
	Name  string  `json:"name" validate:"required"`
	Email *string `json:"email" validate:"omitempty,email"`
}

type updateUserRequest struct {
	Name  string  `json:"name"`
	Email *string `json:"email" validate:"omitempty,email"`
}

type getUserListRequest struct {
	Name  string `form:"name"`
	Email string `form:"email"`
	p.Pagination
}

type userListResponse struct {
	Users []*model.User `json:"users"`
	p.Pagination
}

type Handler struct {
	svc service.Service
}

func New(svc service.Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Create(c *gin.Context) {
	var req createUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(err)
		return
	}

	if err := validation.ValidateStruct(util.TrimStructStr(req)); err != nil {
		c.Error(err)
		return
	}

	user, err := h.svc.Create(c.Request.Context(), &model.User{
		Name:  req.Name,
		Email: req.Email,
	})
	if err != nil {
		c.Error(err)
		return
	}

	response.JSON(c, http.StatusCreated, user)
}

func (h *Handler) GetByID(c *gin.Context) {
	userID, err := strconv.ParseUint(c.Param("userID"), 10, 64)
	if err != nil {
		c.Error(err)
		return
	}

	user, err := h.svc.GetByID(c.Request.Context(), userID)
	if err != nil {
		c.Error(err)
		return
	}

	response.JSON(c, http.StatusOK, user)
}

func (h *Handler) GetList(c *gin.Context) {
	var req getUserListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.Error(err)
		return
	}

	if err := validation.ValidateStruct(util.TrimStructStr(req)); err != nil {
		c.Error(err)
		return
	}

	users, err := h.svc.GetList(c.Request.Context(), &req.Pagination, &model.UserListFilter{
		Name:  req.Name,
		Email: req.Email,
	})
	if err != nil {
		c.Error(err)
		return
	}

	response.JSON(c, http.StatusOK, &userListResponse{
		Users:      users,
		Pagination: req.Pagination,
	})
}

func (h *Handler) Update(c *gin.Context) {
	userID, err := strconv.ParseUint(c.Param("userID"), 10, 64)
	if err != nil {
		c.Error(err)
		return
	}

	var req updateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(err)
		return
	}

	if err := validation.ValidateStruct(util.TrimStructStr(req)); err != nil {
		c.Error(err)
		return
	}

	user, err := h.svc.Update(c.Request.Context(), &model.User{
		ID:    userID,
		Name:  req.Name,
		Email: req.Email,
	})
	if err != nil {
		c.Error(err)
		return
	}

	response.JSON(c, http.StatusOK, user)
}

func (h *Handler) Delete(c *gin.Context) {
	userID, err := strconv.ParseUint(c.Param("userID"), 10, 64)
	if err != nil {
		c.Error(err)
		return
	}

	if err := h.svc.Delete(c.Request.Context(), userID); err != nil {
		c.Error(err)
		return
	}

	response.JSON(c, http.StatusNoContent, nil)
}
