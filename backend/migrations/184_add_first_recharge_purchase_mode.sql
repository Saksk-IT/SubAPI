-- 首充活动支持站内支付与商品链接两种购买入口。
ALTER TABLE first_recharge_activity_config
    ADD COLUMN IF NOT EXISTS purchase_mode VARCHAR(32) NOT NULL DEFAULT 'internal_payment',
    ADD COLUMN IF NOT EXISTS product_url TEXT NOT NULL DEFAULT '';

DO $$
BEGIN
    ALTER TABLE first_recharge_activity_config
        ADD CONSTRAINT first_recharge_activity_config_purchase_mode_chk CHECK (
            purchase_mode IN ('internal_payment', 'product_link')
        );
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

COMMENT ON COLUMN first_recharge_activity_config.purchase_mode IS '首充购买入口：internal_payment 或 product_link';
COMMENT ON COLUMN first_recharge_activity_config.product_url IS '商品链接入口地址，仅 product_link 模式使用';
