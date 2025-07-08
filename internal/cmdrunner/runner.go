package cmdrunner

import (
	"bufio"
	"context"
	"encoding/base64"
	"fmt"
	"os/exec"
	"strings"
	"sync"

	"websocket-backend/internal/allowedcmds"
	"websocket-backend/pkg/utils"
)

// RunAndStream decodes the Base64-encoded command, executes it on the current host,
func RunAndStream(
	ctx context.Context,
	encodedCmd string,
	wsClient StreamSender,
) error {
	// Decode the Base64-encoded command
	decodedCmdBytes, err := base64.StdEncoding.DecodeString(encodedCmd)
	if err != nil {
		utils.Error("Failed to Base64 decode command:", err)
		return fmt.Errorf("failed to decode command: %w", err)
	}
	fullCmd := string(decodedCmdBytes)
	utils.Info("Attempting to run command:", strings.ReplaceAll(fullCmd, "\n", ""))

	parts := strings.Fields(fullCmd)
	if len(parts) == 0 {
		return fmt.Errorf("empty command received after decoding")
	}
	baseCmd := parts[0]

	// Check if the command is in the black list (contain sudo, rm, systemctl, ...)
	if allowedcmds.IsBlackListCommand(fullCmd) {
		utils.Error("Attempted to run command injection in black list:", baseCmd)
		return fmt.Errorf("command '%s' is in black list", baseCmd)
	}

	// Check if the command is allowed
	if !allowedcmds.IsValidCommand(baseCmd) {
		utils.Error("Attempted to run disallowed command:", baseCmd)
		return fmt.Errorf("command '%s' is not allowed", baseCmd)
	}

	// Create a context for the command
	ctx, cmdCancel := context.WithCancel(ctx)
	defer cmdCancel()

	cmd := exec.CommandContext(ctx, baseCmd, parts[1:]...)

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdout pipe: %w", err)
	}

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to get stderr pipe: %w", err)
	}

	// Set the cancel function in the WebSocket client
	wsClient.SetCancelFunc(cmdCancel)

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start command: %w", err)
	}

	var wg sync.WaitGroup

	// Stream stdout
	wg.Add(1)
	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stdoutPipe)
		for scanner.Scan() {
			select {
			case <-ctx.Done():
				return
			default:
				line := scanner.Bytes()
				lineCopy := make([]byte, len(line))
				copy(lineCopy, line)
				wsClient.Send(lineCopy)
			}
		}
		if err := scanner.Err(); err != nil {
			utils.Error("Error streaming stdout:", err)
		}
	}()

	// Stream stderr
	wg.Add(1)
	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stderrPipe)
		for scanner.Scan() {
			select {
			case <-ctx.Done():
				return
			default:
				line := scanner.Bytes()
				lineCopy := make([]byte, len(line))
				copy(lineCopy, line)
				wsClient.Send(lineCopy)
			}
		}
		if err := scanner.Err(); err != nil {
			utils.Error("Error streaming stderr:", err)
		}
	}()

	// Wait for command to finish or kill on context cancellation
	wg.Add(1)
	go func() {
		defer wg.Done()
		select {
		case <-ctx.Done():
			if cmd.Process != nil {
				cmd.Process.Kill()
				utils.Info("Command process killed due to context cancellation")
			}
			return
		default:
			if err := cmd.Wait(); err != nil {
				utils.Error("Command exited with error:", err)
			} else {
				utils.Info("Command finished successfully.")
			}
		}
	}()

	// Wait for all goroutines to complete
	wg.Wait()
	utils.Info("All streams and command execution complete for:", fullCmd)
	return nil
}
