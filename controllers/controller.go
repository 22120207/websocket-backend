package controllers

import (
	"net/http"
	"websocket-backend/internal/allowedcmds"

	"github.com/gin-gonic/gin"
)

// Return list of available commands
func GetAllowedCommandsHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		allowedCmds := allowedcmds.GetAllowedCommands()

		c.JSON(http.StatusOK, gin.H{
			"allowedCommands": allowedCmds,
		})
	}
}
