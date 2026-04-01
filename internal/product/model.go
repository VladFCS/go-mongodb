package product

import "go.mongodb.org/mongo-driver/bson/primitive"

const (
	DefaultListLimit     = 10
	MaxListLimit         = 100
	MaxDescriptionLength = 500
)

type Product struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Name        string             `bson:"name" json:"name"`
	Price       float64            `bson:"price" json:"price"`
	InStock     bool               `bson:"in_stock" json:"in_stock"`
	Description string             `bson:"description,omitempty" json:"description,omitempty"`
}

type CreateProductRequest struct {
	Name        string  `json:"name"`
	Price       float64 `json:"price"`
	InStock     bool    `json:"in_stock"`
	Description string  `json:"description,omitempty"`
}

type UpdateProductRequest struct {
	Name        string  `json:"name"`
	Price       float64 `json:"price"`
	InStock     bool    `json:"in_stock"`
	Description string  `json:"description,omitempty"`
}

type UpdateManyProductsRequest struct {
	IDs     []string             `json:"ids"`
	Product UpdateProductRequest `json:"product"`
}

type PatchProductRequest struct {
	Name        *string  `json:"name,omitempty"`
	Price       *float64 `json:"price,omitempty"`
	InStock     *bool    `json:"in_stock,omitempty"`
	Description *string  `json:"description,omitempty"`
}

type PatchManyProductsRequest struct {
	IDs     []string            `json:"ids"`
	Product PatchProductRequest `json:"product"`
}

type ListProductsParams struct {
	Limit   int
	Skip    int
	InStock *bool
}
