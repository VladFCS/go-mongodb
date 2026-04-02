package product

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

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

// NewService constructs the product use-case layer.
func NewService(repository Repository) *Service {
	return &Service{repository: repository}
}

// CreateProduct validates and creates one product.
func (s *Service) CreateProduct(ctx context.Context, request CreateProductRequest) (*Product, error) {
	products, err := s.CreateProducts(ctx, []CreateProductRequest{request})
	if err != nil {
		return nil, err
	}

	return &products[0], nil
}

// CreateProductWithTransaction demonstrates MongoDB transactions by inserting a product and audit record atomically.
func (s *Service) CreateProductWithTransaction(ctx context.Context, request CreateProductRequest) (*CreateProductTransactionResult, error) {
	product := &Product{
		Name:        request.Name,
		Price:       request.Price,
		InStock:     request.InStock,
		Description: request.Description,
	}

	if err := validateProduct(product); err != nil {
		return nil, err
	}

	audit := &ProductAudit{
		Action:    "product.created",
		Message:   fmt.Sprintf("product %q created in transaction", product.Name),
		CreatedAt: time.Now().UTC(),
	}

	if err := s.repository.CreateWithAudit(ctx, product, audit); err != nil {
		return nil, err
	}

	return &CreateProductTransactionResult{
		Product: *product,
		Audit:   *audit,
	}, nil
}

// CreateProducts validates and creates multiple products in one request.
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

// ListProducts validates list params and returns matching products.
func (s *Service) ListProducts(ctx context.Context, params ListProductsParams) ([]Product, error) {
	validatedParams, err := validateListProductsParams(params)
	if err != nil {
		return nil, err
	}

	return s.repository.List(ctx, validatedParams)
}

// GetProductByID validates the id and loads one product.
func (s *Service) GetProductByID(ctx context.Context, id string) (*Product, error) {
	objectID, err := parseProductID(id)
	if err != nil {
		return nil, err
	}

	return s.repository.GetByID(ctx, objectID)
}

// ReplaceProduct uses repository.ReplaceOne, which maps to MongoDB ReplaceOne.
func (s *Service) ReplaceProduct(ctx context.Context, id string, request ReplaceProductRequest) (*Product, error) {
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

	if err := s.repository.ReplaceOne(ctx, objectID, product); err != nil {
		return nil, err
	}

	return s.repository.GetByID(ctx, objectID)
}

// ReplaceProducts uses repository.UpdateMany because MongoDB has no ReplaceMany.
func (s *Service) ReplaceProducts(ctx context.Context, request ReplaceProductsRequest) error {
	objectIDs, err := parseProductIDs(request.IDs)
	if err != nil {
		return err
	}

	product := &Product{
		Name:        request.Product.Name,
		Price:       request.Product.Price,
		InStock:     request.Product.InStock,
		Description: request.Product.Description,
	}

	if err := validateProduct(product); err != nil {
		return err
	}

	return s.repository.UpdateMany(ctx, objectIDs, buildReplaceProductsUpdate(product))
}

// UpdateProduct uses repository.UpdateOne, which maps to MongoDB UpdateOne.
func (s *Service) UpdateProduct(ctx context.Context, id string, request UpdateProductRequest) (*Product, error) {
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

	if err := s.repository.UpdateOne(ctx, objectID, update); err != nil {
		return nil, err
	}

	return s.repository.GetByID(ctx, objectID)
}

// UpdateProducts uses repository.UpdateMany, which maps to MongoDB UpdateMany.
func (s *Service) UpdateProducts(ctx context.Context, request UpdateProductsRequest) error {
	objectIDs, err := parseProductIDs(request.IDs)
	if err != nil {
		return err
	}

	update, err := buildUpdateProductDocument(request.Product)
	if err != nil {
		return err
	}

	return s.repository.UpdateMany(ctx, objectIDs, update)
}

// DeleteProduct validates the id and removes one product.
func (s *Service) DeleteProduct(ctx context.Context, id string) error {
	objectID, err := parseProductID(id)
	if err != nil {
		return err
	}

	return s.repository.Delete(ctx, objectID)
}

// parseProductID converts a hex string into a MongoDB ObjectID.
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

// parseProductIDs converts multiple hex strings into MongoDB ObjectIDs.
func parseProductIDs(ids []string) ([]primitive.ObjectID, error) {
	if len(ids) == 0 {
		return nil, fmt.Errorf("%w: at least one id is required", ErrInvalidID)
	}

	objectIDs := make([]primitive.ObjectID, 0, len(ids))
	for _, id := range ids {
		objectID, err := parseProductID(id)
		if err != nil {
			return nil, err
		}

		objectIDs = append(objectIDs, objectID)
	}

	return objectIDs, nil
}

// buildReplaceProductsUpdate creates one shared $set document for bulk replace-like updates.
func buildReplaceProductsUpdate(product *Product) bson.M {
	return bson.M{
		"$set": bson.M{
			"name":        product.Name,
			"price":       product.Price,
			"in_stock":    product.InStock,
			"description": product.Description,
		},
	}
}

// buildUpdateProductDocument creates a MongoDB update document for partial updates.
func buildUpdateProductDocument(request UpdateProductRequest) (bson.M, error) {
	if request.Name == nil && request.Price == nil && request.InStock == nil && request.Description == nil {
		return nil, fmt.Errorf("%w: at least one field is required", ErrInvalidProduct)
	}

	updateFields := bson.M{}

	if request.Name != nil {
		name := strings.TrimSpace(*request.Name)
		if name == "" {
			return nil, fmt.Errorf("%w: name is required", ErrInvalidProduct)
		}

		updateFields["name"] = name
	}

	if request.Price != nil {
		if *request.Price <= 0 {
			return nil, fmt.Errorf("%w: price must be greater than zero", ErrInvalidProduct)
		}

		updateFields["price"] = *request.Price
	}

	if request.InStock != nil {
		updateFields["in_stock"] = *request.InStock
	}

	if request.Description != nil {
		description := strings.TrimSpace(*request.Description)
		if len(description) > MaxDescriptionLength {
			return nil, fmt.Errorf("%w: description must be %d characters or fewer", ErrInvalidProduct, MaxDescriptionLength)
		}

		updateFields["description"] = description
	}

	return bson.M{"$set": updateFields}, nil
}

// validateProduct applies the shared business validation rules for a product payload.
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

// validateListProductsParams validates list pagination and filter params.
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
