package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/lib/pq"
	"go.uber.org/zap"

	"wb-tech-1task/internal/models"
)

type mockRepo struct {
	saveFunc    func(ctx context.Context, order *models.Order) error
	getFunc     func(ctx context.Context, uid string) (*models.Order, error)
	getAllFunc  func(ctx context.Context) ([]*models.Order, error)
	closeCalled bool
}

func (m *mockRepo) GetOrder(ctx context.Context, orderUID string) (*models.Order, error) {
	if m.getFunc != nil {
		return m.getFunc(ctx, orderUID)
	}
	return nil, ErrOrderNotFound
}
func (m *mockRepo) SaveOrder(ctx context.Context, order *models.Order) error {
	if m.saveFunc != nil {
		return m.saveFunc(ctx, order)
	}
	return nil
}
func (m *mockRepo) GetAllOrders(ctx context.Context) ([]*models.Order, error) {
	if m.getAllFunc != nil {
		return m.getAllFunc(ctx)
	}
	return nil, nil
}
func (m *mockRepo) Close() error {
	m.closeCalled = true
	return nil
}

type mockCache struct {
	setFunc func(order *models.Order) error
	getFunc func(uid string) (*models.Order, bool, error)
	dbFunc  func(orders []*models.Order) error
}

func (m *mockCache) Set(order *models.Order) error {
	if m.setFunc != nil {
		return m.setFunc(order)
	}
	return nil
}
func (m *mockCache) Get(uid string) (*models.Order, bool, error) {
	if m.getFunc != nil {
		return m.getFunc(uid)
	}
	return nil, false, nil
}
func (m *mockCache) GetAll() (map[string]*models.Order, error) {
	return nil, nil
}
func (m *mockCache) Delete(uid string) { return }
func (m *mockCache) Count() int        { return 0 }
func (m *mockCache) DBBackup(orders []*models.Order) error {
	if m.dbFunc != nil {
		return m.dbFunc(orders)
	}
	return nil
}

func sampleOrder() *models.Order {
	return &models.Order{
		OrderUID:    "o-123",
		TrackNumber: "track-1",
		Entry:       "ENT",
		Delivery: models.Delivery{
			Name:    "Test",
			Phone:   "+100",
			Zip:     "000",
			City:    "City",
			Address: "Addr",
			Region:  "Reg",
			Email:   "a@b.c",
		},
		Payment: models.Payment{
			Transaction:  "tx-1",
			RequestID:    "",
			Currency:     "USD",
			Provider:     "pay",
			Amount:       100,
			PaymentDt:    time.Now().Unix(),
			Bank:         "bank",
			DeliveryCost: 10,
			GoodsTotal:   90,
			CustomFee:    0,
		},
		Items: []models.Item{
			{
				ChrtID:      1,
				TrackNumber: "track-1",
				Price:       100,
				Rid:         "rid-1",
				Name:        "item1",
				Sale:        0,
				Size:        "M",
				TotalPrice:  100,
				NmID:        11,
				Brand:       "brand",
				Status:      1,
			},
		},
		Locale:            "en",
		InternalSignature: "",
		CustomerID:        "cust-1",
		DeliveryService:   "ds",
		Shardkey:          "1",
		SmID:              0,
		DateCreated:       time.Now(),
		OofShard:          "1",
	}
}

func TestSaveOrder_Success(t *testing.T) {
	ctx := context.Background()
	order := sampleOrder()

	repo := &mockRepo{
		saveFunc: func(ctx context.Context, o *models.Order) error {
			return nil
		},
	}
	cache := &mockCache{
		setFunc: func(o *models.Order) error {
			return nil
		},
	}
	logger := zap.NewNop()
	svc := NewOrderService(cache, repo, logger)
	if err := svc.SaveOrder(ctx, order); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

func TestSaveOrder_DBUniqueViolation(t *testing.T) {
	ctx := context.Background()
	order := sampleOrder()

	repo := &mockRepo{
		saveFunc: func(ctx context.Context, o *models.Order) error {
			return &pq.Error{Code: "23505", Message: "unique violation"}
		},
	}
	cache := &mockCache{
		setFunc: func(o *models.Order) error {
			t.Fatalf("cache.Set should not be called when DB reports unique violation")
			return nil
		},
	}
	logger := zap.NewNop()
	svc := NewOrderService(cache, repo, logger)
	err := svc.SaveOrder(ctx, order)
	if !errors.Is(err, ErrOrderExists) {
		t.Fatalf("expected ErrOrderExists, got %v", err)
	}
}

func TestSaveOrder_CacheFail_IsNonFatal(t *testing.T) {
	ctx := context.Background()
	order := sampleOrder()

	repo := &mockRepo{
		saveFunc: func(ctx context.Context, o *models.Order) error {
			return nil
		},
	}
	cache := &mockCache{
		setFunc: func(o *models.Order) error {
			return errors.New("cache failure")
		},
	}
	logger := zap.NewNop()
	svc := NewOrderService(cache, repo, logger)
	if err := svc.SaveOrder(ctx, order); err != nil {
		t.Fatalf("expected nil error even if cache fails, got %v", err)
	}
}

func TestSaveOrder_DBError_Propagated(t *testing.T) {
	ctx := context.Background()
	order := sampleOrder()

	repo := &mockRepo{
		saveFunc: func(ctx context.Context, o *models.Order) error {
			return errors.New("db down")
		},
	}
	cache := &mockCache{
		setFunc: func(o *models.Order) error {
			t.Fatalf("cache.Set should not be called when DB save fails")
			return nil
		},
	}
	logger := zap.NewNop()
	svc := NewOrderService(cache, repo, logger)
	err := svc.SaveOrder(ctx, order)
	if err == nil || err.Error() != "db down" {
		t.Fatalf("expected db down error, got %v", err)
	}
}
