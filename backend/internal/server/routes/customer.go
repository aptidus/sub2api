package routes

import (
	"time"

	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/middleware"
	servermiddleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// RegisterCustomerRoutes exposes only the self-service endpoints needed by the
// standalone SpearRelay customer site. It shares the same Sub2API userbase and
// JWTs, but deliberately avoids the backend-mode admin-only guard that protects
// the operator/admin WebUI.
func RegisterCustomerRoutes(
	v1 *gin.RouterGroup,
	h *handler.Handlers,
	jwtAuth servermiddleware.JWTAuthMiddleware,
	redisClient *redis.Client,
) {
	rateLimiter := middleware.NewRateLimiter(redisClient)

	customer := v1.Group("/customer")
	{
		auth := customer.Group("/auth")
		{
			auth.POST("/register", rateLimiter.LimitWithOptions("customer-auth-register", 5, time.Minute, middleware.RateLimitOptions{
				FailureMode: middleware.RateLimitFailClose,
			}), h.Auth.Register)
			auth.POST("/login", rateLimiter.LimitWithOptions("customer-auth-login", 20, time.Minute, middleware.RateLimitOptions{
				FailureMode: middleware.RateLimitFailClose,
			}), h.Auth.CustomerLogin)
			auth.POST("/login/2fa", rateLimiter.LimitWithOptions("customer-auth-login-2fa", 20, time.Minute, middleware.RateLimitOptions{
				FailureMode: middleware.RateLimitFailClose,
			}), h.Auth.CustomerLogin2FA)
			auth.POST("/send-verify-code", rateLimiter.LimitWithOptions("customer-auth-send-verify-code", 5, time.Minute, middleware.RateLimitOptions{
				FailureMode: middleware.RateLimitFailClose,
			}), h.Auth.SendVerifyCode)
			auth.POST("/refresh", rateLimiter.LimitWithOptions("customer-refresh-token", 30, time.Minute, middleware.RateLimitOptions{
				FailureMode: middleware.RateLimitFailClose,
			}), h.Auth.CustomerRefreshToken)
			auth.POST("/logout", h.Auth.Logout)
		}

		authenticated := customer.Group("")
		authenticated.Use(gin.HandlerFunc(jwtAuth))
		{
			authenticated.GET("/user/profile", h.User.GetProfile)

			keys := authenticated.Group("/keys")
			{
				keys.GET("", h.APIKey.List)
				keys.GET("/:id", h.APIKey.GetByID)
				keys.POST("/:id/rotate", h.APIKey.Rotate)
			}

			payment := authenticated.Group("/payment")
			{
				payment.GET("/checkout-info", h.Payment.GetCheckoutInfo)

				orders := payment.Group("/orders")
				{
					orders.POST("", h.Payment.CreateOrder)
					orders.POST("/verify", h.Payment.VerifyOrder)
					orders.GET("/my", h.Payment.GetMyOrders)
					orders.GET("/:id", h.Payment.GetOrder)
				}
			}

			subscriptions := authenticated.Group("/subscriptions")
			{
				subscriptions.GET("/summary", h.Subscription.GetSummary)
			}
		}
	}
}
