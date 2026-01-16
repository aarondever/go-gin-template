package service

import (
	"context"

	"github.com/aarondever/go-gin-template/internal/dto"
	"github.com/aarondever/go-gin-template/internal/model"
	"github.com/aarondever/go-gin-template/internal/repository"
	"github.com/aarondever/go-gin-template/internal/shared/database"
	e "github.com/aarondever/go-gin-template/internal/shared/errors"
	"github.com/aarondever/go-gin-template/pkg/logger"
)

type UserService interface {
	CreateUser(ctx context.Context, user *model.User) (*model.User, error)
	GetUserByID(ctx context.Context, userID int64) (*model.User, error)
	GetUserList(ctx context.Context, filter dto.UserListFilter) ([]*model.User, int64, error)
	UpdateUser(ctx context.Context, user *model.User) (*model.User, error)
	DeleteUser(ctx context.Context, userID int64) error
}

type userService struct {
	repo repository.UserRepository
	db   *database.Database
}

func NewUserService(repo repository.UserRepository, db *database.Database) UserService {
	return &userService{repo: repo, db: db}
}

func (s *userService) CreateUser(ctx context.Context, user *model.User) (*model.User, error) {
	if err := s.repo.CreateUser(ctx, s.db.DB, user); err != nil {
		logger.Error("failed to create user", "error", err)
		return nil, err
	}

	return user, nil
}

func (s *userService) GetUserByID(ctx context.Context, userID int64) (*model.User, error) {
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		logger.Error("failed to get user by id", "error", err)
		return nil, err
	}

	if user == nil {
		logger.Error("user not found", "user_id", userID)
		return nil, e.ErrNotFound
	}

	return user, nil
}

func (s *userService) GetUserList(ctx context.Context, filters dto.UserListFilter) ([]*model.User, int64, error) {
	users, total, err := s.repo.GetUserList(ctx, filters)
	if err != nil {
		logger.Error("failed to get user list", "error", err)
		return nil, 0, err
	}

	return users, total, nil
}

func (s *userService) UpdateUser(ctx context.Context, user *model.User) (*model.User, error) {
	if err := s.repo.UpdateUser(ctx, s.db.DB, user); err != nil {
		logger.Error("failed to update user", "error", err)
		return nil, err
	}

	return user, nil
}

func (s *userService) DeleteUser(ctx context.Context, userID int64) error {
	if err := s.repo.DeleteUser(ctx, s.db.DB, userID); err != nil {
		logger.Error("failed to delete user", "error", err)
		return err
	}

	return nil
}
