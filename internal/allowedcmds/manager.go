package allowedcmds

import "slices"

func GetAllowedCommands() []string {
	return []string{"tcpdum", "tail", "top"}
}

func IsValidCommand(cmd string) bool {
	return slices.Contains(GetAllowedCommands(), cmd)
}
