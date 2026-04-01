# Go + MongoDB HTTP Example

This project is a small example of MongoDB usage in Go with a layered structure:

- `repository` talks to MongoDB
- `service` contains business logic and request validation
- `handler` exposes HTTP endpoints
- `chi` handles routing and middleware
- MongoDB enforces a unique index on product `name`

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

Create one product:

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

Create multiple products in one request:

```bash
curl -X POST http://localhost:8080/products \
  -H "Content-Type: application/json" \
  -d '[
    {
      "name": "Mechanical Keyboard",
      "price": 99.99,
      "in_stock": true,
      "description": "Keyboard with tactile switches"
    },
    {
      "name": "Wireless Mouse",
      "price": 49.99,
      "in_stock": true,
      "description": "Mouse with ergonomic shape"
    }
  ]'
```

`POST /products` accepts either a single product object or an array of product objects. A single-object request returns one product, and an array request returns an array of created products.

### GET /products

Fetch all products:

```bash
curl http://localhost:8080/products
```

Fetch products with pagination and filters:

```bash
curl "http://localhost:8080/products?limit=10&skip=0&in_stock=true"
```

Validation rules:

- `name` is required
- `price` must be greater than `0`
- `description` can be at most `500` characters
- product `name` must be unique

Duplicate names return `409 Conflict`.

### GET /products/{id}

Fetch one product by MongoDB id:

```bash
curl http://localhost:8080/products/<product_id>
```

### PUT /products/{id}

Replace the full product. This route demonstrates MongoDB `ReplaceOne`:

```bash
curl -X PUT http://localhost:8080/products/<product_id> \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Mechanical Keyboard V2",
    "price": 129.99,
    "in_stock": true,
    "description": "Full replacement payload"
  }'
```

### PUT /products/bulk

Apply the same full field set to multiple products:

```bash
curl -X PUT http://localhost:8080/products/bulk \
  -H "Content-Type: application/json" \
  -d '{
    "ids": [
      "<product_id_1>",
      "<product_id_2>"
    ],
    "product": {
      "name": "Mechanical Keyboard V2",
      "price": 129.99,
      "in_stock": true,
      "description": "Shared replacement payload"
    }
  }'
```

MongoDB has `ReplaceOne`, but not `ReplaceMany`, so this bulk `PUT` path uses `UpdateMany` with `$set` under the hood.

### PATCH /products/{id}

Update only the fields you send. This route demonstrates MongoDB `UpdateOne`:

```bash
curl -X PATCH http://localhost:8080/products/<product_id> \
  -H "Content-Type: application/json" \
  -d '{
    "price": 109.99
  }'
```

`PATCH` is partial. Omitted fields keep their existing values.

### PATCH /products/bulk

Patch multiple products with the same fields. This route demonstrates MongoDB `UpdateMany`:

```bash
curl -X PATCH http://localhost:8080/products/bulk \
  -H "Content-Type: application/json" \
  -d '{
    "ids": [
      "<product_id_1>",
      "<product_id_2>"
    ],
    "product": {
      "price": 109.99
    }
  }'
```

### DELETE /products/{id}

Delete a product:

```bash
curl -X DELETE http://localhost:8080/products/<product_id>
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

If port `8080` is already in use, run the API on another port:

```bash
HTTP_ADDR=:18080 go run ./cmd
```

## Stop MongoDB

```bash
docker compose down
```

Remove data too:

```bash
docker compose down -v
```
