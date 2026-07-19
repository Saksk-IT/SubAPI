package migrations

import (
	"strings"
	"testing"
)

func TestMigration184AddsFirstRechargePurchaseModes(t *testing.T) {
	content, err := FS.ReadFile("184_add_first_recharge_purchase_mode.sql")
	if err != nil {
		t.Fatalf("read migration 184: %v", err)
	}

	sql := string(content)
	for _, expected := range []string{
		"ADD COLUMN IF NOT EXISTS purchase_mode",
		"ADD COLUMN IF NOT EXISTS product_url",
		"purchase_mode IN ('internal_payment', 'product_link')",
	} {
		if !strings.Contains(sql, expected) {
			t.Fatalf("migration 184 is missing %q", expected)
		}
	}
}
