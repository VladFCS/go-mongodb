# Go + MongoDB HTTP Example

This project is a small example of MongoDB usage in Go with a layered structure:

- `repository` talks to MongoDB
- `service` contains business logic and validation
- `handler` exposes HTTP endpoints
- `chi` handles routing and middleware

## Project structure

```text
.
├── cmd
│   └── main.go
├── docker-compose.yml
├── internal
│   └── product
│       ├── handler.go
│       ├── model.go
│       ├── repository.go
│       └── service.go
```

## Run MongoDB locally

```bash
docker compose up -d
```

MongoDB will be available at `mongodb://localhost:27017`.

## Run the API

```bash
go run ./cmd
```

The server starts on `http://localhost:8080`.

## Endpoints

### POST /products

Create a product:

```bash
curl -X POST http://localhost:8080/products \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Mechanical Keyboard",
    "price": 99.99,
    "in_stock": true,
    "description": "Keyboard with tactile switches"
  }'
```

### GET /products

Fetch all products:

```bash
curl http://localhost:8080/products
```

### GET /healthz

Health check:

```bash
curl http://localhost:8080/healthz
```

## Optional environment variables

```bash
export MONGODB_URI="mongodb://localhost:27017"
export MONGODB_DATABASE="store"
export MONGODB_COLLECTION="products"
export HTTP_ADDR=":8080"
go run ./cmd
```

## Stop MongoDB

```bash
docker compose down
```

Remove data too:

```bash
docker compose down -v
```
