package routes

import (
	"websocket-backend/controllers"
	"websocket-backend/internal/config"
	"websocket-backend/internal/websocket"
	"websocket-backend/pkg/utils"

	"github.com/gin-gonic/gin"
)

type Routes struct {
	router    *gin.Engine
	appConfig *config.Config
}

func NewRoutes(cfg *config.Config) *Routes {
	r := &Routes{
		router:    gin.Default(),
		appConfig: cfg,
	}

	r.router.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Content-Type", "application/json")
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Max-Age", "86400")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, GET, PUT, DELETE, UPDATE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, X-Max")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(200)
		} else {
			c.Next()
		}
	})

	return r
}

func (r *Routes) Setup() *gin.Engine {
	// API route group for HTTP endpoints
	httpGroup := r.router.Group("/http")
	{
		httpGroup.GET("/allowed", controllers.GetAllowedCommandsHandler())
		httpGroup.GET("/health", func(c *gin.Context) {
			c.JSON(200, gin.H{"status": "ok"})
		})
	}

	// API route group for WebSocket endpoints
	wsGroup := r.router.Group("/ws")
	{
		wsGroup.GET("", websocket.NewWebSocketHandler())
	}

	utils.Info("Router setup complete. HTTP endpoints under /http, WebSocket endpoint /ws")

	return r.router
}
