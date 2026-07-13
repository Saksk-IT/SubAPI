package service

import "context"

// apiKeyAuthCacheBypassContextKey is intentionally private. HTTP headers,
// query parameters, and request bodies cannot manufacture this value; only a
// trusted in-process caller can opt a request into a current database read.
type apiKeyAuthCacheBypassContextKey struct{}

// WithAPIKeyAuthCacheBypass marks an internal request that must authenticate
// against the current database row without reading, joining, or populating the
// API-key L1/L2 caches. The normal authentication middleware still performs
// every key, user, group, ACL, quota, subscription, and balance check.
func WithAPIKeyAuthCacheBypass(ctx context.Context) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, apiKeyAuthCacheBypassContextKey{}, true)
}

func shouldBypassAPIKeyAuthCache(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	bypass, _ := ctx.Value(apiKeyAuthCacheBypassContextKey{}).(bool)
	return bypass
}
