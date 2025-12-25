package user

import (
	"context"

	"github.com/aarondever/go-gin-template/internal/shared/database"
	e "github.com/aarondever/go-gin-template/internal/shared/errors"
	"github.com/aarondever/go-gin-template/pkg/logger"
	"github.com/aarondever/go-gin-template/pkg/pagination"
)

type Service interface {
	CreateUser(ctx context.Context, req CreateUserRequest) (*User, error)
	GetUserByID(ctx context.Context, id int64) (*User, error)
	ListUsers(ctx context.Context, req ListUsersRequest) ([]*User, pagination.Pagination, error)
	UpdateUser(ctx context.Context, id int64, req UpdateUserRequest) (*User, error)
	DeleteUser(ctx context.Context, id int64) error
}

type service struct {
	db   *database.Database
	repo Repository
}

func NewService(db *database.Database, repo Repository) Service {
	return &service{db: db, repo: repo}
}

func (s *service) CreateUser(ctx context.Context, req CreateUserRequest) (*User, error) {
	user := &User{
		Email: req.Email,
		Name:  req.Name,
	}

	if err := s.repo.CreateUser(ctx, s.db.DB, user); err != nil {
		logger.Error("failed to create user", "error", err)
		return nil, err
	}

	return user, nil
}

func (s *service) GetUserByID(ctx context.Context, id int64) (*User, error) {
	user, err := s.repo.GetUserByID(ctx, id)
	if err != nil {
		logger.Error("failed to get user by id", "error", err)
		return nil, err
	}

	if user == nil {
		logger.Error("user not found", "user_id", id)
		return nil, e.ErrNotFound
	}

	return user, nil
}

func (s *service) ListUsers(ctx context.Context, req ListUsersRequest) ([]*User, pagination.Pagination, error) {
	var f ListUsersFilter
	f.Limit = req.GetLimit()
	f.Offset = req.GetOffset()
	f.Name = req.Name

	var p pagination.Pagination

	users, total, err := s.repo.ListUsers(ctx, f)
	if err != nil {
		logger.Error("failed to list users", "error", err)
		return nil, p, err
	}

	p.Page = req.Page
	p.PageSize = req.PageSize
	p.SetTotal(total)

	return users, p, nil
}

func (s *service) UpdateUser(ctx context.Context, id int64, req UpdateUserRequest) (*User, error) {
	user, err := s.GetUserByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := s.repo.UpdateUser(ctx, s.db.DB, user); err != nil {
		logger.Error("failed to update user", "error", err)
		return nil, err
	}

	return user, nil
}

func (s *service) DeleteUser(ctx context.Context, id int64) error {
	if err := s.repo.DeleteUser(ctx, s.db.DB, id); err != nil {
		logger.Error("failed to delete user", "error", err)
		return err
	}

	return nil
}
