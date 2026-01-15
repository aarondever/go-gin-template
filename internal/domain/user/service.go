package user

import (
	"context"

	"github.com/aarondever/go-gin-template/internal/shared/database"
	e "github.com/aarondever/go-gin-template/internal/shared/errors"
	"github.com/aarondever/go-gin-template/pkg/logger"
)

type Service interface {
	CreateUser(ctx context.Context, user *User) (*User, error)
	GetUserByID(ctx context.Context, userID int64) (*User, error)
	GetUserList(ctx context.Context, filter UserListFilter) ([]*User, int64, error)
	UpdateUser(ctx context.Context, user *User) (*User, error)
	DeleteUser(ctx context.Context, userID int64) error
}

type service struct {
	repo Repository
	db   *database.Database
}

func NewService(repo Repository, db *database.Database) Service {
	return &service{repo: repo, db: db}
}

func (s *service) CreateUser(ctx context.Context, user *User) (*User, error) {
	if err := s.repo.CreateUser(ctx, s.db.DB, user); err != nil {
		logger.Error("failed to create user", "error", err)
		return nil, err
	}

	return user, nil
}

func (s *service) GetUserByID(ctx context.Context, userID int64) (*User, error) {
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

func (s *service) GetUserList(ctx context.Context, filters UserListFilter) ([]*User, int64, error) {
	users, total, err := s.repo.GetUserList(ctx, filters)
	if err != nil {
		logger.Error("failed to get user list", "error", err)
		return nil, 0, err
	}

	return users, total, nil
}

func (s *service) UpdateUser(ctx context.Context, user *User) (*User, error) {
	if err := s.repo.UpdateUser(ctx, s.db.DB, user); err != nil {
		logger.Error("failed to update user", "error", err)
		return nil, err
	}

	return user, nil
}

func (s *service) DeleteUser(ctx context.Context, userID int64) error {
	if err := s.repo.DeleteUser(ctx, s.db.DB, userID); err != nil {
		logger.Error("failed to delete user", "error", err)
		return err
	}

	return nil
}
