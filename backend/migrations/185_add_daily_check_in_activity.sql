-- 每日签到活动：全局配置、用户查看状态与按自然日唯一的签到记录。
CREATE TABLE IF NOT EXISTS daily_check_in_activity_config (
    id SMALLINT PRIMARY KEY DEFAULT 1,
    enabled BOOLEAN NOT NULL DEFAULT FALSE,
    reward_amount DECIMAL(20,8) NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT daily_check_in_activity_config_singleton CHECK (id = 1),
    CONSTRAINT daily_check_in_activity_reward_chk CHECK (reward_amount > 0)
);

INSERT INTO daily_check_in_activity_config (id, enabled, reward_amount)
VALUES (1, FALSE, 1)
ON CONFLICT (id) DO NOTHING;

CREATE TABLE IF NOT EXISTS daily_check_in_user_states (
    user_id BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    viewed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS daily_check_in_records (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    check_in_date DATE NOT NULL,
    reward_amount DECIMAL(20,8) NOT NULL,
    balance_after DECIMAL(20,8) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT daily_check_in_records_reward_chk CHECK (reward_amount > 0),
    CONSTRAINT daily_check_in_records_user_date_uq UNIQUE (user_id, check_in_date)
);

CREATE INDEX IF NOT EXISTS idx_daily_check_in_records_user_created
    ON daily_check_in_records(user_id, created_at DESC, id DESC);
CREATE INDEX IF NOT EXISTS idx_daily_check_in_records_date
    ON daily_check_in_records(check_in_date);

COMMENT ON TABLE daily_check_in_activity_config IS '每日签到活动全局配置';
COMMENT ON TABLE daily_check_in_user_states IS '每日签到活动用户查看状态';
COMMENT ON TABLE daily_check_in_records IS '每日签到奖励发放审计记录';
COMMENT ON COLUMN daily_check_in_records.check_in_date IS 'Asia/Shanghai 时区下的签到自然日';
COMMENT ON COLUMN daily_check_in_records.balance_after IS '本次奖励发放后的用户余额快照';
