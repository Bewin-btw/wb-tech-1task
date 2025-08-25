package service

import (
	"context"
	"log"
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
	if order, exists, err := s.cache.Get(orderUID); err == nil && exists {
		return order, nil
	} // кеш сразу отдаю без обработок

	order, err := s.repo.GetOrder(ctx, orderUID)
	if err != nil {
		return nil, err
	}

	if err = s.cache.Set(order); err != nil {
		log.Printf("failed to add order %s in cache: %v", orderUID, err)
	}

	return order, nil
}

func (s *OrderService) SaveOrder(ctx context.Context, order *models.Order) error {
	if _, err := s.repo.GetOrder(ctx, order.OrderUID); err == nil {
		return ErrOrderExists
	}

	if err := s.repo.SaveOrder(ctx, order); err != nil {
		return err
	}

	if err := s.cache.Set(order); err != nil {
		log.Printf("failed to add order %s in cache: %v", order.OrderUID, err)
	}

	return nil
}

func (s *OrderService) GetAllOrders(ctx context.Context) ([]*models.Order, error) {
	return s.repo.GetAllOrders(ctx)
}
