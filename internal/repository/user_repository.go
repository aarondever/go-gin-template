package repository

import (
	"context"
	"errors"

	e "github.com/aarondever/go-gin-template/errors"
	"github.com/aarondever/go-gin-template/internal/database"
	"github.com/aarondever/go-gin-template/internal/model"
	"github.com/aarondever/go-gin-template/pkg/logger"
	"github.com/aarondever/go-gin-template/pkg/pagination"
	"gorm.io/gorm"
)

type UserListQuery struct {
	Name  string
	Email string
	*pagination.Pagination
}

type UserRepository interface {
	CreateUser(ctx context.Context, user *model.User) error
	GetUserByID(ctx context.Context, userID int64) (*model.User, error)
	GetUserList(ctx context.Context, query UserListQuery) ([]*model.User, error)
	UpdateUser(ctx context.Context, user *model.User) error
	DeleteUser(ctx context.Context, userID int64) error
}

type userRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) CreateUser(ctx context.Context, user *model.User) error {
	if err := database.ExtractTx(ctx, r.db).WithContext(ctx).Create(user).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			logger.Error("user already exists", "error", err)
			return e.ErrConflict
		}
		return err
	}
	return nil
}

func (r *userRepository) GetUserByID(ctx context.Context, userID int64) (*model.User, error) {
	var user model.User
	if err := database.ExtractTx(ctx, r.db).WithContext(ctx).Take(&user, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			logger.Error("user not found", "error", err)
			return nil, e.ErrNotFound
		}
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) GetUserList(ctx context.Context, query UserListQuery) ([]*model.User, error) {
	var users []*model.User
	q := database.ExtractTx(ctx, r.db).WithContext(ctx).Scopes(paginate(query.Pagination))
	if query.Name != "" {
		q = q.Where("name LIKE ?", "%"+query.Name+"%")
	}
	if query.Email != "" {
		q = q.Where("email = ?", query.Email)
	}
	if err := q.Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

func (r *userRepository) UpdateUser(ctx context.Context, user *model.User) error {
	if err := database.ExtractTx(ctx, r.db).WithContext(ctx).Updates(user).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			logger.Error("user already exists", "error", err)
			return e.ErrConflict
		}
		return err
	}
	return nil
}

func (r *userRepository) DeleteUser(ctx context.Context, userID int64) error {
	if err := database.ExtractTx(ctx, r.db).WithContext(ctx).Delete(&model.User{}, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			logger.Error("user not found", "error", err)
			return e.ErrNotFound
		}
		return err
	}
	return nil
}
