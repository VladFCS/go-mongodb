package product

import "go.mongodb.org/mongo-driver/bson/primitive"

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
