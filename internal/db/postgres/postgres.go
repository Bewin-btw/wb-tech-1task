package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"wb-tech-1task/internal/models"
)

type PostgresRepository struct {
	db *sql.DB
}

func NewPostgresRepository(connectionString string) (*PostgresRepository, error) {
	db, err := sql.Open("postgres", connectionString)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %v", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %v", err)
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
			return fmt.Errorf("failed to create table %v: %v", table, err)
		}
	}

	return nil
}

func (r *PostgresRepository) GetOrder(ctx context.Context, orderUID string) (*models.Order, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	d
}
