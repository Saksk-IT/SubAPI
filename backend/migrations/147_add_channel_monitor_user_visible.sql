-- Migration: 147_add_channel_monitor_user_visible
-- 控制单个渠道监控是否展示在普通用户的渠道状态页。
-- 默认 TRUE 保持升级后现有监控的用户可见行为不变。

ALTER TABLE channel_monitors
    ADD COLUMN IF NOT EXISTS user_visible BOOLEAN NOT NULL DEFAULT TRUE;

CREATE INDEX IF NOT EXISTS idx_channel_monitors_user_visible
    ON channel_monitors (user_visible);
