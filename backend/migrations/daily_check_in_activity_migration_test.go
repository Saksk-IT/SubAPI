package migrations

import (
	"strings"
	"testing"
)

func TestMigration185AddsAtomicDailyCheckInStorage(t *testing.T) {
	content, err := FS.ReadFile("185_add_daily_check_in_activity.sql")
	if err != nil {
		t.Fatalf("read migration 185: %v", err)
	}

	sql := string(content)
	for _, expected := range []string{
		"CREATE TABLE IF NOT EXISTS daily_check_in_activity_config",
		"CREATE TABLE IF NOT EXISTS daily_check_in_user_states",
		"CREATE TABLE IF NOT EXISTS daily_check_in_records",
		"UNIQUE (user_id, check_in_date)",
		"reward_amount DECIMAL(20,8) NOT NULL",
		"balance_after DECIMAL(20,8) NOT NULL",
	} {
		if !strings.Contains(sql, expected) {
			t.Fatalf("migration 185 is missing %q", expected)
		}
	}
}
