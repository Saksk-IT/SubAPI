ALTER TABLE balance_products
    ADD COLUMN IF NOT EXISTS purchase_limit INT NOT NULL DEFAULT 0;

