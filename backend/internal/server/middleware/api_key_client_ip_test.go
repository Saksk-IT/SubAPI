package middleware

import (
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestAPIKeyACLClientIPMatchesAuthenticationTrustPolicy(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, engine := gin.CreateTestContext(httptest.NewRecorder())
	require.NoError(t, engine.SetTrustedProxies(nil))
	c.Request = httptest.NewRequest("GET", "/", nil)
	c.Request.RemoteAddr = "9.9.9.9:12345"
	c.Request.Header.Set("CF-Connecting-IP", "1.2.3.4")
	c.Request.Header.Set("X-Forwarded-For", "1.2.3.4")

	cfg := &config.Config{}
	require.Equal(t, "9.9.9.9", APIKeyACLClientIP(c, cfg))

	cfg.SetTrustForwardedIPForAPIKeyACL(true)
	require.Equal(t, "1.2.3.4", APIKeyACLClientIP(c, cfg))
}
