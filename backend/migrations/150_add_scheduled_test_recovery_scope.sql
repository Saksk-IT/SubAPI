-- 150: Add configurable recovery scopes to scheduled test plans.

ALTER TABLE scheduled_test_plans
    ADD COLUMN IF NOT EXISTS auto_recover_manual_stop BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS auto_recover_error_code_stop BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS auto_recover_runtime_state BOOLEAN NOT NULL DEFAULT false;

UPDATE scheduled_test_plans
SET auto_recover_error_code_stop = true,
    auto_recover_runtime_state = true
WHERE auto_recover = true
  AND auto_recover_manual_stop = false
  AND auto_recover_error_code_stop = false
  AND auto_recover_runtime_state = false;
