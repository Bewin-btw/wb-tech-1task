package service

import (
	"context"
	"wb-tech-1task/internal/models"
)

type OrderCache interface {
	Set(order *models.Order) error
	Get(orderUID string) (*models.Order, bool, error)
	GetAll() (map[string]*models.Order, error)
	Delete(orderUID string)
	Count() int
	DBBackup(orders []*models.Order) error
}

type OrderRepository interface {
	GetOrder(ctx context.Context, orderUID string) (*models.Order, error)
	SaveOrder(ctx context.Context, order *models.Order) error
	GetAllOrders(ctx context.Context) ([]*models.Order, error)
	Close() error
}

type OrderService struct {
	cache OrderCache
	repo  OrderRepository
}

func NewOrderService(cache OrderCache, repo OrderRepository) *OrderService {
	return &OrderService{
		cache,
		repo,
	}
}

func (s *OrderService) GetOrder(ctx context.Context, orderUID string) (*models.Order, error) {
	order, exists, err :=
}
