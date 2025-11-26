package user

import (
	"context"
	"database/sql"
	"errors"

	"github.com/aarondever/go-gin-template/internal/database"
)

type Repository struct {
	db *database.Database
}

func NewRepository(db *database.Database) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, user *User) error {
	query := `
		INSERT INTO users (email, name, created_at, updated_at)
		VALUES ($1, $2, NOW(), NOW())
		RETURNING id, created_at, updated_at`

	return r.db.QueryRowContext(ctx, query, user.Email, user.Name).
		Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)
}

func (r *Repository) GetByID(ctx context.Context, id int64) (*User, error) {
	var user User
	query := `SELECT id, email, name, created_at, updated_at FROM users WHERE id = $1`

	err := r.db.GetContext(ctx, &user, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return &user, nil
}

func (r *Repository) List(ctx context.Context, limit, offset int) ([]*User, error) {
	var users []*User
	query := `SELECT id, email, name, created_at, updated_at FROM users ORDER BY id LIMIT $1 OFFSET $2`

	err := r.db.SelectContext(ctx, &users, query, limit, offset)
	return users, err
}

func (r *Repository) Update(ctx context.Context, user *User) error {
	query := `
		UPDATE users 
		SET email = $1, name = $2, updated_at = NOW()
		WHERE id = $3
		RETURNING updated_at`

	return r.db.QueryRowContext(ctx, query, user.Email, user.Name, user.ID).
		Scan(&user.UpdatedAt)
}

func (r *Repository) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM users WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}
