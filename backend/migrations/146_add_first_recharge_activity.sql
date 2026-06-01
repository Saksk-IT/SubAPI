-- 首充活动：配置、档位、指定用户与用户状态。
CREATE TABLE IF NOT EXISTS first_recharge_activity_config (
    id SMALLINT PRIMARY KEY DEFAULT 1,
    enabled BOOLEAN NOT NULL DEFAULT FALSE,
    eligibility_scope VARCHAR(32) NOT NULL DEFAULT 'new_users_after_enabled',
    eligible_since TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT first_recharge_activity_config_singleton CHECK (id = 1),
    CONSTRAINT first_recharge_activity_config_scope_chk CHECK (
        eligibility_scope IN ('new_users_after_enabled', 'all_users', 'specified_users')
    )
);

INSERT INTO first_recharge_activity_config (id, enabled, eligibility_scope, eligible_since)
VALUES (1, FALSE, 'new_users_after_enabled', NULL)
ON CONFLICT (id) DO NOTHING;

CREATE TABLE IF NOT EXISTS first_recharge_offers (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    price DECIMAL(20,2) NOT NULL,
    amount DECIMAL(20,8) NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    sort_order INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT first_recharge_offers_price_chk CHECK (price > 0),
    CONSTRAINT first_recharge_offers_amount_chk CHECK (amount > 0)
);

CREATE INDEX IF NOT EXISTS idx_first_recharge_offers_enabled_sort
    ON first_recharge_offers(enabled, sort_order, id);

CREATE TABLE IF NOT EXISTS first_recharge_activity_users (
    user_id BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    created_by BIGINT REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS first_recharge_user_states (
    user_id BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    popup_dismissed_at TIMESTAMPTZ,
    completed_order_id BIGINT REFERENCES payment_orders(id) ON DELETE SET NULL,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE payment_orders
    ADD COLUMN IF NOT EXISTS activity_type VARCHAR(32) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS first_recharge_offer_id BIGINT REFERENCES first_recharge_offers(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_payment_orders_activity_type
    ON payment_orders(activity_type);
CREATE INDEX IF NOT EXISTS idx_payment_orders_first_recharge_offer_id
    ON payment_orders(first_recharge_offer_id);
CREATE INDEX IF NOT EXISTS idx_payment_orders_first_recharge_completed_user
    ON payment_orders(user_id, activity_type, status)
    WHERE activity_type = 'first_recharge';

COMMENT ON TABLE first_recharge_activity_config IS '首充活动全局配置';
COMMENT ON TABLE first_recharge_offers IS '首充活动价格与到账额度档位';
COMMENT ON TABLE first_recharge_activity_users IS '首充活动指定用户名单';
COMMENT ON TABLE first_recharge_user_states IS '首充活动用户弹窗关闭与完成状态';
COMMENT ON COLUMN payment_orders.activity_type IS '活动类型，first_recharge 表示首充活动订单';
COMMENT ON COLUMN payment_orders.first_recharge_offer_id IS '首充活动档位 ID';
