package user

import (
	"context"
	"errors"

	"github.com/aarondever/go-gin-template/internal/shared/database"
	"gorm.io/gorm"
)

type Repository interface {
	CreateUser(ctx context.Context, tx *gorm.DB, user *User) error
	GetUserByID(ctx context.Context, userID int64) (*User, error)
	ListUsers(ctx context.Context, filter UserListFilter) ([]*User, int64, error)
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
	return tx.WithContext(ctx).Create(user).Error
}

func (r *repository) GetUserByID(ctx context.Context, userID int64) (*User, error) {
	var user User
	err := r.db.WithContext(ctx).Take(&user, userID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, err
	}

	return &user, nil
}

func (r *repository) ListUsers(ctx context.Context, filter UserListFilter) ([]*User, int64, error) {
	var users []*User
	var total int64

	query := r.db.WithContext(ctx).
		Model(&User{}).
		Count(&total).
		Limit(filter.Limit).
		Offset(filter.Offset)

	if filter.Name != "" {
		query = query.Where("name LIKE ?", "%"+filter.Name+"%")
	}

	err := query.Find(&users).Error
	if err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

func (r *repository) UpdateUser(ctx context.Context, tx *gorm.DB, user *User) error {
	return tx.WithContext(ctx).Updates(user).Error
}

func (r *repository) DeleteUser(ctx context.Context, tx *gorm.DB, userID int64) error {
	return tx.WithContext(ctx).Delete(&User{}, userID).Error
}
