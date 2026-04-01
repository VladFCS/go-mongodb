package product

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var (
	ErrInvalidProduct = errors.New("invalid product")
	ErrInvalidID      = errors.New("invalid product id")
	ErrInvalidQuery   = errors.New("invalid query")
)

type Service struct {
	repository Repository
}

func NewService(repository Repository) *Service {
	return &Service{repository: repository}
}

func (s *Service) CreateProduct(ctx context.Context, request CreateProductRequest) (*Product, error) {
	products, err := s.CreateProducts(ctx, []CreateProductRequest{request})
	if err != nil {
		return nil, err
	}

	return &products[0], nil
}

func (s *Service) CreateProducts(ctx context.Context, requests []CreateProductRequest) ([]Product, error) {
	if len(requests) == 0 {
		return nil, fmt.Errorf("%w: at least one product is required", ErrInvalidProduct)
	}

	products := make([]*Product, 0, len(requests))
	for _, request := range requests {
		product := &Product{
			Name:        request.Name,
			Price:       request.Price,
			InStock:     request.InStock,
			Description: request.Description,
		}

		if err := validateProduct(product); err != nil {
			return nil, err
		}

		products = append(products, product)
	}

	if len(products) == 1 {
		if err := s.repository.Create(ctx, products[0]); err != nil {
			return nil, err
		}
	} else {
		if err := s.repository.CreateMany(ctx, products); err != nil {
			return nil, err
		}
	}

	createdProducts := make([]Product, len(products))
	for i, product := range products {
		createdProducts[i] = *product
	}

	return createdProducts, nil
}

func (s *Service) ListProducts(ctx context.Context, params ListProductsParams) ([]Product, error) {
	validatedParams, err := validateListProductsParams(params)
	if err != nil {
		return nil, err
	}

	return s.repository.List(ctx, validatedParams)
}

func (s *Service) GetProductByID(ctx context.Context, id string) (*Product, error) {
	objectID, err := parseProductID(id)
	if err != nil {
		return nil, err
	}

	return s.repository.GetByID(ctx, objectID)
}

func (s *Service) ReplaceProduct(ctx context.Context, id string, request UpdateProductRequest) (*Product, error) {
	objectID, err := parseProductID(id)
	if err != nil {
		return nil, err
	}

	product := &Product{
		ID:          objectID,
		Name:        request.Name,
		Price:       request.Price,
		InStock:     request.InStock,
		Description: request.Description,
	}

	if err := validateProduct(product); err != nil {
		return nil, err
	}

	update := bson.M{
		"$set": bson.M{
			"name":        product.Name,
			"price":       product.Price,
			"in_stock":    product.InStock,
			"description": product.Description,
		},
	}

	if err := s.repository.Update(ctx, objectID, update); err != nil {
		return nil, err
	}

	return s.repository.GetByID(ctx, objectID)
}

func (s *Service) PatchProduct(ctx context.Context, id string, request PatchProductRequest) (*Product, error) {
	objectID, err := parseProductID(id)
	if err != nil {
		return nil, err
	}

	product, err := s.repository.GetByID(ctx, objectID)
	if err != nil {
		return nil, err
	}

	if request.Name == nil && request.Price == nil && request.InStock == nil && request.Description == nil {
		return nil, fmt.Errorf("%w: at least one field is required", ErrInvalidProduct)
	}

	if request.Name != nil {
		product.Name = *request.Name
	}

	if request.Price != nil {
		product.Price = *request.Price
	}

	if request.InStock != nil {
		product.InStock = *request.InStock
	}

	if request.Description != nil {
		product.Description = *request.Description
	}

	if err := validateProduct(product); err != nil {
		return nil, err
	}

	update := bson.M{
		"$set": bson.M{
			"name":        product.Name,
			"price":       product.Price,
			"in_stock":    product.InStock,
			"description": product.Description,
		},
	}

	if err := s.repository.Update(ctx, objectID, update); err != nil {
		return nil, err
	}

	return s.repository.GetByID(ctx, objectID)
}

func (s *Service) DeleteProduct(ctx context.Context, id string) error {
	objectID, err := parseProductID(id)
	if err != nil {
		return err
	}

	return s.repository.Delete(ctx, objectID)
}

func parseProductID(id string) (primitive.ObjectID, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return primitive.NilObjectID, fmt.Errorf("%w: id is required", ErrInvalidID)
	}

	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return primitive.NilObjectID, fmt.Errorf("%w: invalid id format", ErrInvalidID)
	}

	return objectID, nil
}

func validateProduct(product *Product) error {
	product.Name = strings.TrimSpace(product.Name)
	product.Description = strings.TrimSpace(product.Description)

	if product.Name == "" {
		return fmt.Errorf("%w: name is required", ErrInvalidProduct)
	}

	if product.Price <= 0 {
		return fmt.Errorf("%w: price must be greater than zero", ErrInvalidProduct)
	}

	if len(product.Description) > MaxDescriptionLength {
		return fmt.Errorf("%w: description must be %d characters or fewer", ErrInvalidProduct, MaxDescriptionLength)
	}

	return nil
}

func validateListProductsParams(params ListProductsParams) (ListProductsParams, error) {
	if params.Limit == 0 {
		params.Limit = DefaultListLimit
	}

	if params.Limit < 0 {
		return ListProductsParams{}, fmt.Errorf("%w: limit must be greater than or equal to zero", ErrInvalidQuery)
	}

	if params.Skip < 0 {
		return ListProductsParams{}, fmt.Errorf("%w: skip must be greater than or equal to zero", ErrInvalidQuery)
	}

	if params.Limit > MaxListLimit {
		return ListProductsParams{}, fmt.Errorf("%w: limit must be less than or equal to %d", ErrInvalidQuery, MaxListLimit)
	}

	return params, nil
}
