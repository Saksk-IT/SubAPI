package admin

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestParseCodexSessionImportEntriesSupportsRawTokenJSONAndArray(t *testing.T) {
	token1 := "raw-access-token-1"
	token2 := buildCodexImportTestJWT(t, time.Now().Add(time.Hour), map[string]any{
		"email": "json@example.com",
	})
	token3 := "raw-access-token-3"

	req := CodexSessionImportRequest{
		Content: fmt.Sprintf("%s\n{\"accessToken\":%q}\n[%q]", token1, token2, token3),
	}

	entries, err := parseCodexSessionImportEntries(req)
	if err != nil {
		t.Fatalf("parseCodexSessionImportEntries error = %v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("len(entries) = %d, want 3", len(entries))
	}

	first, err := normalizeCodexImportEntry(entries[0])
	if err != nil {
		t.Fatalf("normalize raw token error = %v", err)
	}
	if first.Credentials["access_token"] != token1 {
		t.Fatalf("raw token access_token = %v, want %s", first.Credentials["access_token"], token1)
	}

	second, err := normalizeCodexImportEntry(entries[1])
	if err != nil {
		t.Fatalf("normalize json token error = %v", err)
	}
	if second.Email != "json@example.com" {
		t.Fatalf("email = %q, want json@example.com", second.Email)
	}

	third, err := normalizeCodexImportEntry(entries[2])
	if err != nil {
		t.Fatalf("normalize array token error = %v", err)
	}
	if third.Credentials["access_token"] != token3 {
		t.Fatalf("array token access_token = %v, want %s", third.Credentials["access_token"], token3)
	}
}

func TestParseCodexSessionImportEntriesFallsBackToLineModeForMixedJSONAndToken(t *testing.T) {
	req := CodexSessionImportRequest{
		Content: "{\"accessToken\":\"json-line-token\"}\nraw-line-token",
	}

	entries, err := parseCodexSessionImportEntries(req)
	if err != nil {
		t.Fatalf("parseCodexSessionImportEntries error = %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("len(entries) = %d, want 2", len(entries))
	}

	first, err := normalizeCodexImportEntry(entries[0])
	if err != nil {
		t.Fatalf("normalize json line error = %v", err)
	}
	if first.Credentials["access_token"] != "json-line-token" {
		t.Fatalf("json line access_token = %v, want json-line-token", first.Credentials["access_token"])
	}

	second, err := normalizeCodexImportEntry(entries[1])
	if err != nil {
		t.Fatalf("normalize raw line error = %v", err)
	}
	if second.Credentials["access_token"] != "raw-line-token" {
		t.Fatalf("raw line access_token = %v, want raw-line-token", second.Credentials["access_token"])
	}
}

func TestNormalizeCodexSessionJSONExtractsCredentialsAndIgnoresSessionToken(t *testing.T) {
	accessToken := buildCodexImportTestJWT(t, time.Now().Add(time.Hour), map[string]any{
		"email": "claim@example.com",
		"https://api.openai.com/auth": map[string]any{
			"chatgpt_account_id": "acct-from-claim",
			"chatgpt_user_id":    "user-from-claim",
			"chatgpt_plan_type":  "plus",
			"poid":               "org-from-claim",
		},
	})
	raw := map[string]any{
		"user": map[string]any{
			"id":    "user-from-json",
			"name":  "Sup OO",
			"email": "json@example.com",
			"image": "https://example.com/avatar.png",
		},
		"account": map[string]any{
			"id":       "acct-from-json",
			"planType": "free",
		},
		"accessToken":  accessToken,
		"sessionToken": "secret-session-token",
		"expires":      "2026-08-05T13:40:42.836Z",
	}

	item, err := normalizeCodexImportEntry(codexImportEntry{Index: 1, Value: raw})
	if err != nil {
		t.Fatalf("normalizeCodexImportEntry error = %v", err)
	}
	if item.Credentials["access_token"] != accessToken {
		t.Fatalf("access_token not stored")
	}
	if item.Credentials["email"] != "json@example.com" {
		t.Fatalf("email = %v, want json@example.com", item.Credentials["email"])
	}
	if item.Credentials["chatgpt_account_id"] != "acct-from-json" {
		t.Fatalf("chatgpt_account_id = %v, want acct-from-json", item.Credentials["chatgpt_account_id"])
	}
	if item.Credentials["chatgpt_user_id"] != "user-from-json" {
		t.Fatalf("chatgpt_user_id = %v, want user-from-json", item.Credentials["chatgpt_user_id"])
	}
	if item.Credentials["plan_type"] != "free" {
		t.Fatalf("plan_type = %v, want free", item.Credentials["plan_type"])
	}
	if _, ok := item.Credentials["session_token"]; ok {
		t.Fatalf("session_token should not be written to credentials")
	}
	if item.Extra["session_token_present"] != true {
		t.Fatalf("session_token_present = %v, want true", item.Extra["session_token_present"])
	}
	if item.Extra["session_expires_at"] != "2026-08-05T13:40:42Z" {
		t.Fatalf("session_expires_at = %v", item.Extra["session_expires_at"])
	}
	if item.TokenExpiresAt == nil {
		t.Fatalf("TokenExpiresAt should be parsed from accessToken")
	}
}

func TestMergeCodexImportCredentialsClearsStaleRefreshFieldsWhenIncomingHasNoRefreshToken(t *testing.T) {
	existing := map[string]any{
		"access_token":       "old-access-token",
		"refresh_token":      "old-refresh-token",
		"client_id":          "old-client-id",
		"id_token":           "old-id-token",
		"model_mapping":      map[string]any{"from": "existing"},
		"chatgpt_account_id": "acct-old",
		"unrelated_existing": "keep",
	}
	incoming := map[string]any{
		"access_token":       "new-access-token",
		"expires_at":         "2026-08-05T13:40:42Z",
		"chatgpt_account_id": "acct-new",
	}
	item := &codexImportAccount{
		AccessToken: "new-access-token",
	}

	merged := mergeCodexImportCredentials(existing, incoming, item)

	if merged["access_token"] != "new-access-token" {
		t.Fatalf("access_token = %v, want new-access-token", merged["access_token"])
	}
	if merged["chatgpt_account_id"] != "acct-new" {
		t.Fatalf("chatgpt_account_id = %v, want acct-new", merged["chatgpt_account_id"])
	}
	if _, ok := merged["refresh_token"]; ok {
		t.Fatalf("refresh_token should be cleared")
	}
	if _, ok := merged["client_id"]; ok {
		t.Fatalf("client_id should be cleared")
	}
	if _, ok := merged["id_token"]; ok {
		t.Fatalf("id_token should be cleared")
	}
	if merged["unrelated_existing"] != "keep" {
		t.Fatalf("unrelated_existing = %v, want keep", merged["unrelated_existing"])
	}
	if _, ok := merged["model_mapping"]; !ok {
		t.Fatalf("model_mapping should be preserved")
	}
}

func TestMergeCodexImportCredentialsKeepsRefreshFieldsWhenIncomingHasRefreshToken(t *testing.T) {
	existing := map[string]any{
		"refresh_token": "old-refresh-token",
		"client_id":     "old-client-id",
		"id_token":      "old-id-token",
	}
	incoming := map[string]any{
		"access_token":  "new-access-token",
		"refresh_token": "new-refresh-token",
		"client_id":     "new-client-id",
		"id_token":      "new-id-token",
	}
	item := &codexImportAccount{
		AccessToken:  "new-access-token",
		RefreshToken: "new-refresh-token",
		IDToken:      "new-id-token",
	}

	merged := mergeCodexImportCredentials(existing, incoming, item)

	if merged["refresh_token"] != "new-refresh-token" {
		t.Fatalf("refresh_token = %v, want new-refresh-token", merged["refresh_token"])
	}
	if merged["client_id"] != "new-client-id" {
		t.Fatalf("client_id = %v, want new-client-id", merged["client_id"])
	}
	if merged["id_token"] != "new-id-token" {
		t.Fatalf("id_token = %v, want new-id-token", merged["id_token"])
	}
}

func TestNormalizeCodexImportRejectsExpiredAccessToken(t *testing.T) {
	expiredToken := buildCodexImportTestJWT(t, time.Now().Add(-time.Hour), map[string]any{})

	_, err := normalizeCodexImportEntry(codexImportEntry{Index: 1, Value: expiredToken})
	if err == nil {
		t.Fatal("normalizeCodexImportEntry error = nil, want expired token error")
	}
	if !strings.Contains(err.Error(), "已过期") {
		t.Fatalf("error = %v, want expired token message", err)
	}
}

func TestResolveCodexImportExpiryForNoRefreshTokenUsesTokenExpiry(t *testing.T) {
	tokenExpiresAt := time.Now().Add(time.Hour).UTC()
	item := &codexImportAccount{
		AccessToken:    "access-token",
		Credentials:    map[string]any{"access_token": "access-token"},
		TokenExpiresAt: &tokenExpiresAt,
		WarningTexts:   []string{},
	}
	disabled := false
	req := CodexSessionImportRequest{AutoPauseOnExpired: &disabled}

	accountExpiresAt, credentialExpiresAt, autoPause, warnings, err := resolveCodexImportExpiry(req, item)
	if err != nil {
		t.Fatalf("resolveCodexImportExpiry error = %v", err)
	}
	if accountExpiresAt == nil || *accountExpiresAt != tokenExpiresAt.Unix() {
		t.Fatalf("account expires_at = %v, want %d", accountExpiresAt, tokenExpiresAt.Unix())
	}
	if credentialExpiresAt == nil || credentialExpiresAt.Unix() != tokenExpiresAt.Unix() {
		t.Fatalf("credential expires_at = %v, want %s", credentialExpiresAt, tokenExpiresAt)
	}
	if autoPause == nil || !*autoPause {
		t.Fatalf("autoPause = %v, want true", autoPause)
	}
	if len(warnings) == 0 {
		t.Fatalf("warnings should not be empty")
	}
}

func TestResolveCodexImportExpiryForNoRefreshTokenRequiresExpiry(t *testing.T) {
	item := &codexImportAccount{
		AccessToken:  "opaque-access-token",
		Credentials:  map[string]any{"access_token": "opaque-access-token"},
		WarningTexts: []string{},
	}

	_, _, _, _, err := resolveCodexImportExpiry(CodexSessionImportRequest{}, item)
	if err == nil {
		t.Fatal("resolveCodexImportExpiry error = nil, want missing expiry error")
	}
	if !strings.Contains(err.Error(), "无法解析 accessToken 过期时间") {
		t.Fatalf("error = %v, want missing expiry message", err)
	}
}

func TestResolveCodexImportExpiryForNoRefreshTokenUsesEarlierRequestExpiry(t *testing.T) {
	tokenExpiresAt := time.Now().Add(2 * time.Hour).UTC()
	requestExpiresAt := time.Now().Add(time.Hour).UTC()
	item := &codexImportAccount{
		AccessToken:    "access-token",
		Credentials:    map[string]any{"access_token": "access-token"},
		TokenExpiresAt: &tokenExpiresAt,
		WarningTexts:   []string{},
	}
	reqUnix := requestExpiresAt.Unix()
	req := CodexSessionImportRequest{ExpiresAt: &reqUnix}

	accountExpiresAt, credentialExpiresAt, _, _, err := resolveCodexImportExpiry(req, item)
	if err != nil {
		t.Fatalf("resolveCodexImportExpiry error = %v", err)
	}
	if accountExpiresAt == nil || *accountExpiresAt != requestExpiresAt.Unix() {
		t.Fatalf("account expires_at = %v, want %d", accountExpiresAt, requestExpiresAt.Unix())
	}
	if credentialExpiresAt == nil || credentialExpiresAt.Unix() != requestExpiresAt.Unix() {
		t.Fatalf("credential expires_at = %v, want %s", credentialExpiresAt, requestExpiresAt)
	}
}

func TestCodexIdentityKeysPreferStrongIdentifiers(t *testing.T) {
	keys := buildCodexIdentityKeys("acct-1", "acct-user-1", "user-1", "org-1", "same@example.com", "token", "refresh-token")
	for _, key := range keys {
		if strings.HasPrefix(key, "account:") || strings.HasPrefix(key, "user:") || strings.HasPrefix(key, "email:") {
			t.Fatalf("identity should avoid broad user/email keys: %v", keys)
		}
	}

	keys = buildCodexIdentityKeys("", "", "", "org-1", "same@example.com", "token", "")
	hasEmailOrg := false
	for _, key := range keys {
		if key == "email_org:same@example.com:org-1" {
			hasEmailOrg = true
		}
	}
	if !hasEmailOrg {
		t.Fatalf("weak identity with organization should include email_org fallback: %v", keys)
	}
}

func TestCodexIdentityKeysDoNotTreatSharedUserAsDuplicateAcrossOrganizations(t *testing.T) {
	first := buildCodexIdentityKeys("", "", "user-1", "org-1", "same@example.com", "token-1", "")
	second := buildCodexIdentityKeys("", "", "user-1", "org-2", "same@example.com", "token-2", "")

	seen := map[string]int{}
	markCodexIdentitySeen(seen, first, 1)
	if duplicateIndex, ok := firstSeenCodexIdentity(seen, second); ok {
		t.Fatalf("different organization should not duplicate item %d: first=%v second=%v", duplicateIndex, first, second)
	}
}

func TestNormalizeCodexImportEntriesDoNotDuplicateDifferentAccountUsersInSharedWorkspace(t *testing.T) {
	firstToken := buildCodexImportTestJWT(t, time.Now().Add(time.Hour), map[string]any{
		"sub":   "shared-sub",
		"email": "first@example.com",
		"https://api.openai.com/auth": map[string]any{
			"chatgpt_account_id":      "shared-workspace",
			"chatgpt_account_user_id": "account-user-1",
			"chatgpt_user_id":         "user-1",
			"poid":                    "shared-org",
		},
	})
	secondToken := buildCodexImportTestJWT(t, time.Now().Add(time.Hour), map[string]any{
		"sub":   "shared-sub",
		"email": "second@example.com",
		"https://api.openai.com/auth": map[string]any{
			"chatgpt_account_id":      "shared-workspace",
			"chatgpt_account_user_id": "account-user-2",
			"chatgpt_user_id":         "user-2",
			"poid":                    "shared-org",
		},
	})

	first, err := normalizeCodexImportEntry(codexImportEntry{Index: 1, Value: map[string]any{"accessToken": firstToken}})
	if err != nil {
		t.Fatalf("normalize first error = %v", err)
	}
	second, err := normalizeCodexImportEntry(codexImportEntry{Index: 2, Value: map[string]any{"accessToken": secondToken}})
	if err != nil {
		t.Fatalf("normalize second error = %v", err)
	}

	seen := map[string]int{}
	markCodexIdentitySeen(seen, first.IdentityKeys, 1)
	if duplicateIndex, ok := firstSeenCodexIdentity(seen, second.IdentityKeys); ok {
		t.Fatalf("different chatgpt account users should not duplicate item %d: first=%v second=%v", duplicateIndex, first.IdentityKeys, second.IdentityKeys)
	}
}

func TestNormalizeCodexSessionJSONUsesAccountUserIDFromAccessToken(t *testing.T) {
	token := buildCodexImportTestJWT(t, time.Now().Add(time.Hour), map[string]any{
		"sub": "auth0|provider-user",
		"https://api.openai.com/auth": map[string]any{
			"chatgpt_account_id":      "workspace-1",
			"chatgpt_account_user_id": "user-1__workspace-1",
			"chatgpt_user_id":         "user-1",
			"poid":                    "org-1",
		},
		"https://api.openai.com/profile": map[string]any{
			"email": "profile@example.com",
		},
	})
	raw := map[string]any{
		"user": map[string]any{
			"id":    "user-1",
			"email": "json@example.com",
		},
		"account": map[string]any{
			"id":             "workspace-1",
			"organizationId": "org-1",
			"planType":       "team",
		},
		"accessToken": token,
	}

	item, err := normalizeCodexImportEntry(codexImportEntry{Index: 1, Value: raw})
	if err != nil {
		t.Fatalf("normalizeCodexImportEntry error = %v", err)
	}
	if item.AccountUserID != "user-1__workspace-1" {
		t.Fatalf("AccountUserID = %q, want user-1__workspace-1", item.AccountUserID)
	}
	if item.Credentials["chatgpt_account_user_id"] != "user-1__workspace-1" {
		t.Fatalf("credential chatgpt_account_user_id = %v", item.Credentials["chatgpt_account_user_id"])
	}
	hasAccountUserKey := false
	for _, key := range item.IdentityKeys {
		if key == "account_user:workspace-1:user-1__workspace-1" {
			hasAccountUserKey = true
		}
	}
	if !hasAccountUserKey {
		t.Fatalf("identity keys should include account_user key: %v", item.IdentityKeys)
	}
}

func buildCodexImportTestJWT(t *testing.T, exp time.Time, extraClaims map[string]any) string {
	t.Helper()
	header := map[string]any{
		"alg": "none",
		"typ": "JWT",
	}
	claims := map[string]any{
		"sub": "user-from-sub",
		"exp": exp.Unix(),
		"iat": time.Now().Unix(),
	}
	for k, v := range extraClaims {
		claims[k] = v
	}
	headerBytes, err := json.Marshal(header)
	if err != nil {
		t.Fatalf("marshal header: %v", err)
	}
	claimBytes, err := json.Marshal(claims)
	if err != nil {
		t.Fatalf("marshal claims: %v", err)
	}
	return base64.RawURLEncoding.EncodeToString(headerBytes) + "." + base64.RawURLEncoding.EncodeToString(claimBytes) + "."
}
