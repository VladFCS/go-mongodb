package product

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Routes() http.Handler {
	router := chi.NewRouter()

	router.Route("/products", func(r chi.Router) {
		r.Get("/", h.ListProducts)
		r.Post("/", h.CreateProduct)

		r.Route("/{id}", func(r chi.Router) {
			r.Get("/", h.GetProductByID)
			r.Put("/", h.ReplaceProduct)
			r.Patch("/", h.PatchProduct)
			r.Delete("/", h.DeleteProduct)
		})
	})

	return router
}

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

func (h *Handler) CreateProduct(w http.ResponseWriter, r *http.Request) {
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

	product, err := h.service.CreateProduct(ctx, request)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, product)
}

func (h *Handler) ReplaceProduct(w http.ResponseWriter, r *http.Request) {
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

	product, err := h.service.ReplaceProduct(ctx, id, request)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, product)
}

func (h *Handler) PatchProduct(w http.ResponseWriter, r *http.Request) {
	defer func() {
		_ = r.Body.Close()
	}()

	id := chi.URLParam(r, "id")

	var request PatchProductRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	product, err := h.service.PatchProduct(ctx, id, request)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, product)
}

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

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

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
