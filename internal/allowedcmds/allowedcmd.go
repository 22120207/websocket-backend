package allowedcmds

import (
	"strings"
	"sync"
)

var (
	allowedCommands = map[string]bool{
		"tail":       true,
		"ls":         true,
		"journalctl": true,
		"tcpdump":    true,
	}

	blackListCommands = map[string]bool{
		"sudo":      true,
		"rm":        true,
		"systemctl": true,
		"reboot":    true,
		"shutdown":  true,
		"passwd":    true,
		"chown":     true,
		"chmod":     true,
		"kill":      true,
		"killall":   true,
		"init":      true,
		"mount":     true,
		"umount":    true,
		"&&":        true,
	}

	mu sync.RWMutex
)

// IsValidCommand checks if a given command is in the list of allowed commands.
func IsValidCommand(cmd string) bool {
	mu.RLock()
	defer mu.RUnlock()
	return allowedCommands[cmd]
}

func IsBlackListCommand(cmd string) bool {
	mu.RLock()
	defer mu.RUnlock()

	// Normalize the cmd before process
	cmd = strings.ToLower(strings.TrimSpace(cmd))

	tokens := strings.Fields(cmd)
	for _, token := range tokens {
		if blackListCommands[token] {
			return true
		}
	}
	return false
}

// GetAllAllowedCommands returns a slice of all commands currently allowed.
func GetAllAllowedCommands() []string {
	mu.RLock()
	defer mu.RUnlock()
	commands := make([]string, 0, len(allowedCommands))
	for cmd := range allowedCommands {
		commands = append(commands, cmd)
	}
	return commands
}
