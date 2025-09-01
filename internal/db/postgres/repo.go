package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"
	"wb-tech-1task/internal/service"

	_ "github.com/lib/pq"

	migrate "github.com/golang-migrate/migrate/v4"
	migratepg "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	"wb-tech-1task/internal/models"
)

type PostgresRepository struct {
	db *sql.DB
}

func NewPostgresRepository(dsn string) (*PostgresRepository, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	db.SetMaxOpenConns(50)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)

	pingDeadline := time.Now().Add(30 * time.Second)
	for {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		err = db.PingContext(ctx)
		cancel()
		if err == nil {
			break
		}
		if time.Now().After(pingDeadline) {
			_ = db.Close()
			return nil, fmt.Errorf("ping postgres failed after retries: %w", err)
		}
		time.Sleep(1 * time.Second)
	}

	if err := runMigrations(dsn, db); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("run migrations: %w", err)
	}

	return &PostgresRepository{db: db}, nil
}

func runMigrations(dsn string, db *sql.DB) error {
	driver, err := migratepg.WithInstance(db, &migratepg.Config{})
	if err != nil {
		return fmt.Errorf("migrate: create driver: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance("file://migrations", "postgres", driver)
	if err != nil {
		return fmt.Errorf("migrate: new instance: %w", err)
	}
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("migrate up: %w", err)
	}
	return nil
}

func ctxWithTimeout(ctx context.Context, d time.Duration) (context.Context, context.CancelFunc) {
	if _, ok := ctx.Deadline(); ok {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, d)
}

func (r *PostgresRepository) Close() error {
	return r.db.Close()
}

func (r *PostgresRepository) GetOrder(ctx context.Context, orderUID string) (*models.Order, error) {
	qctx, cancel := ctxWithTimeout(ctx, 5*time.Second)
	defer cancel()

	tx, err := r.db.BeginTx(qctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	order := &models.Order{}
	var dateCreated time.Time

	row := tx.QueryRowContext(qctx, `
SELECT order_uid, track_number, entry, locale, internal_signature,
customer_id, delivery_service, shardkey, sm_id, date_created, oof_shard
FROM orders WHERE order_uid = $1`, orderUID)

	if err := row.Scan(&order.OrderUID, &order.TrackNumber, &order.Entry, &order.Locale, &order.InternalSignature,
		&order.CustomerID, &order.DeliveryService, &order.Shardkey, &order.SmID, &dateCreated, &order.OofShard); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, service.ErrOrderNotFound
		}
		return nil, err
	}
	order.DateCreated = dateCreated

	d := &models.Delivery{}
	row = tx.QueryRowContext(qctx, `
SELECT name, phone, zip, city, address, region, email
FROM delivery WHERE order_uid = $1`, orderUID)
	if err := row.Scan(&d.Name, &d.Phone, &d.Zip, &d.City, &d.Address, &d.Region, &d.Email); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("delivery not found for order %s: %w", orderUID, err)
		}
		return nil, err
	}
	order.Delivery = *d

	p := &models.Payment{}
	row = tx.QueryRowContext(qctx, `
SELECT transaction, request_id, currency, provider, amount, payment_dt, bank, delivery_cost, goods_total, custom_fee
FROM payment WHERE order_uid = $1`, orderUID)
	if err := row.Scan(&p.Transaction, &p.RequestID, &p.Currency, &p.Provider, &p.Amount, &p.PaymentDt, &p.Bank,
		&p.DeliveryCost, &p.GoodsTotal, &p.CustomFee); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("payment not found for order %s: %w", orderUID, err)
		}
		return nil, err
	}
	p.OrderUID = orderUID
	order.Payment = *p

	rows, err := tx.QueryContext(qctx, `
SELECT chrt_id, track_number, price, rid, name, sale, size, total_price, nm_id, brand, status
FROM items WHERE order_uid = $1 ORDER BY id`, orderUID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []models.Item
	for rows.Next() {
		var it models.Item
		if err := rows.Scan(&it.ChrtID, &it.TrackNumber, &it.Price, &it.Rid, &it.Name, &it.Sale,
			&it.Size, &it.TotalPrice, &it.NmID, &it.Brand, &it.Status); err != nil {
			return nil, err
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	order.Items = items

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return order, nil
}

func (r *PostgresRepository) SaveOrder(ctx context.Context, order *models.Order) error {
	qctx, cancel := ctxWithTimeout(ctx, 8*time.Second)
	defer cancel()

	tx, err := r.db.BeginTx(qctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(qctx, `
INSERT INTO orders (order_uid, track_number, entry, locale, internal_signature,
customer_id, delivery_service, shardkey, sm_id, date_created, oof_shard)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
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
oof_shard = EXCLUDED.oof_shard
`, order.OrderUID, order.TrackNumber, order.Entry, order.Locale, order.InternalSignature,
		order.CustomerID, order.DeliveryService, order.Shardkey, order.SmID, order.DateCreated, order.OofShard)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(qctx, `
INSERT INTO delivery (order_uid, name, phone, zip, city, address, region, email)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
ON CONFLICT (order_uid) DO UPDATE SET
name = EXCLUDED.name,
phone = EXCLUDED.phone,
zip = EXCLUDED.zip,
city = EXCLUDED.city,
address = EXCLUDED.address,
region = EXCLUDED.region,
email = EXCLUDED.email
`, order.OrderUID, order.Delivery.Name, order.Delivery.Phone, order.Delivery.Zip, order.Delivery.City,
		order.Delivery.Address, order.Delivery.Region, order.Delivery.Email)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(qctx, `
INSERT INTO payment (order_uid, transaction, request_id, currency, provider, amount, payment_dt, bank, delivery_cost, goods_total, custom_fee)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
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
order_uid = EXCLUDED.order_uid
`, order.OrderUID, order.Payment.Transaction, order.Payment.RequestID, order.Payment.Currency, order.Payment.Provider,
		order.Payment.Amount, order.Payment.PaymentDt, order.Payment.Bank, order.Payment.DeliveryCost, order.Payment.GoodsTotal,
		order.Payment.CustomFee)
	if err != nil {
		return err
	}

	if _, err := tx.ExecContext(qctx, "DELETE FROM items WHERE order_uid = $1", order.OrderUID); err != nil {
		return err
	}

	insertItemSQL := `
INSERT INTO items (order_uid, chrt_id, track_number, price, rid, name, sale, size, total_price, nm_id, brand, status)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
`
	for _, it := range order.Items {
		if _, err := tx.ExecContext(qctx, insertItemSQL, order.OrderUID, it.ChrtID, it.TrackNumber, it.Price,
			it.Rid, it.Name, it.Sale, it.Size, it.TotalPrice, it.NmID, it.Brand, it.Status); err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}

func (r *PostgresRepository) GetAllOrders(ctx context.Context) ([]*models.Order, error) {
	qctx, cancel := ctxWithTimeout(ctx, 15*time.Second)
	defer cancel()

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
	'date_created', to_char(o.date_created AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"'),
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
	rows, err := r.db.QueryContext(qctx, query)
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
