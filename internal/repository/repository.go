package repository

import (
	"context"
	"errors"
	"fmt"

	e "github.com/aarondever/go-gin-template/internal/apperror"
	"github.com/aarondever/go-gin-template/internal/model"
	"github.com/aarondever/go-gin-template/pkg/database"
	p "github.com/aarondever/go-gin-template/pkg/pagination"
	"gorm.io/gorm"
)

type Repository interface {
	Create(ctx context.Context, user *model.User) error
	GetByID(ctx context.Context, userID uint64) (*model.User, error)
	GetList(ctx context.Context, page *p.Pagination, filter *model.UserListFilter) ([]*model.User, error)
	Update(ctx context.Context, user *model.User) error
	Delete(ctx context.Context, userID uint64) error
}

type repository struct {
	db *gorm.DB
}

func New(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) Create(ctx context.Context, user *model.User) error {
	if err := database.ExtractTx(ctx, r.db).WithContext(ctx).Create(user).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return e.Wrap(err, e.CodeConflict, "email already in use")
		}
		return fmt.Errorf("create user: %w", err)
	}
	return nil
}

func (r *repository) GetByID(ctx context.Context, userID uint64) (*model.User, error) {
	var user model.User
	if err := database.ExtractTx(ctx, r.db).WithContext(ctx).Take(&user, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, e.Wrap(err, e.CodeNotFound, "user not found")
		}
		return nil, fmt.Errorf("get user %d: %w", userID, err)
	}
	return &user, nil
}

func (r *repository) GetList(ctx context.Context, page *p.Pagination, filter *model.UserListFilter) ([]*model.User, error) {
	q := database.ExtractTx(ctx, r.db).WithContext(ctx)
	if filter.Name != "" {
		q = q.Where("name LIKE ?", "%"+filter.Name+"%")
	}
	if filter.Email != "" {
		q = q.Where("email = ?", filter.Email)
	}

	var users []*model.User
	if err := q.Scopes(database.Paginate(page)).Find(&users).Error; err != nil {
		return nil, fmt.Errorf("get user list: %w", err)
	}

	return users, nil
}

func (r *repository) Update(ctx context.Context, user *model.User) error {
	result := database.ExtractTx(ctx, r.db).WithContext(ctx).Updates(user)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrDuplicatedKey) {
			return e.Wrap(result.Error, e.CodeConflict, "email already in use")
		}
		return fmt.Errorf("update user %d: %w", user.ID, result.Error)
	}
	return nil
}

func (r *repository) Delete(ctx context.Context, userID uint64) error {
	result := database.ExtractTx(ctx, r.db).WithContext(ctx).Delete(&model.User{}, userID)
	if result.Error != nil {
		return fmt.Errorf("delete user %d: %w", userID, result.Error)
	}
	return nil
}
