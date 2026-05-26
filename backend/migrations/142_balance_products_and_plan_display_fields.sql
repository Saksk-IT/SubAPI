CREATE TABLE IF NOT EXISTS balance_products (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    price DECIMAL(20,2) NOT NULL,
    amount DECIMAL(20,2) NOT NULL,
    original_price DECIMAL(20,2),
    tags TEXT NOT NULL DEFAULT '',
    features TEXT NOT NULL DEFAULT '',
    product_name VARCHAR(100) NOT NULL DEFAULT '',
    for_sale BOOLEAN NOT NULL DEFAULT TRUE,
    sort_order INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_balance_products_for_sale ON balance_products(for_sale);
CREATE INDEX IF NOT EXISTS idx_balance_products_sort_order ON balance_products(sort_order);

ALTER TABLE subscription_plans
    ADD COLUMN IF NOT EXISTS tags TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS total_quota DECIMAL(20,2),
    ADD COLUMN IF NOT EXISTS daily_quota DECIMAL(20,2),
    ADD COLUMN IF NOT EXISTS display_notes TEXT NOT NULL DEFAULT '';

ALTER TABLE payment_orders
    ADD COLUMN IF NOT EXISTS balance_product_id BIGINT;

CREATE INDEX IF NOT EXISTS idx_payment_orders_balance_product_id ON payment_orders(balance_product_id);
