package repository

import (
	"context"
	"time"

	"github.com/aarondever/go-gin-template/internal/database"
)

type User struct {
	ID        int64     `db:"id" json:"id"`
	Username  string    `db:"username" json:"username"`
	Email     string    `db:"email" json:"email"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

type UserRepository struct {
	db *database.Database
}

func NewUserRepository(db *database.Database) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(ctx context.Context, user *User) error {
	query := `INSERT INTO users (username, email, created_at, updated_at) 
              VALUES (?, ?, NOW(), NOW())`

	result, err := r.db.ExecContext(ctx, query, user.Username, user.Email)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}

	user.ID = id
	return nil
}

func (r *UserRepository) GetByID(ctx context.Context, id int64) (*User, error) {
	var user User
	query := `SELECT id, username, email, created_at, updated_at 
              FROM users WHERE id = ?`

	err := r.db.GetContext(ctx, &user, query, id)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *UserRepository) List(ctx context.Context, limit, offset int) ([]User, error) {
	var users []User
	query := `SELECT id, username, email, created_at, updated_at 
              FROM users ORDER BY id DESC LIMIT ? OFFSET ?`

	err := r.db.SelectContext(ctx, &users, query, limit, offset)
	if err != nil {
		return nil, err
	}

	return users, nil
}

func (r *UserRepository) Update(ctx context.Context, user *User) error {
	query := `UPDATE users SET username = ?, email = ?, updated_at = NOW() 
              WHERE id = ?`

	_, err := r.db.ExecContext(ctx, query, user.Username, user.Email, user.ID)
	return err
}

func (r *UserRepository) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM users WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}
