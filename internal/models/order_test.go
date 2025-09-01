package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestOrder_Validate(t *testing.T) {
	tests := []struct {
		name    string
		order   Order
		wantErr bool
	}{
		{
			name: "valid order",
			order: Order{
				OrderUID:        "test123",
				TrackNumber:     "TRACK123",
				Entry:           "WBIL",
				CustomerID:      "test_customer",
				DeliveryService: "meest",
				DateCreated:     time.Now(),
				Delivery: Delivery{
					Name:    "Test Testov",
					Phone:   "+9720000000",
					Zip:     "2639809",
					City:    "Kiryat Mozkin",
					Address: "Ploshad Mira 15",
					Region:  "Kraiot",
					Email:   "test@gmail.com",
				},
				Payment: Payment{
					Transaction:  "test123",
					Currency:     "USD",
					Provider:     "wbpay",
					Bank:         "alpha",
					Amount:       1000,
					PaymentDt:    1637900000,
					DeliveryCost: 150,
					GoodsTotal:   850,
					CustomFee:    0,
				},
				Items: []Item{
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
			},
			wantErr: false,
		},
		{
			name:    "empty order_uid",
			order:   Order{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.order.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
