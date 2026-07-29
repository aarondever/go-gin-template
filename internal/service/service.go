package service

import (
	"context"
	"fmt"

	"github.com/aarondever/go-gin-template/internal/model"
	"github.com/aarondever/go-gin-template/internal/repository"
	p "github.com/aarondever/go-gin-template/pkg/pagination"
	"github.com/aarondever/go-gin-template/pkg/util"
	"github.com/aarondever/go-gin-template/pkg/validation"
)

type Service interface {
	Create(ctx context.Context, user *model.User) (*model.User, error)
	GetByID(ctx context.Context, userID uint64) (*model.User, error)
	GetList(ctx context.Context, page *p.Pagination, filter *model.UserListFilter) ([]*model.User, error)
	Update(ctx context.Context, user *model.User) (*model.User, error)
	Delete(ctx context.Context, userID uint64) error
}

type service struct {
	repo repository.Repository
}

func New(repo repository.Repository) Service {
	return &service{repo: repo}
}

func (s *service) Create(ctx context.Context, user *model.User) (*model.User, error) {
	if err := validation.ValidateStruct(util.TrimStructStr(user)); err != nil {
		return nil, err
	}
	if err := s.repo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("service.Create: %w", err)
	}
	return user, nil
}

func (s *service) GetByID(ctx context.Context, userID uint64) (*model.User, error) {
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("service.GetByID: %w", err)
	}
	return user, nil
}

func (s *service) GetList(
	ctx context.Context,
	page *p.Pagination,
	filter *model.UserListFilter,
) ([]*model.User, error) {
	if err := validation.ValidateStruct(util.TrimStructStr(filter)); err != nil {
		return nil, err
	}
	users, err := s.repo.GetList(ctx, page, filter)
	if err != nil {
		return nil, fmt.Errorf("service.GetList: %w", err)
	}
	return users, nil
}

func (s *service) Update(ctx context.Context, user *model.User) (*model.User, error) {
	if err := validation.ValidateStruct(util.TrimStructStr(user), "Name"); err != nil {
		return nil, err
	}
	if err := s.repo.Update(ctx, user); err != nil {
		return nil, fmt.Errorf("service.Update: %w", err)
	}
	return user, nil
}

func (s *service) Delete(ctx context.Context, userID uint64) error {
	if err := s.repo.Delete(ctx, userID); err != nil {
		return fmt.Errorf("service.Delete: %w", err)
	}
	return nil
}
