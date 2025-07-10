package websocket

import (
	"bufio"
	"context"
	"encoding/base64"
	"fmt"
	"os/exec"
	"strings"
	"sync"

	log "github.com/sirupsen/logrus"
)

// RunAndStream decodes the Base64-encoded command, executes it on the current host,
func runAndStream(
	ctx context.Context,
	encodedCmd string,
	wsClient *Client,
) error {
	// Decode the Base64-encoded command
	decodedCmdBytes, err := base64.StdEncoding.DecodeString(encodedCmd)
	if err != nil {
		log.Error("Failed to Base64 decode command:", err)
		return fmt.Errorf("failed to decode command: %w", err)
	}
	fullCmd := string(decodedCmdBytes)
	log.Info("Attempting to run command:", strings.ReplaceAll(fullCmd, "\n", ""))

	parts := strings.Fields(fullCmd)
	if len(parts) == 0 {
		return fmt.Errorf("empty command received after decoding")
	}
	baseCmd := parts[0]

	// Check if the command is in the black list (contain sudo, rm, systemctl, ...)
	if isBlackListCommand(fullCmd) {
		log.Error("Attempted to run command injection in black list:", baseCmd)
		return fmt.Errorf("command '%s' is in black list", fullCmd)
	}

	// Check if the command is allowed
	if !isValidCommand(baseCmd) {
		log.Error("Attempted to run disallowed command:", baseCmd)
		return fmt.Errorf("command '%s' is not allowed", baseCmd)
	}

	// Create a cancel func for the command
	_, cmdCancel := context.WithCancel(ctx)
	defer cmdCancel()

	cmd := exec.CommandContext(ctx, "bash", "-c", fullCmd)

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdout pipe: %w", err)
	}

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to get stderr pipe: %w", err)
	}

	// Set the cancel function in the WebSocket client
	wsClient.SetCmdCancelFunc(cmdCancel)

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start command: %w", err)
	}

	wsClient.SetCmd(cmd)

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
			log.Error("Error streaming stdout:", err)
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
			log.Error("Error streaming stderr:", err)
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
				log.Info("Command process killed due to context cancellation")
			}
			return
		default:
			if err := cmd.Wait(); err != nil {
				log.Error("Command exited with error:", err)
			} else {
				log.Info("Command finished successfully.")
				wsClient.Send([]byte("command finished successfully"))
			}
		}
	}()

	// Wait for all goroutines to complete
	wg.Wait()
	log.Info("All streams and command execution complete for:", fullCmd)
	return nil
}
