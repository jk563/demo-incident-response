package handler

import (
	"encoding/json"
	"log/slog"
	"math"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/example/demo-incident-response/demo-order-api/internal/discount"
	"github.com/example/demo-incident-response/demo-order-api/internal/model"
	"github.com/example/demo-incident-response/demo-order-api/internal/store"
)

// Orders groups the HTTP handlers for order operations.
type Orders struct {
	store *store.OrderStore
}

// NewOrders creates an Orders handler with the given store.
func NewOrders(s *store.OrderStore) *Orders {
	return &Orders{store: s}
}

// Create handles POST /orders.
func (h *Orders) Create(w http.ResponseWriter, r *http.Request) {
	var req model.CreateOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if len(req.Items) == 0 {
		writeError(w, http.StatusBadRequest, "order must contain at least one item")
		return
	}

	// Calculate subtotal.
	var subtotal float64
	for _, item := range req.Items {
		subtotal += float64(item.Quantity) * item.UnitPrice
	}
	subtotal = roundPence(subtotal)

	// Apply discount if a code was provided.
	var discountAmount float64
	if req.DiscountCode != "" {
		tier, ok := discount.Lookup(req.DiscountCode) // panics for WELCOME
		if !ok {
			writeError(w, http.StatusBadRequest, "unrecognised discount code")
			return
		}
		discountAmount = roundPence(subtotal * tier.Rate)
	}

	now := time.Now().UTC()
	order := model.Order{
		ID:             uuid.New().String(),
		Items:          req.Items,
		Subtotal:       subtotal,
		DiscountCode:   req.DiscountCode,
		DiscountAmount: discountAmount,
		Total:          roundPence(subtotal - discountAmount),
		Status:         model.StatusConfirmed,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := h.store.Create(r.Context(), order); err != nil {
		slog.ErrorContext(r.Context(), "failed to create order", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to create order")
		return
	}

	writeJSON(w, http.StatusCreated, order)
}

// Get handles GET /orders/{id}.
func (h *Orders) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	order, err := h.store.Get(r.Context(), id)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to get order", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to get order")
		return
	}
	if order == nil {
		writeError(w, http.StatusNotFound, "order not found")
		return
	}

	writeJSON(w, http.StatusOK, order)
}

// List handles GET /orders with optional ?status= filter.
func (h *Orders) List(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")

	orders, err := h.store.List(r.Context(), status)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to list orders", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to list orders")
		return
	}
	if orders == nil {
		orders = []model.Order{}
	}

	writeJSON(w, http.StatusOK, orders)
}

// Refund handles POST /orders/{id}/refund.
func (h *Orders) Refund(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	order, err := h.store.Get(r.Context(), id)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to get order for refund", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to process refund")
		return
	}
	if order == nil {
		writeError(w, http.StatusNotFound, "order not found")
		return
	}

	if order.Status == model.StatusRefunded {
		writeError(w, http.StatusConflict, "order already refunded")
		return
	}

	order.Status = model.StatusRefunded
	order.UpdatedAt = time.Now().UTC()

	if err := h.store.Update(r.Context(), *order); err != nil {
		slog.ErrorContext(r.Context(), "failed to update order", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to process refund")
		return
	}

	writeJSON(w, http.StatusOK, order)
}

// roundPence rounds to 2 decimal places.
func roundPence(v float64) float64 {
	return math.Round(v*100) / 100
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
