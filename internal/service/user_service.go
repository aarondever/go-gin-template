package service

import (
	"context"
	"errors"

	e "github.com/aarondever/go-gin-template/errors"
	"github.com/aarondever/go-gin-template/internal/dto"
	"github.com/aarondever/go-gin-template/internal/model"
	"github.com/aarondever/go-gin-template/internal/repository"
	"github.com/aarondever/go-gin-template/pkg/logger"
	"github.com/aarondever/go-gin-template/pkg/utils"
	"gorm.io/gorm"
)

type UserService interface {
	CreateUser(ctx context.Context, user *model.User) (*model.User, error)
	GetUserByID(ctx context.Context, userID int64) (*model.User, error)
	GetUserList(ctx context.Context, f *dto.UserListFilter) ([]*model.User, int64, error)
	UpdateUser(ctx context.Context, user *model.User) (*model.User, error)
	DeleteUser(ctx context.Context, userID int64) error
}

type userService struct {
	repo repository.UserRepository
}

func NewUserService(repo repository.UserRepository) UserService {
	return &userService{repo: repo}
}

func (s *userService) CreateUser(ctx context.Context, user *model.User) (*model.User, error) {
	utils.TrimStruct(user)
	if err := s.repo.CreateUser(ctx, user); err != nil {
		logger.Error("failed to create user", "error", err)
		return nil, err
	}

	return user, nil
}

func (s *userService) GetUserByID(ctx context.Context, userID int64) (*model.User, error) {
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			logger.Error("user not found", "user_id", userID)
			return nil, e.ErrNotFound
		}

		logger.Error("failed to get user by id", "error", err)
		return nil, err
	}

	return user, nil
}

func (s *userService) GetUserList(ctx context.Context, f *dto.UserListFilter) ([]*model.User, int64, error) {
	utils.TrimStruct(f)
	users, total, err := s.repo.GetUserList(ctx, f)
	if err != nil {
		logger.Error("failed to get user list", "error", err)
		return nil, 0, err
	}

	return users, total, nil
}

func (s *userService) UpdateUser(ctx context.Context, user *model.User) (*model.User, error) {
	utils.TrimStruct(user)
	if err := s.repo.UpdateUser(ctx, user); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			logger.Error("user not found", "user_id", user.ID)
			return nil, e.ErrNotFound
		}

		logger.Error("failed to update user", "error", err)
		return nil, err
	}

	return user, nil
}

func (s *userService) DeleteUser(ctx context.Context, userID int64) error {
	if err := s.repo.DeleteUser(ctx, userID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			logger.Error("user not found", "user_id", userID)
			return e.ErrNotFound
		}

		logger.Error("failed to delete user", "error", err)
		return err
	}

	return nil
}
