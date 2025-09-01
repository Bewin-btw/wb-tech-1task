package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"go.uber.org/zap"

	"wb-tech-1task/internal/models"
	"wb-tech-1task/internal/service"
)

type Handler struct {
	orderService *service.OrderService
	logger       *zap.Logger
}

func NewHandler(orderService *service.OrderService, logger *zap.Logger) *Handler {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Handler{
		orderService: orderService,
		logger:       logger,
	}
}

func (h *Handler) GetOrder(w http.ResponseWriter, r *http.Request) {
	orderUID := r.URL.Query().Get("uid")
	if orderUID == "" {
		http.Error(w, "OrderUID is required", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	order, err := h.orderService.GetOrder(ctx, orderUID)
	if err != nil {
		if errors.Is(err, service.ErrOrderNotFound) {
			http.Error(w, "Order not found", http.StatusNotFound)
		} else {
			http.Error(w, "Internal error", http.StatusInternalServerError)
			h.logger.Error("error on getting order", zap.String("order_uid", orderUID), zap.Error(err))
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(order); err != nil {
		h.logger.Error("failed to encode order", zap.String("order_uid", orderUID), zap.Error(err))
	}
}

func (h *Handler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	var order models.Order
	if err := json.NewDecoder(r.Body).Decode(&order); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if err := order.Validate(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := h.orderService.SaveOrder(ctx, &order); err != nil {
		if errors.Is(err, service.ErrOrderExists) {
			http.Error(w, "Order already exists", http.StatusBadRequest)
		} else {
			http.Error(w, "Internal error", http.StatusInternalServerError)
		}
		h.logger.Error("error on saving order", zap.String("order_uid", order.OrderUID), zap.Error(err))
		return
	}

	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(order); err != nil {
		h.logger.Error("failed to encode response for saved order", zap.String("order_uid", order.OrderUID), zap.Error(err))
	}
}

func readinessHandler(svc *service.OrderService, logger *zap.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()
		if _, err := svc.GetAllOrders(ctx); err != nil {
			logger.Warn("readiness check failed", zap.Error(err))
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}
