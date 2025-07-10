package websocket

import (
	"net/http"
	"strings"
	"sync"
	"websocket-backend/internal/configs"
	"websocket-backend/internal/helpers"

	"github.com/gin-gonic/gin"
)

var (
	allowedCommands = make([]string, 0)

	blackListCommands = make([]string, 0)

	mu sync.RWMutex
)

// IsValidCommand checks if a given command is in the list of allowed commands.
func isValidCommand(cmd string) bool {
	return helpers.IsStrInStrLst(cmd, allowedCommands)
}

func isBlackListCommand(cmd string) bool {
	// Normalize the cmd before process
	cmd = strings.ToLower(strings.TrimSpace(cmd))

	for _, value := range blackListCommands {
		if strings.Contains(cmd, value) {
			return true
		}
	}

	return false
}

// getAllowedCommands returns a slice of all commands currently allowed.
func getAllowedCommands() []string {
	return allowedCommands
}

// A func that load all allowed cmd from the config
func LoadAllowedCmds(c configs.Config) {
	mu.Lock()
	defer mu.Unlock()

	allowed := c["websocket"].(map[string]interface{})["allowed_cmds"].([]interface{})

	for _, value := range allowed {
		allowedCommands = append(allowedCommands, value.(string))
	}
}

// A func that load all black list cmd from the config
func LoadBlacklistCmds(c configs.Config) {
	mu.Lock()
	defer mu.Unlock()

	blacklisted := c["websocket"].(map[string]interface{})["blacklist_cmds"].([]interface{})

	for _, value := range blacklisted {
		blackListCommands = append(blackListCommands, value.(string))
	}
}

// GetAllowedCommandsHandler handle request and return list of allowed command
func GetAllowedCommandsHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		commands := getAllowedCommands()

		c.JSON(http.StatusOK, gin.H{
			"status":   "success",
			"commands": commands,
		})
	}
}
