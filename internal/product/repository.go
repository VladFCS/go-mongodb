package product

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Repository interface {
	Create(ctx context.Context, product *Product) error
	List(ctx context.Context) ([]Product, error)
}

type MongoRepository struct {
	collection *mongo.Collection
}

func NewMongoRepository(collection *mongo.Collection) *MongoRepository {
	return &MongoRepository{collection: collection}
}

func (r *MongoRepository) Create(ctx context.Context, product *Product) error {
	result, err := r.collection.InsertOne(ctx, product)
	if err != nil {
		return err
	}

	if insertedObjectID, ok := result.InsertedID.(primitive.ObjectID); ok {
		product.ID = insertedObjectID
	}

	return nil
}

func (r *MongoRepository) List(ctx context.Context) ([]Product, error) {
	cursor, err := r.collection.Find(ctx, bson.D{}, options.Find().SetSort(bson.D{{Key: "name", Value: 1}}))
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = cursor.Close(ctx)
	}()

	var products []Product
	if err := cursor.All(ctx, &products); err != nil {
		return nil, err
	}

	return products, nil
}
