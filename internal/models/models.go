package models

import (
	"errors"
	"time"
	"unicode/utf8"
)

type Order struct {
	OrderUID          string    `json:"order_uid"`
	TrackNumber       string    `json:"track_number"`
	Entry             string    `json:"entry"`
	Delivery          Delivery  `json:"delivery"`
	Payment           Payment   `json:"payment"`
	Items             []Item    `json:"items"`
	Locale            string    `json:"locale"`
	InternalSignature string    `json:"internal_signature"`
	CustomerID        string    `json:"customer_id"`
	DeliveryService   string    `json:"delivery_service"`
	Shardkey          string    `json:"shardkey"`
	SmID              int       `json:"sm_id"`
	DateCreated       time.Time `json:"date_created"`
	OofShard          string    `json:"oof_shard"`
}

type Delivery struct {
	Name    string `json:"name"`
	Phone   string `json:"phone"`
	Zip     string `json:"zip"`
	City    string `json:"city"`
	Address string `json:"address"`
	Region  string `json:"region"`
	Email   string `json:"email"`
}

type Payment struct {
	OrderUID     string `json:"-"`
	Transaction  string `json:"transaction"`
	RequestID    string `json:"request_id"`
	Currency     string `json:"currency"`
	Provider     string `json:"provider"`
	Amount       int    `json:"amount"`
	PaymentDt    int64  `json:"payment_dt"`
	Bank         string `json:"bank"`
	DeliveryCost int    `json:"delivery_cost"`
	GoodsTotal   int    `json:"goods_total"`
	CustomFee    int    `json:"custom_fee"`
}

type Item struct {
	ChrtID      int    `json:"chrt_id"`
	TrackNumber string `json:"track_number"`
	Price       int    `json:"price"`
	Rid         string `json:"rid"`
	Name        string `json:"name"`
	Sale        int    `json:"sale"`
	Size        string `json:"size"`
	TotalPrice  int    `json:"total_price"`
	NmID        int    `json:"nm_id"`
	Brand       string `json:"brand"`
	Status      int    `json:"status"`
}

func (o *Order) Validate() error {
	if o.OrderUID == "" {
		return errors.New("order_uid is required")
	}
	if utf8.RuneCountInString(o.OrderUID) > 50 {
		return errors.New("order_uid too long")
	}
	if o.TrackNumber == "" {
		return errors.New("track_number is required")
	}
	if o.Entry == "" {
		return errors.New("entry is required")
	}
	if o.CustomerID == "" {
		return errors.New("customer_id is required")
	}
	if o.DeliveryService == "" {
		return errors.New("delivery_service is required")
	}
	if o.DateCreated.IsZero() {
		return errors.New("date_created is required and must be valid datetime")
	}

	if o.Delivery.Name == "" {
		return errors.New("delivery.name is required")
	}
	if o.Delivery.Phone == "" {
		return errors.New("delivery.phone is required")
	}
	if o.Delivery.Zip == "" {
		return errors.New("delivery.zip is required")
	}
	if o.Delivery.City == "" {
		return errors.New("delivery.city is required")
	}
	if o.Delivery.Address == "" {
		return errors.New("delivery.address is required")
	}
	if o.Delivery.Region == "" {
		return errors.New("delivery.region is required")
	}
	if o.Delivery.Email == "" {
		return errors.New("delivery.email is required")
	}

	if o.Payment.Transaction == "" {
		return errors.New("payment.transaction is required")
	}
	if o.Payment.Currency == "" {
		return errors.New("payment.currency is required")
	}
	if o.Payment.Provider == "" {
		return errors.New("payment.provider is required")
	}
	if o.Payment.Bank == "" {
		return errors.New("payment.bank is required")
	}

	if len(o.Items) == 0 {
		return errors.New("items cannot be empty")
	}
	for _, item := range o.Items {
		if item.TrackNumber == "" {
			return errors.New("items.track_number is required")
		}
		if item.Name == "" {
			return errors.New("items.name is required")
		}
		if item.Brand == "" {
			return errors.New("items.brand is required")
		}
		if item.Rid == "" {
			return errors.New("items.rid is required")
		}
		if item.Size == "" {
			return errors.New("items.size is required")
		}
		if item.Price <= 0 {
			return errors.New("items.price must be positive")
		}
		if item.TotalPrice <= 0 {
			return errors.New("items.total_price must be positive")
		}
		if item.ChrtID <= 0 {
			return errors.New("items.chrt_id must be positive")
		}
		if item.NmID <= 0 {
			return errors.New("items.nm_id must be positive")
		}
		if item.Status < 0 {
			return errors.New("items.status cannot be negative")
		}
		if item.Sale < 0 {
			return errors.New("items.sale cannot be negative")
		}
	}

	return nil
}
