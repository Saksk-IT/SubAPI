-- Add custom display ordering for channel monitors.
ALTER TABLE channel_monitors
    ADD COLUMN IF NOT EXISTS sort_order INT NOT NULL DEFAULT 0;

-- Backfill existing monitors so the current ID-desc list order is preserved after
-- switching the default order to sort_order ASC, id ASC.
WITH ranked AS (
    SELECT id, ROW_NUMBER() OVER (ORDER BY id DESC) * 10 AS next_sort_order
    FROM channel_monitors
)
UPDATE channel_monitors cm
SET sort_order = ranked.next_sort_order
FROM ranked
WHERE cm.id = ranked.id
  AND cm.sort_order = 0;

CREATE INDEX IF NOT EXISTS idx_channel_monitors_sort_order
    ON channel_monitors (sort_order, id);

CREATE INDEX IF NOT EXISTS idx_channel_monitors_enabled_sort_order
    ON channel_monitors (enabled, sort_order, id);
