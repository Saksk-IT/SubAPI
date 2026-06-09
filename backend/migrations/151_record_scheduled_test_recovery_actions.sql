-- 151: Record automatic recovery actions on scheduled test results.

ALTER TABLE scheduled_test_results
    ADD COLUMN IF NOT EXISTS recovery_cleared_error BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS recovery_cleared_runtime_state BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS recovery_restored_scheduling BOOLEAN NOT NULL DEFAULT false;
