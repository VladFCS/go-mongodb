package product

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

var ErrInvalidProduct = errors.New("invalid product")

type Service struct {
	repository Repository
}

func NewService(repository Repository) *Service {
	return &Service{repository: repository}
}

func (s *Service) CreateProduct(ctx context.Context, request CreateProductRequest) (*Product, error) {
	name := strings.TrimSpace(request.Name)
	if name == "" {
		return nil, fmt.Errorf("%w: name is required", ErrInvalidProduct)
	}

	if request.Price < 0 {
		return nil, fmt.Errorf("%w: price must be greater than or equal to zero", ErrInvalidProduct)
	}

	product := &Product{
		Name:        name,
		Price:       request.Price,
		InStock:     request.InStock,
		Description: strings.TrimSpace(request.Description),
	}

	if err := s.repository.Create(ctx, product); err != nil {
		return nil, err
	}

	return product, nil
}

func (s *Service) ListProducts(ctx context.Context) ([]Product, error) {
	return s.repository.List(ctx)
}
