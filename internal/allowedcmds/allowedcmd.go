package allowedcmds

import "sync"

var (
	allowedCommands = map[string]bool{
		"tail":    true,
		"top":     true,
		"tcpdump": true,
	}
	mu sync.RWMutex // Mutex to protect access to allowedCommands
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

// AddAllowedCommand adds a new command to the allowed list.
func AddAllowedCommand(cmd string) {
	mu.Lock()
	defer mu.Unlock()
	allowedCommands[cmd] = true
}

// RemoveAllowedCommand removes a command from the allowed list.
func RemoveAllowedCommand(cmd string) {
	mu.Lock()
	defer mu.Unlock()
	delete(allowedCommands, cmd)
}
