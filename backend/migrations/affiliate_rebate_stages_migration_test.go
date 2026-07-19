package migrations

import (
	"strings"
	"testing"
)

func TestMigration186AddsAtomicAffiliateRebateStages(t *testing.T) {
	content, err := FS.ReadFile("186_affiliate_rebate_stages.sql")
	if err != nil {
		t.Fatalf("read migration 186: %v", err)
	}

	sql := string(content)
	for _, expected := range []string{
		"ADD COLUMN IF NOT EXISTS rebate_stage",
		"PARTITION BY source_user_id",
		"idx_user_affiliate_ledger_first_rebate_uniq",
		"WHERE action = 'accrue' AND rebate_stage = 'first'",
	} {
		if !strings.Contains(sql, expected) {
			t.Fatalf("migration 186 is missing %q", expected)
		}
	}
}
