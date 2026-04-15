package repository

import (
	"context"
	"errors"

	"github.com/aarondever/go-gin-template/internal/database"
	"github.com/aarondever/go-gin-template/internal/dto"
	"github.com/aarondever/go-gin-template/internal/model"
	"gorm.io/gorm"
)

type UserRepository interface {
	CreateUser(ctx context.Context, user *model.User) error
	GetUserByID(ctx context.Context, userID int64) (*model.User, error)
	GetUserList(ctx context.Context, filter dto.UserListFilter) ([]*model.User, int64, error)
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
	return database.ExtractTx(ctx, r.db).WithContext(ctx).Create(user).Error
}

func (r *userRepository) GetUserByID(ctx context.Context, userID int64) (*model.User, error) {
	var user model.User
	err := database.ExtractTx(ctx, r.db).WithContext(ctx).Take(&user, userID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, err
	}

	return &user, nil
}

func (r *userRepository) GetUserList(ctx context.Context, filter dto.UserListFilter) ([]*model.User, int64, error) {
	var users []*model.User
	var total int64

	query := database.ExtractTx(ctx, r.db).WithContext(ctx).
		Model(&model.User{}).
		Limit(filter.Limit).
		Offset(filter.Offset)

	if filter.Name != "" {
		query = query.Where("name LIKE ?", "%"+filter.Name+"%")
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.Find(&users).Error; err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

func (r *userRepository) UpdateUser(ctx context.Context, user *model.User) error {
	return database.ExtractTx(ctx, r.db).WithContext(ctx).Updates(user).Error
}

func (r *userRepository) DeleteUser(ctx context.Context, userID int64) error {
	return database.ExtractTx(ctx, r.db).WithContext(ctx).Delete(&model.User{}, userID).Error
}
