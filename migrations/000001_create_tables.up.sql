-- orders table
CREATE TABLE IF NOT EXISTS orders (
order_uid VARCHAR(50) PRIMARY KEY,
track_number VARCHAR(50) NOT NULL,
entry VARCHAR(10) NOT NULL,
locale VARCHAR(10) NOT NULL,
internal_signature VARCHAR(255),
customer_id VARCHAR(50) NOT NULL,
delivery_service VARCHAR(100) NOT NULL,
shardkey VARCHAR(50) NOT NULL,
sm_id INTEGER NOT NULL CHECK (sm_id >= 0),
date_created TIMESTAMPTZ NOT NULL,
oof_shard VARCHAR(50) NOT NULL,
raw_payload JSONB,
created_at TIMESTAMPTZ DEFAULT now() NOT NULL
);

-- delivery table
CREATE TABLE IF NOT EXISTS delivery (
order_uid VARCHAR(50) PRIMARY KEY REFERENCES orders(order_uid) ON DELETE CASCADE,
name VARCHAR(200) NOT NULL,
phone VARCHAR(50) NOT NULL,
zip VARCHAR(50) NOT NULL,
city VARCHAR(200) NOT NULL,
address VARCHAR(400) NOT NULL,
region VARCHAR(200) NOT NULL,
email VARCHAR(200) NOT NULL
);

-- payment table
CREATE TABLE IF NOT EXISTS payment (
id BIGSERIAL PRIMARY KEY,
order_uid VARCHAR(50) NOT NULL REFERENCES orders(order_uid) ON DELETE CASCADE,
transaction VARCHAR(200) NOT NULL UNIQUE,
request_id VARCHAR(200),
currency VARCHAR(10) NOT NULL,
provider VARCHAR(200) NOT NULL,
amount INTEGER NOT NULL,
payment_dt BIGINT NOT NULL,
bank VARCHAR(200) NOT NULL,
delivery_cost INTEGER NOT NULL,
goods_total INTEGER NOT NULL,
custom_fee INTEGER NOT NULL
);

-- items table
CREATE TABLE IF NOT EXISTS items (
id BIGSERIAL PRIMARY KEY,
order_uid VARCHAR(50) NOT NULL REFERENCES orders(order_uid) ON DELETE CASCADE,
chrt_id BIGINT NOT NULL,
track_number VARCHAR(50) NOT NULL,
price INTEGER NOT NULL,
rid VARCHAR(200) NOT NULL,
name TEXT NOT NULL,
sale INTEGER NOT NULL CHECK (sale >= 0),
size VARCHAR(50) NOT NULL,
total_price INTEGER NOT NULL,
nm_id BIGINT NOT NULL,
brand VARCHAR(200) NOT NULL,
status INTEGER NOT NULL CHECK (status >= 0),
UNIQUE (order_uid, chrt_id)
);

-- indexes
CREATE INDEX IF NOT EXISTS idx_orders_track_number ON orders(track_number);
CREATE INDEX IF NOT EXISTS idx_orders_customer_id ON orders(customer_id);
CREATE INDEX IF NOT EXISTS idx_orders_date_created ON orders(date_created);
CREATE INDEX IF NOT EXISTS idx_items_order_uid ON items(order_uid);
