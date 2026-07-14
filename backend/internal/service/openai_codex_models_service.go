package service

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/httpclient"
)

// chatgptCodexModelsURL is the ChatGPT Codex models manifest endpoint.
// Package-level variable so tests can point it at a stub server.
var chatgptCodexModelsURL = "https://chatgpt.com/backend-api/codex/models"

const codexModelsManifestBodyLimit int64 = 8 << 20

// CodexModelsManifest carries the raw upstream manifest payload plus caching
// metadata so handlers can pass both through to the client untouched.
type CodexModelsManifest struct {
	Body        []byte
	ETag        string
	NotModified bool
}

// FetchCodexModelsManifestForGroup selects a usable ChatGPT OAuth account and
// fetches the Codex models manifest. API-key accounts cannot authenticate the
// ChatGPT backend manifest endpoint, so they are skipped even when they are
// otherwise schedulable for the OpenAI platform. Credential failures are
// isolated to the affected account and retried on the next account in the
// group; transport/upstream failures are returned immediately.
func (s *OpenAIGatewayService) FetchCodexModelsManifestForGroup(ctx context.Context, groupID *int64, clientVersion, ifNoneMatch string) (*CodexModelsManifest, *Account, error) {
	excludedIDs := make(map[int64]struct{})
	var lastAccount *Account
	var lastCredentialErr error

	for {
		account, err := s.SelectAccountForModelWithExclusions(ctx, groupID, "", "", excludedIDs)
		if err != nil {
			if !errors.Is(err, ErrNoAvailableAccounts) {
				return nil, lastAccount, infraerrors.New(http.StatusInternalServerError, "OPENAI_CODEX_MODELS_ACCOUNT_SELECTION_FAILED", "failed to select an OpenAI account for Codex models").WithCause(err)
			}
			if lastCredentialErr != nil {
				return nil, lastAccount, lastCredentialErr
			}
			return nil, lastAccount, infraerrors.New(http.StatusServiceUnavailable, "OPENAI_CODEX_MODELS_NO_OAUTH_ACCOUNT", "No available OpenAI OAuth accounts for Codex models")
		}

		lastAccount = account
		excludedIDs[account.ID] = struct{}{}
		if !account.IsOpenAIOAuth() {
			continue
		}

		manifest, err := s.FetchCodexModelsManifest(ctx, account, clientVersion, ifNoneMatch)
		if err == nil {
			return manifest, account, nil
		}
		if isCodexModelsCredentialError(err) {
			lastCredentialErr = err
			continue
		}
		return nil, account, err
	}
}

func isCodexModelsCredentialError(err error) bool {
	switch infraerrors.Reason(err) {
	case "OPENAI_CODEX_MODELS_CREDENTIALS_FAILED",
		"OPENAI_CODEX_MODELS_OAUTH_REQUIRED",
		"OPENAI_CODEX_MODELS_TOKEN_UNAVAILABLE":
		return true
	default:
		return false
	}
}

// FetchCodexModelsManifest fetches the live Codex models manifest from the
// ChatGPT backend using the account's OAuth credentials.
//
// The response body is passed through verbatim: the manifest schema evolves
// with Codex client releases, and interpreting it here would force the gateway
// to chase upstream changes. Passing it through keeps the gateway
// schema-agnostic and always reflects the account's real entitlements.
func (s *OpenAIGatewayService) FetchCodexModelsManifest(ctx context.Context, account *Account, clientVersion, ifNoneMatch string) (*CodexModelsManifest, error) {
	if account == nil {
		return nil, infraerrors.New(http.StatusInternalServerError, "OPENAI_CODEX_MODELS_ACCOUNT_REQUIRED", "account is required")
	}
	credAccount, err := resolveCredentialAccount(ctx, s.accountRepo, account)
	if err != nil {
		return nil, infraerrors.Newf(http.StatusInternalServerError, "OPENAI_CODEX_MODELS_CREDENTIALS_FAILED", "resolve credential account: %v", err)
	}
	if !credAccount.IsOpenAIOAuth() {
		return nil, infraerrors.New(http.StatusServiceUnavailable, "OPENAI_CODEX_MODELS_OAUTH_REQUIRED", "Codex models manifest requires an OpenAI OAuth account")
	}
	accessToken, tokenType, err := s.GetAccessToken(ctx, credAccount)
	if err != nil {
		return nil, infraerrors.New(http.StatusBadGateway, "OPENAI_CODEX_MODELS_TOKEN_UNAVAILABLE", "account has no usable Codex backend access token").WithCause(err)
	}
	if tokenType != "oauth" || strings.TrimSpace(accessToken) == "" {
		return nil, infraerrors.New(http.StatusBadGateway, "OPENAI_CODEX_MODELS_TOKEN_UNAVAILABLE", "account has no usable Codex backend access token")
	}

	clientVersion = strings.TrimSpace(clientVersion)
	if clientVersion == "" {
		clientVersion = openAICodexProbeVersion
	}
	requestURL := chatgptCodexModelsURL + "?client_version=" + url.QueryEscape(clientVersion)

	reqCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, infraerrors.Newf(http.StatusInternalServerError, "OPENAI_CODEX_MODELS_REQUEST_FAILED", "create codex models request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Originator", "codex_cli_rs")
	req.Header.Set("Version", clientVersion)
	req.Header.Set("User-Agent", codexCLIUserAgent)
	if ifNoneMatch = strings.TrimSpace(ifNoneMatch); ifNoneMatch != "" {
		req.Header.Set("If-None-Match", ifNoneMatch)
	}
	setOpenAIChatGPTAccountHeaders(req.Header, credAccount)

	proxyURL := ""
	if account.ProxyID != nil && account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}
	client, err := httpclient.GetClient(httpclient.Options{
		ProxyURL:              proxyURL,
		Timeout:               15 * time.Second,
		ResponseHeaderTimeout: 10 * time.Second,
	})
	if err != nil {
		return nil, infraerrors.Newf(http.StatusInternalServerError, "OPENAI_CODEX_MODELS_PROXY_INVALID", "invalid proxy configuration: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, infraerrors.Newf(http.StatusBadGateway, "OPENAI_CODEX_MODELS_UPSTREAM_FAILED", "codex models manifest request failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotModified {
		return &CodexModelsManifest{ETag: resp.Header.Get("ETag"), NotModified: true}, nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		message := strings.TrimSpace(string(body))
		if message == "" {
			message = resp.Status
		}
		return nil, infraerrors.Newf(http.StatusBadGateway, "OPENAI_CODEX_MODELS_UPSTREAM_FAILED", "codex models manifest upstream error %d: %s", resp.StatusCode, message)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, codexModelsManifestBodyLimit))
	if err != nil {
		return nil, infraerrors.Newf(http.StatusBadGateway, "OPENAI_CODEX_MODELS_UPSTREAM_FAILED", "read codex models manifest response: %v", err)
	}
	return &CodexModelsManifest{Body: body, ETag: resp.Header.Get("ETag")}, nil
}
