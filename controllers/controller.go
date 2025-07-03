package controllers

import (
	"net/http"
	"websocket-backend/internal/allowedcmds"

	"github.com/gin-gonic/gin"
)

// GetAllowedCommandsHandler all allowed commands
func GetAllowedCommandsHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		commands := allowedcmds.GetAllAllowedCommands()

		c.JSON(http.StatusOK, gin.H{
			"status":   "success",
			"commands": commands,
		})
	}
}
