package ip

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestGetTrustedClientIPIgnoresForwardedHeadersFromUntrustedRemote(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	require.NoError(t, r.SetTrustedProxies([]string{"10.0.0.0/8"}))
	r.GET("/t", func(c *gin.Context) {
		c.String(200, GetTrustedClientIP(c))
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/t", nil)
	req.RemoteAddr = "8.8.8.8:12345"
	req.Header.Set("X-Forwarded-For", "1.2.3.4")
	req.Header.Set("X-Real-IP", "1.2.3.4")
	r.ServeHTTP(w, req)

	require.Equal(t, 200, w.Code)
	require.Equal(t, "8.8.8.8", w.Body.String())
}

func TestGetTrustedClientIPUsesForwardedHeaderFromTrustedRemote(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	require.NoError(t, r.SetTrustedProxies([]string{"10.0.0.0/8"}))
	r.GET("/t", func(c *gin.Context) {
		c.String(200, GetTrustedClientIP(c))
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/t", nil)
	req.RemoteAddr = "10.1.2.3:12345"
	req.Header.Set("X-Forwarded-For", "1.2.3.4")
	r.ServeHTTP(w, req)

	require.Equal(t, 200, w.Code)
	require.Equal(t, "1.2.3.4", w.Body.String())
}
