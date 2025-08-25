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
	cache2 "wb-tech-1task/internal/cache"
	"wb-tech-1task/internal/db/postgres"
	"wb-tech-1task/internal/kafka"
	"wb-tech-1task/internal/server"
	"wb-tech-1task/internal/service"
)

func main() {
	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")

	connectionString := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		dbUser, dbPassword, dbHost, dbPort, dbName)

	repo, err := postgres.NewPostgresRepository(connectionString)

	cache := cache2.New(10 * time.Minute)

	//repo, err := postgres.NewPostgresRepository("postgres://user:password@postgres:5432/wb_orders?sslmode=disable")
	if err != nil {
		log.Fatal("Failed to created a repo", err)
	}
	defer repo.Close()

	// можно вынести в отдельный сервис
	ctx := context.Background()
	orders, err := repo.GetAllOrders(ctx)
	if err != nil {
		log.Fatal("Failed to get orders from DB:", err)
	}
	if err := cache.DBBackup(orders); err != nil {
		log.Fatal("Failed to restore cache from DB:", err)
	}
	log.Printf("restored %d orders to cache", cache.Count())

	orderService := service.NewOrderService(cache, repo)

	kafkaBrokers := os.Getenv("KAFKA_BROKERS")
	if kafkaBrokers == "" {
		kafkaBrokers = "localhost:9092" // fallback
	}

	consumer := kafka.NewConsumer(
		[]string{kafkaBrokers},
		"orders",
		"order-service-group",
		orderService,
	)

	go consumer.Start(ctx)
	defer consumer.Close()

	handler := server.NewHandler(orderService)

	srv := &http.Server{
		Addr:    ":8080",
		Handler: handler,
	}

	go func() {
		log.Printf("server starting on :8080")
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal("Server failed to start:", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Printf("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Server exited")
}
