package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go-mongodb/internal/product"
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

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := connectMongo(ctx, cfg.MongoURI)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := client.Disconnect(context.Background()); err != nil {
			log.Println("disconnect error:", err)
		}
	}()

	collection := client.Database(cfg.DatabaseName).Collection(cfg.CollectionName)
	repository := product.NewMongoRepository(collection)
	service := product.NewService(repository)
	handler := product.NewHandler(service)

	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)

	router.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	router.Mount("/products", handler.Routes())

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
	fmt.Println("POST /products")

	log.Fatal(server.ListenAndServe())
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
