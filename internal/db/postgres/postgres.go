package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
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
		order_uid        VARCHAR(50) PRIMARY KEY,
		track_number     VARCHAR(50) NOT NULL,
		entry            VARCHAR(10) NOT NULL,
		locale           VARCHAR(2) NOT NULL,
		internal_signature     VARCHAR(100),
		customer_id      VARCHAR(50) NOT NULL,
		delivery_service VARCHAR(50) NOT NULL,
		shardkey         VARCHAR(10) NOT NULL,
		sm_id            INTEGER NOT NULL CHECK (sm_id >= 0),
		date_created     TIMESTAMPTZ NOT NULL,
		oof_shard        VARCHAR(10) NOT NULL,
		raw_payload      JSONB,
		created_at       TIMESTAMPTZ DEFAULT now() NOT NULL
	);`

	deliveryTable := `
	CREATE TABLE IF NOT EXISTS delivery (
		id         BIGSERIAL PRIMARY KEY,
		order_uid  VARCHAR(50) NOT NULL REFERENCES orders(order_uid) ON DELETE CASCADE,
		name       VARCHAR(100) NOT NULL,
		phone      VARCHAR(30) NOT NULL,
		zip        VARCHAR(20) NOT NULL,
		city       VARCHAR(100) NOT NULL,
		address    VARCHAR(200) NOT NULL,
		region     VARCHAR(100) NOT NULL,
		email      VARCHAR(200) NOT NULL,
		UNIQUE (order_uid)
	);`

	paymentTable := `
	CREATE TABLE IF NOT EXISTS payment (
		id            BIGSERIAL PRIMARY KEY,
		order_uid     VARCHAR(50) NOT NULL REFERENCES orders(order_uid) ON DELETE CASCADE,
		transaction   VARCHAR(100) NOT NULL UNIQUE,
		request_id    VARCHAR(100),
		currency      VARCHAR(3) NOT NULL CHECK (char_length(currency) = 3),
		provider      VARCHAR(100) NOT NULL,
		amount        INTEGER NOT NULL,
		payment_dt    BIGINT NOT NULL,
		bank          VARCHAR(100) NOT NULL,
		delivery_cost INTEGER NOT NULL,
		goods_total   INTEGER NOT NULL,
		custom_fee    INTEGER NOT NULL
	);`

	itemsTable := `
	CREATE TABLE IF NOT EXISTS items (
		id         BIGSERIAL PRIMARY KEY,
		order_uid  VARCHAR(50) NOT NULL REFERENCES orders(order_uid) ON DELETE CASCADE,
		chrt_id    BIGINT NOT NULL,
		track_number VARCHAR(50) NOT NULL,
		price      INTEGER NOT NULL,
		rid        VARCHAR(100) NOT NULL,
		name       TEXT NOT NULL,
		sale       INTEGER NOT NULL CHECK (sale >= 0),
		size       VARCHAR(20) NOT NULL,
		total_price INTEGER NOT NULL,
		nm_id      BIGINT NOT NULL,
		brand      VARCHAR(200) NOT NULL,
		status     INTEGER NOT NULL CHECK (status >= 0),
		UNIQUE (order_uid, chrt_id)
	);`

	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_orders_track_number ON orders(track_number);",
		"CREATE INDEX IF NOT EXISTS idx_orders_customer_id ON orders(customer_id);",
		"CREATE INDEX IF NOT EXISTS idx_orders_date_created ON orders(date_created);",
		"CREATE INDEX IF NOT EXISTS idx_items_order_uid ON items(order_uid);",
	}

	ddl := map[string]string{
		"orders":   ordersTable,
		"delivery": deliveryTable,
		"payment":  paymentTable,
		"items":    itemsTable,
	}

	for name, sqlStmt := range ddl {
		if _, err := db.Exec(sqlStmt); err != nil {
			return fmt.Errorf("failed to create %s table: %v", name, err)
		}
	}

	for _, idx := range indexes {
		if _, err := db.Exec(idx); err != nil {
			return fmt.Errorf("failed to create index: %v", err)
		}
	}

	return nil
}

func (r *PostgresRepository) GetOrder(ctx context.Context, orderUID string) (*models.Order, error) {
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
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
		return nil, err
	}

	delivery := &models.Delivery{}
	row = tx.QueryRowContext(ctx, `
		SELECT name, phone, zip, city, address, region, email
		FROM delivery WHERE order_uid = $1`, orderUID)

	err = row.Scan(&delivery.Name, &delivery.Phone, &delivery.Zip, &delivery.City, &delivery.Address, &delivery.Region,
		&delivery.Email)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("delivery not found for order %s: %w", orderUID, err)
		}
		return nil, err
	}
	order.Delivery = *delivery

	payment := &models.Payment{}
	row = tx.QueryRowContext(ctx, `
		SELECT transaction, request_id, currency, provider, amount, payment_dt, bank, delivery_cost, goods_total, custom_fee
		FROM payment WHERE order_uid = $1`, orderUID)

	err = row.Scan(&payment.Transaction, &payment.RequestID, &payment.Currency, &payment.Provider, &payment.Amount,
		&payment.PaymentDt, &payment.Bank, &payment.DeliveryCost, &payment.GoodsTotal, &payment.CustomFee)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("payment not found for order %s: %w", orderUID, err)
		}
		return nil, err
	}
	payment.OrderUID = orderUID
	order.Payment = *payment

	rows, err := tx.QueryContext(ctx, `
		SELECT chrt_id, track_number, price, rid, name, sale, size, total_price, nm_id, brand, status
		FROM items WHERE order_uid = $1 ORDER BY id`, orderUID)
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
		INSERT INTO payment (order_uid, transaction, request_id, currency, provider, amount, 
			payment_dt, bank, delivery_cost, goods_total, custom_fee)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (transaction) DO UPDATE SET
			request_id = EXCLUDED.request_id,
			currency = EXCLUDED.currency,
			provider = EXCLUDED.provider,
			amount = EXCLUDED.amount,
			payment_dt = EXCLUDED.payment_dt,
			bank = EXCLUDED.bank,
			delivery_cost = EXCLUDED.delivery_cost,
			goods_total = EXCLUDED.goods_total,
			custom_fee = EXCLUDED.custom_fee,
			order_uid = EXCLUDED.order_uid`,
		order.OrderUID, order.Payment.Transaction, order.Payment.RequestID, order.Payment.Currency,
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

	insertItemSQL := `
		INSERT INTO items (order_uid, chrt_id, track_number, price, rid, name, sale, size, total_price, nm_id, brand, status)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
	`
	for _, item := range order.Items {
		_, err := tx.ExecContext(ctx, insertItemSQL, order.OrderUID, item.ChrtID, item.TrackNumber, item.Price,
			item.Rid, item.Name, item.Sale, item.Size, item.TotalPrice, item.NmID, item.Brand, item.Status)
		if err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}

func (r *PostgresRepository) GetAllOrders(ctx context.Context) ([]*models.Order, error) {
	query := `
    SELECT json_build_object(
      'order_uid', o.order_uid,
      'track_number', o.track_number,
      'entry', o.entry,
      'locale', o.locale,
      'internal_signature', o.internal_signature,
      'customer_id', o.customer_id,
      'delivery_service', o.delivery_service,
      'shardkey', o.shardkey,
      'sm_id', o.sm_id,
      'date_created', o.date_created,
      'oof_shard', o.oof_shard,
      'delivery', json_build_object(
        'name', d.name, 
        'phone', d.phone, 
        'zip', d.zip, 
        'city', d.city, 
        'address', d.address, 
        'region', d.region, 
        'email', d.email
      ),
      'payment', json_build_object(
        'transaction', p.transaction, 
        'request_id', p.request_id, 
        'currency', p.currency, 
        'provider', p.provider, 
        'amount', p.amount, 
        'payment_dt', p.payment_dt, 
        'bank', p.bank, 
        'delivery_cost', p.delivery_cost, 
        'goods_total', p.goods_total, 
        'custom_fee', p.custom_fee
      ),
      'items', COALESCE(it.items, '[]'::json)
    ) AS order_json
    FROM orders o
    LEFT JOIN delivery d ON d.order_uid = o.order_uid
    LEFT JOIN payment p ON p.order_uid = o.order_uid
    LEFT JOIN (
      SELECT order_uid, json_agg(json_build_object(
        'chrt_id', chrt_id,
        'track_number', track_number,
        'price', price,
        'rid', rid,
        'name', name,
        'sale', sale,
        'size', size,
        'total_price', total_price,
        'nm_id', nm_id,
        'brand', brand,
        'status', status
      )) AS items
      FROM items
      GROUP BY order_uid
    ) it ON it.order_uid = o.order_uid
    ORDER BY o.date_created DESC;
    `

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []*models.Order
	for rows.Next() {
		var raw json.RawMessage
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}

		var order models.Order
		if err := json.Unmarshal(raw, &order); err != nil {
			return nil, fmt.Errorf("failed to unmarshal order json: %w", err)
		}

		order.Payment.OrderUID = order.OrderUID

		orders = append(orders, &order)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return orders, nil
}

func (r *PostgresRepository) Close() error {
	return r.db.Close()
}
