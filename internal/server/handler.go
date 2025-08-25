package server

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"path/filepath"
	"wb-tech-1task/internal/models"
	"wb-tech-1task/internal/service"
)

type Handler struct {
	orderService *service.OrderService
}

func NewHandler(orderService *service.OrderService) *Handler {
	return &Handler{
		orderService,
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/":
		h.serveHTML(w, r)
	case "/script.js":
		h.serveJS(w, r)
	case "/order":
		if r.Method == http.MethodGet {
			h.GetOrder(w, r)
		} else if r.Method == http.MethodPost {
			h.CreateOrder(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	default:
		http.Error(w, "Not found", http.StatusNotFound)
	}
}

func (h *Handler) GetOrder(w http.ResponseWriter, r *http.Request) {
	orderUID := r.URL.Query().Get("uid")
	if orderUID == "" {
		http.Error(w, "OrderUID is required", http.StatusBadRequest)
		return
	}

	order, err := h.orderService.GetOrder(r.Context(), orderUID)
	if err != nil {
		if errors.Is(err, service.ErrOrderNotFound) {
			http.Error(w, "Order not found", http.StatusNotFound)
		} else {
			http.Error(w, "Internal error", http.StatusInternalServerError)
			log.Printf("error on getting order: %v", err)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(order)
	if err != nil {
		log.Printf("error on encoding order: %v", err)
	}
	return
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

	if err := h.orderService.SaveOrder(r.Context(), &order); err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		log.Printf("error on saving order: %v", err)
		return
	}

	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(order); err != nil {
		log.Printf("failed to encode order while sending: %v", err)
	}
}

// СДЕЛАТЬ НЕ КОНСТ, ВЫНЕСТИ В ПЕРЕМЕННУЮ ОКР!!!
func (h *Handler) serveHTML(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	http.ServeFile(w, r, filepath.Join("web", "static", "index.html"))
}

func (h *Handler) serveJS(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/javascript")
	http.ServeFile(w, r, filepath.Join("web", "static", "script.js"))
}
