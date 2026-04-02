package product

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readconcern"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"
)

var (
	ErrNotFound      = errors.New("product not found")
	ErrDuplicateName = errors.New("product name already exists")
)

type Repository interface {
	Create(ctx context.Context, product *Product) error
	CreateMany(ctx context.Context, products []*Product) error
	CreateWithAudit(ctx context.Context, product *Product, audit *ProductAudit) error
	List(ctx context.Context, params ListProductsParams) ([]Product, error)
	GetByID(ctx context.Context, id primitive.ObjectID) (*Product, error)
	UpdateOne(ctx context.Context, id primitive.ObjectID, update bson.M) error
	UpdateMany(ctx context.Context, ids []primitive.ObjectID, update bson.M) error
	ReplaceOne(ctx context.Context, id primitive.ObjectID, product *Product) error
	Delete(ctx context.Context, id primitive.ObjectID) error
}

type MongoRepository struct {
	client          *mongo.Client
	collection      *mongo.Collection
	auditCollection *mongo.Collection
}

// NewMongoRepository constructs a product repository backed by a MongoDB collection.
func NewMongoRepository(client *mongo.Client, collection *mongo.Collection, auditCollection *mongo.Collection) *MongoRepository {
	return &MongoRepository{
		client:          client,
		collection:      collection,
		auditCollection: auditCollection,
	}
}

// Create stores one product using MongoDB InsertOne.
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

// CreateMany stores multiple products using MongoDB InsertMany.
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

// CreateWithAudit stores a product and its audit record atomically in one MongoDB transaction.
func (r *MongoRepository) CreateWithAudit(ctx context.Context, product *Product, audit *ProductAudit) error {
	if product.ID.IsZero() {
		product.ID = primitive.NewObjectID()
	}

	if audit.ID.IsZero() {
		audit.ID = primitive.NewObjectID()
	}

	audit.ProductID = product.ID

	session, err := r.client.StartSession()
	if err != nil {
		return err
	}
	defer session.EndSession(ctx)

	transactionOptions := options.Transaction().
		SetReadConcern(readconcern.Snapshot()).
		SetWriteConcern(writeconcern.Majority())

	_, err = session.WithTransaction(ctx, func(sessionCtx mongo.SessionContext) (any, error) {
		if _, err := r.collection.InsertOne(sessionCtx, product); err != nil {
			return nil, mapMongoError(err)
		}

		if _, err := r.auditCollection.InsertOne(sessionCtx, audit); err != nil {
			return nil, mapMongoError(err)
		}

		return nil, nil
	}, transactionOptions)
	if err != nil {
		return mapMongoError(err)
	}

	return nil
}

// List returns products filtered by the supported query params.
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

// GetByID loads one product by its MongoDB ObjectID.
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

// UpdateOne uses MongoDB UpdateOne with an update document such as $set.
func (r *MongoRepository) UpdateOne(ctx context.Context, id primitive.ObjectID, update bson.M) error {
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

// UpdateMany uses MongoDB UpdateMany with one shared update document for all ids.
func (r *MongoRepository) UpdateMany(ctx context.Context, ids []primitive.ObjectID, update bson.M) error {
	if len(ids) == 0 {
		return nil
	}

	uniqueIDs := make([]primitive.ObjectID, 0, len(ids))
	seenIDs := make(map[primitive.ObjectID]struct{}, len(ids))
	for _, id := range ids {
		if _, exists := seenIDs[id]; exists {
			continue
		}

		seenIDs[id] = struct{}{}
		uniqueIDs = append(uniqueIDs, id)
	}

	filter := bson.M{
		"_id": bson.M{
			"$in": uniqueIDs,
		},
	}

	res, err := r.collection.UpdateMany(ctx, filter, update)
	if err != nil {
		return mapMongoError(err)
	}

	if res.MatchedCount != int64(len(uniqueIDs)) {
		return ErrNotFound
	}

	return nil
}

// ReplaceOne uses MongoDB ReplaceOne with a full replacement document.
func (r *MongoRepository) ReplaceOne(ctx context.Context, id primitive.ObjectID, product *Product) error {
	filter := bson.M{"_id": id}
	product.ID = id

	res, err := r.collection.ReplaceOne(ctx, filter, product)
	if err != nil {
		return mapMongoError(err)
	}

	if res.MatchedCount == 0 {
		return ErrNotFound
	}

	return nil
}

// Delete removes one product using MongoDB DeleteOne.
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

// mapMongoError converts MongoDB driver errors into domain-level repository errors.
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
