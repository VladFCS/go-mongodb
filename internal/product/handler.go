package product

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	service *Service
}

// NewHandler constructs the HTTP layer for product routes.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// Routes registers all product HTTP endpoints.
func (h *Handler) Routes() http.Handler {
	router := chi.NewRouter()

	router.Route("/products", func(r chi.Router) {
		r.Get("/", h.ListProducts)
		r.Post("/", h.CreateProduct)
		r.Post("/transaction", h.CreateProductTransaction)
		r.Put("/bulk", h.ReplaceProducts)
		r.Patch("/bulk", h.UpdateProducts)

		r.Route("/{id}", func(r chi.Router) {
			r.Get("/", h.GetProductByID)
			r.Put("/", h.ReplaceProduct)
			r.Patch("/", h.UpdateProduct)
			r.Delete("/", h.DeleteProduct)
		})
	})

	return router
}

// ListProducts handles GET /products.
func (h *Handler) ListProducts(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	params, err := parseListProductsParams(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	products, err := h.service.ListProducts(ctx, params)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, products)
}

// GetProductByID handles GET /products/{id}.
func (h *Handler) GetProductByID(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	id := chi.URLParam(r, "id")

	product, err := h.service.GetProductByID(ctx, id)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, product)
}

// CreateProduct handles POST /products for one or many create payloads.
func (h *Handler) CreateProduct(w http.ResponseWriter, r *http.Request) {
	defer func() {
		_ = r.Body.Close()
	}()

	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	switch detectJSONPayload(body) {
	case '{':
		var request CreateProductRequest
		if err := decodeStrictJSON(body, &request); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}

		product, err := h.service.CreateProduct(ctx, request)
		if err != nil {
			writeServiceError(w, err)
			return
		}

		writeJSON(w, http.StatusCreated, product)
	case '[':
		var requests []CreateProductRequest
		if err := decodeStrictJSON(body, &requests); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}

		products, err := h.service.CreateProducts(ctx, requests)
		if err != nil {
			writeServiceError(w, err)
			return
		}

		writeJSON(w, http.StatusCreated, products)
	default:
		writeError(w, http.StatusBadRequest, "invalid JSON body")
	}
}

// CreateProductTransaction handles POST /products/transaction for a transactional create example.
func (h *Handler) CreateProductTransaction(w http.ResponseWriter, r *http.Request) {
	defer func() {
		_ = r.Body.Close()
	}()

	var request CreateProductRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	result, err := h.service.CreateProductWithTransaction(ctx, request)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, result)
}

// ReplaceProduct handles full replacement via service.ReplaceProduct / MongoDB ReplaceOne.
func (h *Handler) ReplaceProduct(w http.ResponseWriter, r *http.Request) {
	defer func() {
		_ = r.Body.Close()
	}()

	id := chi.URLParam(r, "id")

	var request ReplaceProductRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	product, err := h.service.ReplaceProduct(ctx, id, request)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, product)
}

// ReplaceProducts handles bulk full-field replacement semantics via MongoDB UpdateMany.
func (h *Handler) ReplaceProducts(w http.ResponseWriter, r *http.Request) {
	defer func() {
		_ = r.Body.Close()
	}()

	var request ReplaceProductsRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if err := h.service.ReplaceProducts(ctx, request); err != nil {
		writeServiceError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// UpdateProduct handles partial updates via service.UpdateProduct / MongoDB UpdateOne.
func (h *Handler) UpdateProduct(w http.ResponseWriter, r *http.Request) {
	defer func() {
		_ = r.Body.Close()
	}()

	id := chi.URLParam(r, "id")

	var request UpdateProductRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	product, err := h.service.UpdateProduct(ctx, id, request)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, product)
}

// UpdateProducts handles bulk partial updates via service.UpdateProducts / MongoDB UpdateMany.
func (h *Handler) UpdateProducts(w http.ResponseWriter, r *http.Request) {
	defer func() {
		_ = r.Body.Close()
	}()

	var request UpdateProductsRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if err := h.service.UpdateProducts(ctx, request); err != nil {
		writeServiceError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// DeleteProduct handles DELETE /products/{id}.
func (h *Handler) DeleteProduct(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	id := chi.URLParam(r, "id")

	if err := h.service.DeleteProduct(ctx, id); err != nil {
		writeServiceError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// writeJSON writes a JSON response payload with the given HTTP status.
func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

// writeError writes a standard JSON error response.
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

// writeServiceError maps domain errors to HTTP status codes.
func writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrInvalidProduct), errors.Is(err, ErrInvalidID), errors.Is(err, ErrInvalidQuery):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, ErrNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, ErrDuplicateName):
		writeError(w, http.StatusConflict, err.Error())
	default:
		writeError(w, http.StatusInternalServerError, err.Error())
	}
}

// parseListProductsParams parses supported query params from the request URL.
func parseListProductsParams(r *http.Request) (ListProductsParams, error) {
	query := r.URL.Query()
	params := ListProductsParams{}

	if limitValue := query.Get("limit"); limitValue != "" {
		limit, err := strconv.Atoi(limitValue)
		if err != nil {
			return ListProductsParams{}, fmt.Errorf("%w: limit must be an integer", ErrInvalidQuery)
		}
		params.Limit = limit
	}

	if skipValue := query.Get("skip"); skipValue != "" {
		skip, err := strconv.Atoi(skipValue)
		if err != nil {
			return ListProductsParams{}, fmt.Errorf("%w: skip must be an integer", ErrInvalidQuery)
		}
		params.Skip = skip
	}

	if inStockValue := query.Get("in_stock"); inStockValue != "" {
		inStock, err := strconv.ParseBool(inStockValue)
		if err != nil {
			return ListProductsParams{}, fmt.Errorf("%w: in_stock must be true or false", ErrInvalidQuery)
		}
		params.InStock = &inStock
	}

	return params, nil
}

// detectJSONPayload inspects the first non-space byte to distinguish object vs array payloads.
func detectJSONPayload(body []byte) byte {
	trimmedBody := bytes.TrimSpace(body)
	if len(trimmedBody) == 0 {
		return 0
	}

	return trimmedBody[0]
}

// decodeStrictJSON decodes one JSON value and rejects unknown fields or trailing payloads.
func decodeStrictJSON[T any](body []byte, target *T) error {
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(target); err != nil {
		return err
	}

	var extra json.RawMessage
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return errors.New("unexpected extra JSON values")
		}

		return err
	}

	return nil
}
