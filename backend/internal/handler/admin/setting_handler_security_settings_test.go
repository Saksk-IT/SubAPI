package admin

import (
	"encoding/json"
	"slices"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func TestSecuritySettingResolversPreserveOmittedValues(t *testing.T) {
	var req UpdateSettingsRequest
	if err := json.Unmarshal([]byte(`{}`), &req); err != nil {
		t.Fatalf("unmarshal empty settings request: %v", err)
	}

	if got := resolveSessionBindingEnabled(req.SessionBindingEnabled, true); !got {
		t.Fatal("omitted session_binding_enabled must preserve the previous true value")
	}
	if got := resolveAuditLogRetentionDays(req.AuditLogRetentionDays, 180); got != 180 {
		t.Fatalf("omitted audit_log_retention_days = %d, want 180", got)
	}
}

func TestSecuritySettingResolversHonorExplicitZeroValues(t *testing.T) {
	var req UpdateSettingsRequest
	if err := json.Unmarshal([]byte(`{"session_binding_enabled":false,"audit_log_retention_days":0}`), &req); err != nil {
		t.Fatalf("unmarshal explicit settings request: %v", err)
	}

	if got := resolveSessionBindingEnabled(req.SessionBindingEnabled, true); got {
		t.Fatal("explicit false session_binding_enabled must be applied")
	}
	if got := resolveAuditLogRetentionDays(req.AuditLogRetentionDays, 180); got != 0 {
		t.Fatalf("explicit zero audit_log_retention_days = %d, want 0", got)
	}
}

func TestDiffSettingsIncludesAuditSecuritySettings(t *testing.T) {
	before := &service.SystemSettings{
		SessionBindingEnabled: true,
		AuditLogRetentionDays: 180,
	}
	after := &service.SystemSettings{
		SessionBindingEnabled: false,
		AuditLogRetentionDays: 90,
	}

	changed := diffSettings(before, after, nil, nil, UpdateSettingsRequest{})
	for _, field := range []string{"session_binding_enabled", "audit_log_retention_days"} {
		if !slices.Contains(changed, field) {
			t.Fatalf("diffSettings missing %q: %v", field, changed)
		}
	}
}
