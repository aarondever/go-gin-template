package product

import (
	"context"
	"database/sql"
	"errors"

	"github.com/aarondever/go-gin-template/internal/database"
)

type Repository interface {
	Create(ctx context.Context, product *Product) error
	GetByID(ctx context.Context, id int64) (*Product, error)
	List(ctx context.Context, limit, offset int) ([]*Product, error)
	Update(ctx context.Context, product *Product) error
	Delete(ctx context.Context, id int64) error
}

type repository struct {
	db *database.Database
}

func NewRepository(db *database.Database) Repository {
	return &repository{db: db}
}

func (r *repository) Create(ctx context.Context, product *Product) error {
	query := `
		INSERT INTO products (name, description, price, stock, created_at, updated_at)
		VALUES ($1, $2, $3, $4, NOW(), NOW())
		RETURNING id, created_at, updated_at`

	return r.db.QueryRowContext(ctx, query, product.Name, product.Description, product.Price, product.Stock).
		Scan(&product.ID, &product.CreatedAt, &product.UpdatedAt)
}

func (r *repository) GetByID(ctx context.Context, id int64) (*Product, error) {
	var product Product
	query := `SELECT id, name, description, price, stock, created_at, updated_at FROM products WHERE id = $1`

	err := r.db.GetContext(ctx, &product, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return &product, nil
}

func (r *repository) List(ctx context.Context, limit, offset int) ([]*Product, error) {
	var products []*Product
	query := `SELECT id, name, description, price, stock, created_at, updated_at FROM products ORDER BY id LIMIT $1 OFFSET $2`

	err := r.db.SelectContext(ctx, &products, query, limit, offset)
	return products, err
}

func (r *repository) Update(ctx context.Context, product *Product) error {
	query := `
		UPDATE products 
		SET name = $1, description = $2, price = $3, stock = $4, updated_at = NOW()
		WHERE id = $5
		RETURNING updated_at`

	return r.db.QueryRowContext(ctx, query, product.Name, product.Description, product.Price, product.Stock, product.ID).
		Scan(&product.UpdatedAt)
}

func (r *repository) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM products WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}
