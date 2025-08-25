package main

import (
	"context"
	"log"
	"time"
	cache2 "wb-tech-1task/internal/cache"
	"wb-tech-1task/internal/db/postgres"
	"wb-tech-1task/internal/service"
)

func main() {
	cache := cache2.New(10 * time.Minute)

	repo, err := postgres.NewPostgresRepository("postgres://user:password@localhost:5432/wb_orders?sslmode=disable")
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

	//f
}
