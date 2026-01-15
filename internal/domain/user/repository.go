package user

import (
	"context"
	"errors"
	"fmt"

	"github.com/aarondever/go-gin-template/internal/shared/database"
	"gorm.io/gorm"
)

type Repository interface {
	CreateUser(ctx context.Context, tx *gorm.DB, user *User) error
	GetUserByID(ctx context.Context, userID int64) (*User, error)
	GetUserList(ctx context.Context, filter UserListFilter) ([]*User, int64, error)
	UpdateUser(ctx context.Context, tx *gorm.DB, user *User) error
	DeleteUser(ctx context.Context, tx *gorm.DB, userID int64) error
}

type repository struct {
	db *database.Database
}

func NewRepository(db *database.Database) Repository {
	db.DB.AutoMigrate(&User{})
	return &repository{db: db}
}

func (r *repository) CreateUser(ctx context.Context, tx *gorm.DB, user *User) error {
	if err := tx.WithContext(ctx).Create(user).Error; err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

func (r *repository) GetUserByID(ctx context.Context, userID int64) (*User, error) {
	var user User
	err := r.db.WithContext(ctx).Take(&user, userID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, fmt.Errorf("failed to get user by id: %w", err)
	}

	return &user, nil
}

func (r *repository) GetUserList(ctx context.Context, filter UserListFilter) ([]*User, int64, error) {
	var users []*User
	var total int64

	query := r.db.WithContext(ctx).
		Model(&User{}).
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

func (r *repository) UpdateUser(ctx context.Context, tx *gorm.DB, user *User) error {
	if err := tx.WithContext(ctx).Updates(user).Error; err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	return nil
}

func (r *repository) DeleteUser(ctx context.Context, tx *gorm.DB, userID int64) error {
	if err := tx.WithContext(ctx).Delete(&User{}, userID).Error; err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	return nil
}
