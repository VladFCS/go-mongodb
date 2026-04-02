package product

import (
	"context"
	"testing"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type stubRepository struct {
	storedProduct          *Product
	replaceOneCalled       bool
	replaceOneID           primitive.ObjectID
	replaceOneProduct      *Product
	createWithAuditCalled  bool
	createWithAuditProduct *Product
	createWithAuditAudit   *ProductAudit
	listCalled             bool
	listParams             ListProductsParams
	listResult             []Product
	listErr                error
}

func (r *stubRepository) Create(ctx context.Context, product *Product) error {
	return nil
}

func (r *stubRepository) CreateMany(ctx context.Context, products []*Product) error {
	return nil
}

func (r *stubRepository) CreateWithAudit(ctx context.Context, product *Product, audit *ProductAudit) error {
	r.createWithAuditCalled = true

	productCopy := *product
	if productCopy.ID.IsZero() {
		productCopy.ID = primitive.NewObjectID()
	}

	auditCopy := *audit
	if auditCopy.ID.IsZero() {
		auditCopy.ID = primitive.NewObjectID()
	}
	auditCopy.ProductID = productCopy.ID

	*product = productCopy
	*audit = auditCopy

	r.createWithAuditProduct = &productCopy
	r.createWithAuditAudit = &auditCopy
	r.storedProduct = &productCopy

	return nil
}

func (r *stubRepository) List(ctx context.Context, params ListProductsParams) ([]Product, error) {
	r.listCalled = true
	r.listParams = params
	return r.listResult, r.listErr
}

func (r *stubRepository) GetByID(ctx context.Context, id primitive.ObjectID) (*Product, error) {
	if r.storedProduct == nil || r.storedProduct.ID != id {
		return nil, ErrNotFound
	}

	productCopy := *r.storedProduct
	return &productCopy, nil
}

func (r *stubRepository) UpdateOne(ctx context.Context, id primitive.ObjectID, update bson.M) error {
	return nil
}

func (r *stubRepository) UpdateMany(ctx context.Context, ids []primitive.ObjectID, update bson.M) error {
	return nil
}

func (r *stubRepository) ReplaceOne(ctx context.Context, id primitive.ObjectID, product *Product) error {
	r.replaceOneCalled = true
	r.replaceOneID = id

	productCopy := *product
	productCopy.ID = id

	r.replaceOneProduct = &productCopy
	r.storedProduct = &productCopy

	return nil
}

func (r *stubRepository) Delete(ctx context.Context, id primitive.ObjectID) error {
	return nil
}

func TestServiceCreateProductWithTransaction_CreatesProductAndAudit(t *testing.T) {
	t.Parallel()

	repo := &stubRepository{}
	service := NewService(repo)

	result, err := service.CreateProductWithTransaction(context.Background(), CreateProductRequest{
		Name:        "USB-C Dock",
		Price:       149.99,
		InStock:     true,
		Description: "Dock with dual monitor support",
	})
	if err != nil {
		t.Fatalf("CreateProductWithTransaction returned error: %v", err)
	}

	if !repo.createWithAuditCalled {
		t.Fatal("expected CreateWithAudit to be called")
	}

	if repo.createWithAuditProduct == nil {
		t.Fatal("expected product payload to be captured")
	}

	if repo.createWithAuditAudit == nil {
		t.Fatal("expected audit payload to be captured")
	}

	if result.Product.ID.IsZero() {
		t.Fatal("expected created product id to be set")
	}

	if result.Audit.ID.IsZero() {
		t.Fatal("expected audit id to be set")
	}

	if result.Audit.ProductID != result.Product.ID {
		t.Fatalf("expected audit product id %s, got %s", result.Product.ID.Hex(), result.Audit.ProductID.Hex())
	}

	if result.Audit.Action != "product.created" {
		t.Fatalf("expected audit action %q, got %q", "product.created", result.Audit.Action)
	}

	if result.Product.Name != "USB-C Dock" {
		t.Fatalf("expected name %q, got %q", "USB-C Dock", result.Product.Name)
	}
}

func TestServiceReplaceProduct_UsesReplaceOneAndReturnsUpdatedProduct(t *testing.T) {
	t.Parallel()

	repo := &stubRepository{}
	service := NewService(repo)

	productID := primitive.NewObjectID()
	request := ReplaceProductRequest{
		Name:        "Mechanical Keyboard",
		Price:       129.99,
		InStock:     true,
		Description: "Hot-swappable keyboard",
	}

	product, err := service.ReplaceProduct(context.Background(), productID.Hex(), request)
	if err != nil {
		t.Fatalf("ReplaceProduct returned error: %v", err)
	}

	if !repo.replaceOneCalled {
		t.Fatal("expected ReplaceOne to be called")
	}

	if repo.replaceOneID != productID {
		t.Fatalf("expected ReplaceOne id %s, got %s", productID.Hex(), repo.replaceOneID.Hex())
	}

	if repo.replaceOneProduct == nil {
		t.Fatal("expected ReplaceOne product payload to be captured")
	}

	if repo.replaceOneProduct.Name != request.Name {
		t.Fatalf("expected name %q, got %q", request.Name, repo.replaceOneProduct.Name)
	}

	if repo.replaceOneProduct.Price != request.Price {
		t.Fatalf("expected price %v, got %v", request.Price, repo.replaceOneProduct.Price)
	}

	if repo.replaceOneProduct.InStock != request.InStock {
		t.Fatalf("expected in_stock %v, got %v", request.InStock, repo.replaceOneProduct.InStock)
	}

	if repo.replaceOneProduct.Description != request.Description {
		t.Fatalf("expected description %q, got %q", request.Description, repo.replaceOneProduct.Description)
	}

	if product.ID != productID {
		t.Fatalf("expected returned id %s, got %s", productID.Hex(), product.ID.Hex())
	}

	if product.Name != request.Name || product.Price != request.Price || product.InStock != request.InStock || product.Description != request.Description {
		t.Fatal("returned product does not match replacement request")
	}
}

func TestServiceListProducts_UsesValidatedParamsAndReturnsProducts(t *testing.T) {
	t.Parallel()

	inStock := true
	expectedProducts := []Product{
		{
			ID:          primitive.NewObjectID(),
			Name:        "Mechanical Keyboard",
			Price:       129.99,
			InStock:     true,
			Description: "Hot-swappable keyboard",
		},
	}

	repo := &stubRepository{
		listResult: expectedProducts,
	}
	service := NewService(repo)

	products, err := service.ListProducts(context.Background(), ListProductsParams{
		Limit:   0,
		Skip:    5,
		InStock: &inStock,
	})
	if err != nil {
		t.Fatalf("ListProducts returned error: %v", err)
	}

	if !repo.listCalled {
		t.Fatal("expected List to be called")
	}

	if repo.listParams.Limit != DefaultListLimit {
		t.Fatalf("expected limit %d, got %d", DefaultListLimit, repo.listParams.Limit)
	}

	if repo.listParams.Skip != 5 {
		t.Fatalf("expected skip 5, got %d", repo.listParams.Skip)
	}

	if repo.listParams.InStock == nil {
		t.Fatal("expected in_stock filter to be forwarded")
	}

	if *repo.listParams.InStock != inStock {
		t.Fatalf("expected in_stock %v, got %v", inStock, *repo.listParams.InStock)
	}

	if len(products) != len(expectedProducts) {
		t.Fatalf("expected %d products, got %d", len(expectedProducts), len(products))
	}

	if products[0].ID != expectedProducts[0].ID {
		t.Fatalf("expected first product id %s, got %s", expectedProducts[0].ID.Hex(), products[0].ID.Hex())
	}

	if products[0].Name != expectedProducts[0].Name {
		t.Fatalf("expected first product name %q, got %q", expectedProducts[0].Name, products[0].Name)
	}
}

func TestServiceCreateProduct_CallsCreate(t *testing.T) {
	t.Parallel()

	repo := &stubRepository{}
	service := NewService(repo)

	product, err := service.CreateProduct(context.Background(), CreateProductRequest{
		Name:        "Gaming Mouse",
		Price:       59.99,
		InStock:     true,
		Description: "High-precision gaming mouse",
	})

	if err != nil {
		t.Fatalf("CreateProduct returned error: %v", err)
	}

	if product.Name != "Gaming Mouse" {
		t.Fatalf("expected name: %q, got: %q", "Gaming Mouse", product.Name)
	}

	if product.Price != 59.99 {
		t.Fatalf("expected price: %v, got: %v", 59.99, product.Price)
	}

	if product.InStock != true {
		t.Fatalf("expected in_stock: %v, got: %v", true, product.InStock)
	}

	if product.Description != "High-precision gaming mouse" {
		t.Fatalf("expected description: %q, got: %q", "High-precision gaming mouse", product.Description)
	}
}

func TestServiceCreateProduct_WithoutName(t *testing.T) {
	t.Parallel()

	repo := &stubRepository{}
	service := NewService(repo)

	_, err := service.CreateProduct(context.Background(), CreateProductRequest{
		Price:       129.99,
		InStock:     true,
		Description: "Hot-swappable keyboard",
	})

	if err == nil {
		t.Fatalf("expected error when creating product without name")
	}
}

func TestServiceCreateProduct_WithoutPrice(t *testing.T) {
	t.Parallel()

	repo := &stubRepository{}
	service := NewService(repo)

	_, err := service.CreateProduct(context.Background(), CreateProductRequest{
		Name:        "Game Mouse 4999",
		InStock:     true,
		Description: "Cool mouse",
	})

	if err == nil {
		t.Fatalf("expected error when creating product without price")
	}
}
