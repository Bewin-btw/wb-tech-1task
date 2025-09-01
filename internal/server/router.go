package server

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
	"net/http"
	"os"
	"time"

	"wb-tech-1task/internal/service"
)

func NewRouter(svc *service.OrderService, logger *zap.Logger) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(ZapLogger(logger))
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(15 * time.Second))

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	r.Get("/ready", readinessHandler(svc, logger))

	h := NewHandler(svc, logger)
	r.Get("/order", h.GetOrder)
	r.Post("/order", h.CreateOrder)

	var fs http.Handler

	if _, err := os.Stat("web/static"); err == nil {
		fs = http.FileServer(http.Dir("web/static"))
	} else if _, err := os.Stat("/web/static"); err == nil {
		fs = http.FileServer(http.Dir("/web/static"))
	} else {
		fs = http.NotFoundHandler()
	}

	r.Handle("/*", fs)

	return r
}
