package product

import (
	"context"
	"errors"
)

var (
	ErrProductNotFound = errors.New("product not found")
	ErrInvalidInput    = errors.New("invalid input")
)

type Service interface {
	Create(ctx context.Context, req *CreateProductRequest) (*Product, error)
	GetByID(ctx context.Context, id int64) (*Product, error)
	List(ctx context.Context, limit, offset int) ([]*Product, error)
	Update(ctx context.Context, id int64, req *UpdateProductRequest) (*Product, error)
	Delete(ctx context.Context, id int64) error
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) Create(ctx context.Context, req *CreateProductRequest) (*Product, error) {
	product := &Product{
		Name:        req.Name,
		Description: req.Description,
		Price:       req.Price,
		Stock:       req.Stock,
	}

	if err := s.repo.Create(ctx, product); err != nil {
		return nil, err
	}

	return product, nil
}

func (s *service) GetByID(ctx context.Context, id int64) (*Product, error) {
	product, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if product == nil {
		return nil, ErrProductNotFound
	}
	return product, nil
}

func (s *service) List(ctx context.Context, limit, offset int) ([]*Product, error) {
	if limit <= 0 {
		limit = 10
	}
	if offset < 0 {
		offset = 0
	}
	return s.repo.List(ctx, limit, offset)
}

func (s *service) Update(ctx context.Context, id int64, req *UpdateProductRequest) (*Product, error) {
	product, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if product == nil {
		return nil, ErrProductNotFound
	}

	if req.Name != "" {
		product.Name = req.Name
	}
	if req.Description != "" {
		product.Description = req.Description
	}
	if req.Price > 0 {
		product.Price = req.Price
	}
	if req.Stock >= 0 {
		product.Stock = req.Stock
	}

	if err := s.repo.Update(ctx, product); err != nil {
		return nil, err
	}

	return product, nil
}

func (s *service) Delete(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}
