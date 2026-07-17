package middleware

import (
	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ip"
	"github.com/gin-gonic/gin"
)

// APIKeyACLClientIP returns the exact client identity used by API-key IP ACL
// authentication. Callers that persist request metadata for a later internal
// replay must use this helper instead of independently trusting forwarding
// headers, otherwise the replay can authenticate a different address.
func APIKeyACLClientIP(c *gin.Context, cfg *config.Config) string {
	trustForwarded := cfg != nil && cfg.TrustForwardedIPForAPIKeyACL()
	return ip.GetSecurityClientIP(c, trustForwarded)
}
