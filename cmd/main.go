package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go-mongodb/internal/product"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Config struct {
	MongoURI       string
	DatabaseName   string
	CollectionName string
	HTTPAddr       string
}

func main() {
	cfg := loadConfig()

	connectCtx, connectCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer connectCancel()

	client, err := connectMongo(connectCtx, cfg.MongoURI)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := client.Disconnect(context.Background()); err != nil {
			log.Println("disconnect error:", err)
		}
	}()

	collection := client.Database(cfg.DatabaseName).Collection(cfg.CollectionName)
	if err := ensureIndexes(connectCtx, collection); err != nil {
		log.Fatal(err)
	}

	repository := product.NewMongoRepository(collection)
	service := product.NewService(repository)
	handler := product.NewHandler(service)

	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)

	router.Group(func(r chi.Router) {
		r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"status":"ok"}`))
		})
	})

	router.Mount("/", handler.Routes())

	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	fmt.Printf("MongoDB connected: %s\n", cfg.MongoURI)
	fmt.Printf("HTTP server listening on http://localhost%s\n", cfg.HTTPAddr)
	fmt.Println("Available routes:")
	fmt.Println("GET  /healthz")
	fmt.Println("GET  /products")
	fmt.Println("GET  /products?limit=10&skip=0&in_stock=true")
	fmt.Println("GET  /products/{id}")
	fmt.Println("POST /products (single object or array)")
	fmt.Println("PUT  /products/bulk")
	fmt.Println("PATCH /products/bulk")
	fmt.Println("PUT /products/{id}")
	fmt.Println("PATCH /products/{id}")
	fmt.Println("DELETE /products/{id}")

	serverErrors := make(chan error, 1)
	go func() {
		serverErrors <- server.ListenAndServe()
	}()

	shutdownCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	select {
	case err := <-serverErrors:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err)
		}
	case <-shutdownCtx.Done():
		log.Println("shutdown signal received")
	}

	gracefulCtx, gracefulCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer gracefulCancel()

	if err := server.Shutdown(gracefulCtx); err != nil {
		log.Fatal(err)
	}
}

func loadConfig() Config {
	return Config{
		MongoURI:       envOrDefault("MONGODB_URI", "mongodb://localhost:27017"),
		DatabaseName:   envOrDefault("MONGODB_DATABASE", "store"),
		CollectionName: envOrDefault("MONGODB_COLLECTION", "products"),
		HTTPAddr:       envOrDefault("HTTP_ADDR", ":8080"),
	}
}

func envOrDefault(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	return value
}

func connectMongo(ctx context.Context, uri string) (*mongo.Client, error) {
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, err
	}

	if err := client.Ping(ctx, nil); err != nil {
		return nil, err
	}

	return client, nil
}

func ensureIndexes(ctx context.Context, collection *mongo.Collection) error {
	_, err := collection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "name", Value: 1}},
		Options: options.Index().SetUnique(true).SetName("unique_product_name"),
	})

	return err
}
