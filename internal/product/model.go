package product

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

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

type ProductAudit struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	ProductID primitive.ObjectID `bson:"product_id" json:"product_id"`
	Action    string             `bson:"action" json:"action"`
	Message   string             `bson:"message" json:"message"`
	CreatedAt time.Time          `bson:"created_at" json:"created_at"`
}

type CreateProductTransactionResult struct {
	Product Product      `json:"product"`
	Audit   ProductAudit `json:"audit"`
}

type CreateProductRequest struct {
	Name        string  `json:"name"`
	Price       float64 `json:"price"`
	InStock     bool    `json:"in_stock"`
	Description string  `json:"description,omitempty"`
}

type ReplaceProductRequest struct {
	Name        string  `json:"name"`
	Price       float64 `json:"price"`
	InStock     bool    `json:"in_stock"`
	Description string  `json:"description,omitempty"`
}

type ReplaceProductsRequest struct {
	IDs     []string              `json:"ids"`
	Product ReplaceProductRequest `json:"product"`
}

type UpdateProductRequest struct {
	Name        *string  `json:"name,omitempty"`
	Price       *float64 `json:"price,omitempty"`
	InStock     *bool    `json:"in_stock,omitempty"`
	Description *string  `json:"description,omitempty"`
}

type UpdateProductsRequest struct {
	IDs     []string             `json:"ids"`
	Product UpdateProductRequest `json:"product"`
}

type ListProductsParams struct {
	Limit   int
	Skip    int
	InStock *bool
}
