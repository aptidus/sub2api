package routes

import (
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

// RegisterCommonRoutes 注册通用路由（健康检查、状态等）
func RegisterCommonRoutes(r *gin.Engine) {
	// 健康检查
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Claude Code 遥测日志（忽略，直接返回200）
	r.POST("/api/event_logging/batch", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	// Setup status endpoint (always returns needs_setup: false in normal mode)
	// This is used by the frontend to detect when the service has restarted after setup
	r.GET("/setup/status", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"code": 0,
			"data": gin.H{
				"needs_setup": false,
				"step":        "completed",
			},
		})
	})

	registerSpearRelayRoutes(r)
}

func registerSpearRelayRoutes(r *gin.Engine) {
	dir := spearRelayDir()
	if dir == "" {
		return
	}

	r.GET("/spearrelay", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/spearrelay/")
	})
	r.GET("/spearrelay/", func(c *gin.Context) {
		c.File(filepath.Join(dir, "index.html"))
	})
	serveSpearRelayFile := func(name string) gin.HandlerFunc {
		return func(c *gin.Context) {
			filePath := filepath.Join(dir, name)
			info, err := os.Stat(filePath)
			if err != nil || info.IsDir() {
				c.Status(http.StatusNotFound)
				return
			}
			c.File(filePath)
		}
	}
	for _, name := range []string{"app.js", "config.js", "config.example.js", "styles.css"} {
		r.GET("/spearrelay/"+name, serveSpearRelayFile(name))
	}
}

func spearRelayDir() string {
	for _, candidate := range []string{
		"spearrelay",
		filepath.Join("..", "spearrelay"),
		filepath.Join("/app", "spearrelay"),
	} {
		info, err := os.Stat(filepath.Join(candidate, "index.html"))
		if err == nil && !info.IsDir() {
			return candidate
		}
	}
	return ""
}
