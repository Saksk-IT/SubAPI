-- 区分首次付费与持续复购返利，并在数据库层保证每个被邀请用户只有一条首次返利流水。

ALTER TABLE user_affiliate_ledger
    ADD COLUMN IF NOT EXISTS rebate_stage VARCHAR(16) NULL;

WITH ranked AS (
    SELECT id,
           ROW_NUMBER() OVER (
               PARTITION BY source_user_id
               ORDER BY created_at ASC, id ASC
           ) AS sequence_number
    FROM user_affiliate_ledger
    WHERE action = 'accrue'
      AND source_user_id IS NOT NULL
)
UPDATE user_affiliate_ledger ledger
SET rebate_stage = CASE
        WHEN ranked.sequence_number = 1 THEN 'first'
        ELSE 'repeat'
    END
FROM ranked
WHERE ledger.id = ranked.id
  AND ledger.rebate_stage IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_user_affiliate_ledger_first_rebate_uniq
    ON user_affiliate_ledger(source_user_id)
    WHERE action = 'accrue' AND rebate_stage = 'first';

CREATE INDEX IF NOT EXISTS idx_user_affiliate_ledger_rebate_stage
    ON user_affiliate_ledger(rebate_stage, created_at)
    WHERE action = 'accrue' AND rebate_stage IS NOT NULL;

COMMENT ON COLUMN user_affiliate_ledger.rebate_stage IS '返利阶段：first=首次付费，repeat=持续复购；转余额流水为 NULL';
