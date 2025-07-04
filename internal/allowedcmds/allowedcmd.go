package allowedcmds

import "sync"

var (
	allowedCommands = map[string]bool{
		"tail":       true,
		"ls":         true,
		"journalctl": true,
		"tcpdump":    true,
	}
	mu sync.RWMutex
)

// IsValidCommand checks if a given command is in the list of allowed commands.
func IsValidCommand(cmd string) bool {
	mu.RLock()
	defer mu.RUnlock()
	return allowedCommands[cmd]
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
