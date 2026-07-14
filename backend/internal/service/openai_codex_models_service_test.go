package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

type codexModelsTokenCacheStub struct {
	token string
}

func (s *codexModelsTokenCacheStub) GetAccessToken(context.Context, string) (string, error) {
	return s.token, nil
}

func (s *codexModelsTokenCacheStub) SetAccessToken(context.Context, string, string, time.Duration) error {
	return nil
}

func (s *codexModelsTokenCacheStub) DeleteAccessToken(context.Context, string) error {
	return nil
}

func (s *codexModelsTokenCacheStub) AcquireRefreshLock(context.Context, string, time.Duration) (bool, error) {
	return false, nil
}

func (s *codexModelsTokenCacheStub) ReleaseRefreshLock(context.Context, string) error {
	return nil
}

type codexModelsAccountRepoStub struct {
	AccountRepository
	accounts []Account
}

func (r *codexModelsAccountRepoStub) ListSchedulableByGroupIDAndPlatform(context.Context, int64, string) ([]Account, error) {
	return append([]Account(nil), r.accounts...), nil
}

func (r *codexModelsAccountRepoStub) GetByID(_ context.Context, id int64) (*Account, error) {
	for i := range r.accounts {
		if r.accounts[i].ID == id {
			account := r.accounts[i]
			return &account, nil
		}
	}
	return nil, ErrNoAvailableAccounts
}

func newSchedulableCodexModelsAccount(id int64, accountType string, token string, priority int) Account {
	credentials := map[string]any{}
	if accountType == AccountTypeOAuth && token != "" {
		credentials["access_token"] = token
	}
	if accountType == AccountTypeAPIKey {
		credentials["api_key"] = token
	}
	return Account{
		ID:          id,
		Platform:    PlatformOpenAI,
		Type:        accountType,
		Status:      StatusActive,
		Schedulable: true,
		Concurrency: 1,
		Priority:    priority,
		Credentials: credentials,
	}
}

func newCodexModelsTestAccount() *Account {
	return &Account{
		ID:       1,
		Platform: PlatformOpenAI,
		Type:     AccountTypeOAuth,
		Credentials: map[string]any{
			"access_token":       "test-access-token",
			"chatgpt_account_id": "acc-123",
		},
	}
}

func TestFetchCodexModelsManifestPassthrough(t *testing.T) {
	manifestBody := `{"models":[{"slug":"gpt-5.5","display_name":"GPT-5.5"}]}`

	var gotAuth, gotAccountID, gotOriginator, gotClientVersion string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotAccountID = r.Header.Get("chatgpt-account-id")
		gotOriginator = r.Header.Get("Originator")
		gotClientVersion = r.URL.Query().Get("client_version")
		w.Header().Set("ETag", `W/"abc123"`)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(manifestBody))
	}))
	defer server.Close()

	original := chatgptCodexModelsURL
	chatgptCodexModelsURL = server.URL
	defer func() { chatgptCodexModelsURL = original }()

	s := &OpenAIGatewayService{}
	manifest, err := s.FetchCodexModelsManifest(context.Background(), newCodexModelsTestAccount(), "0.137.0", "")
	if err != nil {
		t.Fatalf("FetchCodexModelsManifest returned error: %v", err)
	}

	if string(manifest.Body) != manifestBody {
		t.Errorf("body not passed through verbatim: got %q", manifest.Body)
	}
	if manifest.ETag != `W/"abc123"` {
		t.Errorf("etag not passed through: got %q", manifest.ETag)
	}
	if gotAuth != "Bearer test-access-token" {
		t.Errorf("authorization header: got %q", gotAuth)
	}
	if gotAccountID != "acc-123" {
		t.Errorf("chatgpt-account-id header: got %q", gotAccountID)
	}
	if gotOriginator != "codex_cli_rs" {
		t.Errorf("originator header: got %q", gotOriginator)
	}
	if gotClientVersion != "0.137.0" {
		t.Errorf("client_version query: got %q", gotClientVersion)
	}
}

func TestFetchCodexModelsManifestDefaultClientVersion(t *testing.T) {
	var gotClientVersion string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotClientVersion = r.URL.Query().Get("client_version")
		_, _ = w.Write([]byte(`{"models":[]}`))
	}))
	defer server.Close()

	original := chatgptCodexModelsURL
	chatgptCodexModelsURL = server.URL
	defer func() { chatgptCodexModelsURL = original }()

	s := &OpenAIGatewayService{}
	if _, err := s.FetchCodexModelsManifest(context.Background(), newCodexModelsTestAccount(), "", ""); err != nil {
		t.Fatalf("FetchCodexModelsManifest returned error: %v", err)
	}
	if gotClientVersion != openAICodexProbeVersion {
		t.Errorf("default client_version: got %q, want %q", gotClientVersion, openAICodexProbeVersion)
	}
}

func TestFetchCodexModelsManifestNotModified(t *testing.T) {
	var gotIfNoneMatch string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotIfNoneMatch = r.Header.Get("If-None-Match")
		w.Header().Set("ETag", `W/"abc123"`)
		w.WriteHeader(http.StatusNotModified)
	}))
	defer server.Close()

	original := chatgptCodexModelsURL
	chatgptCodexModelsURL = server.URL
	defer func() { chatgptCodexModelsURL = original }()

	s := &OpenAIGatewayService{}
	manifest, err := s.FetchCodexModelsManifest(context.Background(), newCodexModelsTestAccount(), "0.137.0", `W/"abc123"`)
	if err != nil {
		t.Fatalf("FetchCodexModelsManifest returned error: %v", err)
	}
	if !manifest.NotModified {
		t.Error("expected NotModified to be true")
	}
	if gotIfNoneMatch != `W/"abc123"` {
		t.Errorf("if-none-match header: got %q", gotIfNoneMatch)
	}
}

func TestFetchCodexModelsManifestUpstreamError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"detail":"boom"}`, http.StatusInternalServerError)
	}))
	defer server.Close()

	original := chatgptCodexModelsURL
	chatgptCodexModelsURL = server.URL
	defer func() { chatgptCodexModelsURL = original }()

	s := &OpenAIGatewayService{}
	if _, err := s.FetchCodexModelsManifest(context.Background(), newCodexModelsTestAccount(), "0.137.0", ""); err == nil {
		t.Fatal("expected error for upstream 500, got nil")
	}
}

func TestFetchCodexModelsManifestMissingToken(t *testing.T) {
	account := newCodexModelsTestAccount()
	delete(account.Credentials, "access_token")

	s := &OpenAIGatewayService{}
	if _, err := s.FetchCodexModelsManifest(context.Background(), account, "0.137.0", ""); err == nil {
		t.Fatal("expected error for missing access token, got nil")
	} else if got := infraerrors.Reason(err); got != "OPENAI_CODEX_MODELS_TOKEN_UNAVAILABLE" {
		t.Fatalf("error reason: got %q", got)
	}
}

func TestFetchCodexModelsManifestUsesTokenProviderCache(t *testing.T) {
	var gotAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		_, _ = w.Write([]byte(`{"models":[]}`))
	}))
	defer server.Close()

	original := chatgptCodexModelsURL
	chatgptCodexModelsURL = server.URL
	defer func() { chatgptCodexModelsURL = original }()

	account := newCodexModelsTestAccount()
	delete(account.Credentials, "access_token")
	provider := NewOpenAITokenProvider(nil, &codexModelsTokenCacheStub{token: "cached-access-token"}, nil)
	s := &OpenAIGatewayService{openAITokenProvider: provider}

	if _, err := s.FetchCodexModelsManifest(context.Background(), account, "0.144.3", ""); err != nil {
		t.Fatalf("FetchCodexModelsManifest returned error: %v", err)
	}
	if gotAuth != "Bearer cached-access-token" {
		t.Fatalf("authorization header: got %q", gotAuth)
	}
}

func TestFetchCodexModelsManifestRejectsAPIKeyAccount(t *testing.T) {
	account := newSchedulableCodexModelsAccount(1, AccountTypeAPIKey, "sk-test", 0)
	s := &OpenAIGatewayService{}

	_, err := s.FetchCodexModelsManifest(context.Background(), &account, "0.144.3", "")
	if err == nil {
		t.Fatal("expected API-key account to be rejected")
	}
	if got := infraerrors.Reason(err); got != "OPENAI_CODEX_MODELS_OAUTH_REQUIRED" {
		t.Fatalf("error reason: got %q", got)
	}
}

func TestFetchCodexModelsManifestForGroupSkipsAPIKeyAccount(t *testing.T) {
	accounts := []Account{
		newSchedulableCodexModelsAccount(1, AccountTypeAPIKey, "sk-test", 0),
		newSchedulableCodexModelsAccount(2, AccountTypeOAuth, "oauth-token", 1),
	}
	assertCodexModelsGroupSelectsAccount(t, accounts, 2)
}

func TestFetchCodexModelsManifestForGroupRetriesMissingOAuthToken(t *testing.T) {
	accounts := []Account{
		newSchedulableCodexModelsAccount(1, AccountTypeOAuth, "", 0),
		newSchedulableCodexModelsAccount(2, AccountTypeOAuth, "oauth-token", 1),
	}
	assertCodexModelsGroupSelectsAccount(t, accounts, 2)
}

func assertCodexModelsGroupSelectsAccount(t *testing.T, accounts []Account, wantAccountID int64) {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"models":[]}`))
	}))
	defer server.Close()

	original := chatgptCodexModelsURL
	chatgptCodexModelsURL = server.URL
	defer func() { chatgptCodexModelsURL = original }()

	repo := &codexModelsAccountRepoStub{accounts: accounts}
	s := &OpenAIGatewayService{accountRepo: repo}
	groupID := int64(10)
	manifest, account, err := s.FetchCodexModelsManifestForGroup(context.Background(), &groupID, "0.144.3", "")
	if err != nil {
		t.Fatalf("FetchCodexModelsManifestForGroup returned error: %v", err)
	}
	if manifest == nil {
		t.Fatal("expected manifest")
	}
	if account == nil || account.ID != wantAccountID {
		t.Fatalf("selected account: got %#v, want ID %d", account, wantAccountID)
	}
}
