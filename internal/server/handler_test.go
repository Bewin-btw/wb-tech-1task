package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"

	"wb-tech-1task/internal/models"
	"wb-tech-1task/internal/service"
	"wb-tech-1task/internal/service/mocks"
)

func TestHandler_GetOrder(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCache := mocks.NewMockOrderCache(ctrl)
	mockRepo := mocks.NewMockOrderRepository(ctrl)
	logger := zap.NewNop()

	realService := service.NewOrderService(mockCache, mockRepo, logger)
	handler := NewHandler(realService, logger)

	fixedTime := time.Date(2022, time.January, 1, 12, 0, 0, 0, time.UTC)

	testOrder := &models.Order{
		OrderUID:          "test123",
		TrackNumber:       "TRACK123",
		Entry:             "WBIL",
		Locale:            "en",
		InternalSignature: "",
		CustomerID:        "test_customer",
		DeliveryService:   "meest",
		Shardkey:          "9",
		SmID:              99,
		DateCreated:       fixedTime,
		OofShard:          "1",
		Delivery: models.Delivery{
			Name:    "Test Testov",
			Phone:   "+9720000000",
			Zip:     "2639809",
			City:    "Kiryat Mozkin",
			Address: "Ploshad Mira 15",
			Region:  "Kraiot",
			Email:   "test@gmail.com",
		},
		Payment: models.Payment{
			Transaction:  "test123",
			RequestID:    "",
			Currency:     "USD",
			Provider:     "wbpay",
			Amount:       1817,
			PaymentDt:    1637900000,
			Bank:         "alpha",
			DeliveryCost: 1500,
			GoodsTotal:   317,
			CustomFee:    0,
		},
		Items: []models.Item{
			{
				ChrtID:      9934930,
				TrackNumber: "TRACK123",
				Price:       453,
				Rid:         "ab4219087a764ae0btest",
				Name:        "Mascaras",
				Sale:        30,
				Size:        "0",
				TotalPrice:  317,
				NmID:        2389212,
				Brand:       "Vivienne Sabo",
				Status:      202,
			},
		},
	}

	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		mockCache.EXPECT().Get("test123").Return(testOrder, true, nil)

		req := httptest.NewRequest("GET", "/order?uid=test123", nil)
		w := httptest.NewRecorder()

		handler.GetOrder(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var response models.Order
		json.NewDecoder(w.Body).Decode(&response)
		assert.Equal(t, "test123", response.OrderUID)
	})

	t.Run("not found", func(t *testing.T) {
		mockCache.EXPECT().Get("notfound").Return(nil, false, nil)
		mockRepo.EXPECT().GetOrder(ctx, "notfound").Return(nil, service.ErrOrderNotFound)

		req := httptest.NewRequest("GET", "/order?uid=notfound", nil)
		w := httptest.NewRecorder()

		handler.GetOrder(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestHandler_CreateOrder(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCache := mocks.NewMockOrderCache(ctrl)
	mockRepo := mocks.NewMockOrderRepository(ctrl)
	logger := zap.NewNop()

	realService := service.NewOrderService(mockCache, mockRepo, logger)
	handler := NewHandler(realService, logger)

	fixedTime := time.Date(2022, time.January, 1, 12, 0, 0, 0, time.UTC)

	testOrder := models.Order{
		OrderUID:          "test123",
		TrackNumber:       "TRACK123",
		Entry:             "WBIL",
		Locale:            "en",
		InternalSignature: "",
		CustomerID:        "test_customer",
		DeliveryService:   "meest",
		Shardkey:          "9",
		SmID:              99,
		DateCreated:       fixedTime,
		OofShard:          "1",
		Delivery: models.Delivery{
			Name:    "Test Testov",
			Phone:   "+9720000000",
			Zip:     "2639809",
			City:    "Kiryat Mozkin",
			Address: "Ploshad Mira 15",
			Region:  "Kraiot",
			Email:   "test@gmail.com",
		},
		Payment: models.Payment{
			Transaction:  "test123",
			RequestID:    "",
			Currency:     "USD",
			Provider:     "wbpay",
			Amount:       1817,
			PaymentDt:    1637900000,
			Bank:         "alpha",
			DeliveryCost: 1500,
			GoodsTotal:   317,
			CustomFee:    0,
		},
		Items: []models.Item{
			{
				ChrtID:      9934930,
				TrackNumber: "TRACK123",
				Price:       453,
				Rid:         "ab4219087a764ae0btest",
				Name:        "Mascaras",
				Sale:        30,
				Size:        "0",
				TotalPrice:  317,
				NmID:        2389212,
				Brand:       "Vivienne Sabo",
				Status:      202,
			},
		},
	}
	orderBody, _ := json.Marshal(testOrder)

	t.Run("success", func(t *testing.T) {
		mockRepo.EXPECT().SaveOrder(gomock.Any(), &testOrder).Return(nil)
		mockCache.EXPECT().Set(&testOrder).Return(nil)

		req := httptest.NewRequest("POST", "/order", bytes.NewReader(orderBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.CreateOrder(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
	})

	t.Run("invalid json", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/order", bytes.NewReader([]byte("invalid")))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.CreateOrder(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}
