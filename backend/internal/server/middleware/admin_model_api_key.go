package middleware

import (
	"context"
	"crypto/subtle"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// AdminModelAPIKeyAccess lets the global admin API key make narrowly controlled
// model calls. The model handlers still pin allowed admin-direct model aliases
// and use normal account selection/failover.
func AdminModelAPIKeyAccess(settingService *service.SettingService, apiKeyService *service.APIKeyService) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := extractBearerOrAPIKeyHeader(c)
		if key == "" || !strings.HasPrefix(key, service.AdminAPIKeyPrefix) {
			c.Next()
			return
		}
		if settingService == nil || apiKeyService == nil {
			AbortWithError(c, 503, "ADMIN_MODEL_ACCESS_UNAVAILABLE", "admin model API key access is not configured")
			return
		}

		storedKey, err := settingService.GetAdminAPIKey(c.Request.Context())
		if err != nil {
			AbortWithError(c, 500, "INTERNAL_ERROR", "Internal server error")
			return
		}
		if storedKey == "" || subtle.ConstantTimeCompare([]byte(key), []byte(storedKey)) != 1 {
			c.Next()
			return
		}

		apiKey, err := apiKeyService.BuildAdminModelAPIKeyContext(c.Request.Context())
		if err != nil {
			AbortWithError(c, 503, "ADMIN_MODEL_ACCESS_UNAVAILABLE", err.Error())
			return
		}

		c.Set(string(ContextKeyAPIKey), apiKey)
		c.Set(string(ContextKeyUser), AuthSubject{
			UserID:      apiKey.User.ID,
			Concurrency: apiKey.User.Concurrency,
		})
		c.Set(string(ContextKeyUserRole), apiKey.User.Role)
		c.Set("auth_method", "admin_model_api_key")
		if service.IsGroupContextValid(apiKey.Group) {
			c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), ctxkey.Group, apiKey.Group))
		}

		c.Next()
	}
}

func extractBearerOrAPIKeyHeader(c *gin.Context) string {
	authHeader := c.GetHeader("Authorization")
	if authHeader != "" {
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
			return strings.TrimSpace(parts[1])
		}
	}
	if key := strings.TrimSpace(c.GetHeader("x-api-key")); key != "" {
		return key
	}
	return strings.TrimSpace(c.GetHeader("x-goog-api-key"))
}
