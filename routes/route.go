package routes

import (
	"websocket-backend/internal/websocket"

	"github.com/gin-gonic/gin"
)

type Routes struct {
	router *gin.Engine
}

func SetupRouter() *gin.Engine {

	r := Routes{
		router: gin.Default(),
	}

	r.router.Use(func(c *gin.Context) {
		// add header Access-Control-Allow-Origin
		c.Writer.Header().Set("Content-Type", "application/json")
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Max-Age", "86400")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, GET, PUT, DELETE, UPDATE")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, X-Max")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(200)
		} else {
			c.Next()
		}
	})

	// UPDATE FOR WEBSOCKET 9/7/2025

	apiV1WS := r.router.Group("/v1/api/ws")

	// Route for list of allowed commands
	apiV1WS.GET("/allowed", websocket.GetAllowedCommandsHandler())

	apiV1WS.GET("/stream", websocket.NewWebSocketHandler())

	// END OF UPDATE

	return r.router
}
