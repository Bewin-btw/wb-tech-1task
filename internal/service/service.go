package service

import (
	"context"
	"errors"

	"github.com/lib/pq"
	"go.uber.org/zap"

	"wb-tech-1task/internal/models"
)

var (
	ErrOrderNotFound = errors.New("order not found")
	ErrOrderExists   = errors.New("order already exists")
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
	cache  OrderCache
	repo   OrderRepository
	logger *zap.Logger
}

func NewOrderService(cache OrderCache, repo OrderRepository, logger *zap.Logger) *OrderService {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &OrderService{
		cache:  cache,
		repo:   repo,
		logger: logger,
	}
}

func (s *OrderService) GetOrder(ctx context.Context, orderUID string) (*models.Order, error) {
	order, exists, err := s.cache.Get(orderUID)
	if err != nil {
		s.logger.Warn("cache get failed", zap.String("order_uid", orderUID), zap.Error(err))
	} else if exists {
		s.logger.Debug("cache hit", zap.String("order_uid", orderUID))
		return order, nil
	}

	s.logger.Debug("cache miss; loading from db", zap.String("order_uid", orderUID))
	order, err = s.repo.GetOrder(ctx, orderUID)
	if err != nil {
		if errors.Is(err, ErrOrderNotFound) {
			return nil, ErrOrderNotFound
		}
		s.logger.Error("repo.GetOrder failed", zap.String("order_uid", orderUID), zap.Error(err))
		return nil, err
	}

	if err := s.cache.Set(order); err != nil {
		s.logger.Warn("failed to set order to cache", zap.String("order_uid", orderUID), zap.Error(err))
	}
	return order, nil
}

func (s *OrderService) SaveOrder(ctx context.Context, order *models.Order) error {
	if order == nil {
		return errors.New("nil order")
	}

	if err := s.repo.SaveOrder(ctx, order); err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			s.logger.Info("unique violation on save", zap.String("order_uid", order.OrderUID), zap.Error(err))
			return ErrOrderExists
		}
		s.logger.Error("repo.SaveOrder failed", zap.String("order_uid", order.OrderUID), zap.Error(err))
		return err
	}

	if err := s.cache.Set(order); err != nil {
		s.logger.Warn("cache set failed after save", zap.String("order_uid", order.OrderUID), zap.Error(err))
	}

	s.logger.Info("order saved", zap.String("order_uid", order.OrderUID))
	return nil
}

func (s *OrderService) GetAllOrders(ctx context.Context) ([]*models.Order, error) {
	return s.repo.GetAllOrders(ctx)
}
