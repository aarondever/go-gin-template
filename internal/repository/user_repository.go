package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/aarondever/go-gin-template/internal/dto"
	"github.com/aarondever/go-gin-template/internal/model"
	"github.com/aarondever/go-gin-template/internal/shared/database"
	"gorm.io/gorm"
)

type UserRepository interface {
	CreateUser(ctx context.Context, tx *gorm.DB, user *model.User) error
	GetUserByID(ctx context.Context, userID int64) (*model.User, error)
	GetUserList(ctx context.Context, filter dto.UserListFilter) ([]*model.User, int64, error)
	UpdateUser(ctx context.Context, tx *gorm.DB, user *model.User) error
	DeleteUser(ctx context.Context, tx *gorm.DB, userID int64) error
}

type userRepository struct {
	db *database.Database
}

func NewUserRepository(db *database.Database) UserRepository {
	db.DB.AutoMigrate(&model.User{})
	return &userRepository{db: db}
}

func (r *userRepository) CreateUser(ctx context.Context, tx *gorm.DB, user *model.User) error {
	if err := tx.WithContext(ctx).Create(user).Error; err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

func (r *userRepository) GetUserByID(ctx context.Context, userID int64) (*model.User, error) {
	var user model.User
	err := r.db.WithContext(ctx).Take(&user, userID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, fmt.Errorf("failed to get user by id: %w", err)
	}

	return &user, nil
}

func (r *userRepository) GetUserList(ctx context.Context, filter dto.UserListFilter) ([]*model.User, int64, error) {
	var users []*model.User
	var total int64

	query := r.db.WithContext(ctx).
		Model(&model.User{}).
		Limit(filter.Limit).
		Offset(filter.Offset)

	if filter.Name != "" {
		query = query.Where("name LIKE ?", "%"+filter.Name+"%")
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count users: %w", err)
	}

	if err := query.Find(&users).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to find users: %w", err)
	}

	return users, total, nil
}

func (r *userRepository) UpdateUser(ctx context.Context, tx *gorm.DB, user *model.User) error {
	if err := tx.WithContext(ctx).Updates(user).Error; err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	return nil
}

func (r *userRepository) DeleteUser(ctx context.Context, tx *gorm.DB, userID int64) error {
	if err := tx.WithContext(ctx).Delete(&model.User{}, userID).Error; err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	return nil
}
