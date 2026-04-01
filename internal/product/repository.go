package product

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	ErrNotFound      = errors.New("product not found")
	ErrDuplicateName = errors.New("product name already exists")
)

type Repository interface {
	Create(ctx context.Context, product *Product) error
	CreateMany(ctx context.Context, products []*Product) error
	List(ctx context.Context, params ListProductsParams) ([]Product, error)
	GetByID(ctx context.Context, id primitive.ObjectID) (*Product, error)
	Update(ctx context.Context, id primitive.ObjectID, update bson.M) error
	Delete(ctx context.Context, id primitive.ObjectID) error
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
		return mapMongoError(err)
	}

	if insertedObjectID, ok := result.InsertedID.(primitive.ObjectID); ok {
		product.ID = insertedObjectID
	}

	return nil
}

func (r *MongoRepository) CreateMany(ctx context.Context, products []*Product) error {
	if len(products) == 0 {
		return nil
	}

	documents := make([]any, len(products))
	for i, product := range products {
		documents[i] = product
	}

	result, err := r.collection.InsertMany(ctx, documents)
	if err != nil {
		return mapMongoError(err)
	}

	for i, insertedID := range result.InsertedIDs {
		insertedObjectID, ok := insertedID.(primitive.ObjectID)
		if !ok {
			continue
		}

		products[i].ID = insertedObjectID
	}

	return nil
}

func (r *MongoRepository) List(ctx context.Context, params ListProductsParams) ([]Product, error) {
	filter := bson.M{}
	if params.InStock != nil {
		filter["in_stock"] = *params.InStock
	}

	findOptions := options.Find().
		SetSort(bson.D{{Key: "name", Value: 1}}).
		SetLimit(int64(params.Limit)).
		SetSkip(int64(params.Skip))

	cursor, err := r.collection.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, mapMongoError(err)
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

func (r *MongoRepository) GetByID(ctx context.Context, id primitive.ObjectID) (*Product, error) {
	filter := bson.M{"_id": id}

	var product Product
	if err := r.collection.FindOne(ctx, filter).Decode(&product); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrNotFound
		}

		return nil, mapMongoError(err)
	}

	return &product, nil
}

func (r *MongoRepository) Update(ctx context.Context, id primitive.ObjectID, update bson.M) error {
	filter := bson.M{"_id": id}

	res, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return mapMongoError(err)
	}

	if res.MatchedCount == 0 {
		return ErrNotFound
	}

	return nil
}

func (r *MongoRepository) Delete(ctx context.Context, id primitive.ObjectID) error {
	filter := bson.M{"_id": id}

	res, err := r.collection.DeleteOne(ctx, filter)
	if err != nil {
		return mapMongoError(err)
	}

	if res.DeletedCount == 0 {
		return ErrNotFound
	}

	return nil
}

func mapMongoError(err error) error {
	var writeException mongo.WriteException
	if errors.As(err, &writeException) {
		for _, writeError := range writeException.WriteErrors {
			if writeError.Code == 11000 {
				return ErrDuplicateName
			}
		}
	}

	var bulkWriteException mongo.BulkWriteException
	if errors.As(err, &bulkWriteException) {
		for _, writeError := range bulkWriteException.WriteErrors {
			if writeError.Code == 11000 {
				return ErrDuplicateName
			}
		}
	}

	var commandError mongo.CommandError
	if errors.As(err, &commandError) && commandError.Code == 11000 {
		return ErrDuplicateName
	}

	return err
}
