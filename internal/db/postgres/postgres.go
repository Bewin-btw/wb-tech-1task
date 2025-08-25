package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	_ "github.com/lib/pq"
	"log"
	"time"
	"wb-tech-1task/internal/models"
	"wb-tech-1task/internal/service"
)

type PostgresRepository struct {
	db *sql.DB
}

func NewPostgresRepository(connectionString string) (*PostgresRepository, error) {
	db, err := sql.Open("postgres", connectionString)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %v", err)
	}

	deadline := time.Now().Add(30 * time.Second)

	for {
		if err = db.Ping(); err == nil {
			break
		}
		if time.Now().After(deadline) {
			db.Close()
			return nil, fmt.Errorf("failed to ping database after retries: %v", err)
		}
		log.Printf("postgres ping failed: %v — retrying...", err)
		time.Sleep(2 * time.Second)
	}

	if err := createTables(db); err != nil {
		return nil, fmt.Errorf("failed to create tables: %v", err)
	}

	return &PostgresRepository{db}, nil
}

func createTables(db *sql.DB) error {
	ordersTable := `
	CREATE TABLE IF NOT EXISTS orders (
		order_uid VARCHAR(50) PRIMARY KEY,
		track_number VARCHAR(50) NOT NULL,
		entry VARCHAR(10) NOT NULL,
		locale VARCHAR(2) NOT NULL,
		internal_signature VARCHAR(100),
		customer_id VARCHAR(50) NOT NULL,
		delivery_service VARCHAR(50) NOT NULL,
		shardkey VARCHAR(10) NOT NULL,
		sm_id INTEGER NOT NULL,
		date_created TIMESTAMP NOT NULL,
		oof_shard VARCHAR(10) NOT NULL
	);`

	deliveryTable := `
	CREATE TABLE IF NOT EXISTS delivery (
		order_uid VARCHAR(50) PRIMARY KEY REFERENCES orders(order_uid) ON DELETE CASCADE,
		name VARCHAR(100) NOT NULL,
		phone VARCHAR(20) NOT NULL,
		zip VARCHAR(10) NOT NULL,
		city VARCHAR(50) NOT NULL,
		address VARCHAR(100) NOT NULL,
		region VARCHAR(50) NOT NULL,
		email VARCHAR(100) NOT NULL
	);`

	paymentTable := `
	CREATE TABLE IF NOT EXISTS payment (
		transaction VARCHAR(50) PRIMARY KEY REFERENCES orders(order_uid) ON DELETE CASCADE,
		request_id VARCHAR(50),
		currency VARCHAR(3) NOT NULL,
		provider VARCHAR(50) NOT NULL,
		amount INTEGER NOT NULL,
		payment_dt BIGINT NOT NULL,
		bank VARCHAR(50) NOT NULL,
		delivery_cost INTEGER NOT NULL,
		goods_total INTEGER NOT NULL,
		custom_fee INTEGER NOT NULL
	);`

	itemsTable := `
	CREATE TABLE IF NOT EXISTS items (
		id SERIAL PRIMARY KEY,
		order_uid VARCHAR(50) REFERENCES orders(order_uid) ON DELETE CASCADE,
		chrt_id INTEGER NOT NULL,
		track_number VARCHAR(50) NOT NULL,
		price INTEGER NOT NULL,
		rid VARCHAR(50) NOT NULL,
		name VARCHAR(100) NOT NULL,
		sale INTEGER NOT NULL,
		size VARCHAR(10) NOT NULL,
		total_price INTEGER NOT NULL,
		nm_id INTEGER NOT NULL,
		brand VARCHAR(100) NOT NULL,
		status INTEGER NOT NULL
	);`

	tables := []string{ordersTable, deliveryTable, paymentTable, itemsTable}

	for _, table := range tables {
		if _, err := db.Exec(table); err != nil {
			return fmt.Errorf("failed to create table %s: %v", table, err)
		}
	}

	return nil
}

func (r *PostgresRepository) GetOrder(ctx context.Context, orderUID string) (*models.Order, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	order := &models.Order{}
	row := tx.QueryRowContext(ctx, `
		SELECT order_uid, track_number, entry, locale, internal_signature, 
		customer_id, delivery_service, shardkey, sm_id, date_created, oof_shard
		FROM orders WHERE order_uid = $1`, orderUID)

	err = row.Scan(&order.OrderUID, &order.TrackNumber, &order.Entry, &order.Locale, &order.InternalSignature, &order.CustomerID,
		&order.DeliveryService, &order.Shardkey, &order.SmID, &order.DateCreated, &order.OofShard)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, service.ErrOrderNotFound
		}
	}

	delivery := &models.Delivery{}
	row = tx.QueryRowContext(ctx, `
		SELECT name, phone, zip, city, address, region, email
		FROM delivery WHERE order_uid = $1`, orderUID)

	err = row.Scan(&delivery.Name, &delivery.Phone, &delivery.Zip, &delivery.City, &delivery.Address, &delivery.Region,
		&delivery.Email)

	if err != nil {
		return nil, err
	}
	order.Delivery = *delivery

	payment := &models.Payment{}
	row = tx.QueryRowContext(ctx, `
		SELECT transaction, request_id, currency, provider, amount, payment_dt, 
		bank, delivery_cost, goods_total, custom_fee FROM payment WHERE transaction = $1`, orderUID)

	err = row.Scan(&payment.Transaction, &payment.RequestID, &payment.Currency, &payment.Provider, &payment.Amount,
		&payment.PaymentDt, &payment.Bank, &payment.DeliveryCost, &payment.GoodsTotal, &payment.CustomFee)
	if err != nil {
		return nil, err
	}
	order.Payment = *payment

	rows, err := tx.QueryContext(ctx, `
		SELECT chrt_id, track_number, price, rid, name, sale, size, total_price, nm_id, brand, status
		FROM items WHERE order_uid = $1`, orderUID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []models.Item
	for rows.Next() {
		var item models.Item
		err = rows.Scan(&item.ChrtID, &item.TrackNumber, &item.Price, &item.Rid,
			&item.Name, &item.Sale, &item.Size, &item.TotalPrice, &item.NmID,
			&item.Brand, &item.Status)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	order.Items = items

	if err = tx.Commit(); err != nil {
		return nil, err
	}

	return order, nil
}

func (r *PostgresRepository) SaveOrder(ctx context.Context, order *models.Order) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `
		INSERT INTO orders (order_uid, track_number, entry, locale, internal_signature, 
			customer_id, delivery_service, shardkey, sm_id, date_created, oof_shard)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (order_uid) DO UPDATE SET
			track_number = EXCLUDED.track_number,
			entry = EXCLUDED.entry,
			locale = EXCLUDED.locale,
			internal_signature = EXCLUDED.internal_signature,
			customer_id = EXCLUDED.customer_id,
			delivery_service = EXCLUDED.delivery_service,
			shardkey = EXCLUDED.shardkey,
			sm_id = EXCLUDED.sm_id,
			date_created = EXCLUDED.date_created,
			oof_shard = EXCLUDED.oof_shard`,
		order.OrderUID, order.TrackNumber, order.Entry, order.Locale,
		order.InternalSignature, order.CustomerID, order.DeliveryService,
		order.Shardkey, order.SmID, order.DateCreated, order.OofShard)

	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO delivery (order_uid, name, phone, zip, city, address, region, email)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (order_uid) DO UPDATE SET
			name = EXCLUDED.name,
			phone = EXCLUDED.phone,
			zip = EXCLUDED.zip,
			city = EXCLUDED.city,
			address = EXCLUDED.address,
			region = EXCLUDED.region,
			email = EXCLUDED.email`,
		order.OrderUID, order.Delivery.Name, order.Delivery.Phone, order.Delivery.Zip,
		order.Delivery.City, order.Delivery.Address, order.Delivery.Region, order.Delivery.Email)

	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO payment (transaction, request_id, currency, provider, amount, 
			payment_dt, bank, delivery_cost, goods_total, custom_fee)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (transaction) DO UPDATE SET
			request_id = EXCLUDED.request_id,
			currency = EXCLUDED.currency,
			provider = EXCLUDED.provider,
			amount = EXCLUDED.amount,
			payment_dt = EXCLUDED.payment_dt,
			bank = EXCLUDED.bank,
			delivery_cost = EXCLUDED.delivery_cost,
			goods_total = EXCLUDED.goods_total,
			custom_fee = EXCLUDED.custom_fee`,
		order.Payment.Transaction, order.Payment.RequestID, order.Payment.Currency,
		order.Payment.Provider, order.Payment.Amount, order.Payment.PaymentDt,
		order.Payment.Bank, order.Payment.DeliveryCost, order.Payment.GoodsTotal,
		order.Payment.CustomFee)

	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, "DELETE FROM items WHERE order_uid = $1", order.OrderUID)

	if err != nil {
		return err
	}

	for _, item := range order.Items {
		_, err := tx.ExecContext(ctx, `
			INSERT INTO items (order_uid, chrt_id, track_number, price, rid, name, 
				sale, size, total_price, nm_id, brand, status)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
			order.OrderUID, item.ChrtID, item.TrackNumber, item.Price, item.Rid,
			item.Name, item.Sale, item.Size, item.TotalPrice, item.NmID, item.Brand,
			item.Status)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *PostgresRepository) GetAllOrders(ctx context.Context) ([]*models.Order, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT order_uid FROM orders")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []*models.Order
	for rows.Next() {
		var orderUID string
		if err = rows.Scan(&orderUID); err != nil {
			return nil, err
		}

		order, err := r.GetOrder(ctx, orderUID)
		if err != nil {
			log.Printf("failed to get order %s: %v", orderUID, err)
			continue
		}
		orders = append(orders, order)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}
	return orders, nil
}

func (r *PostgresRepository) Close() error {
	return r.db.Close()
}
