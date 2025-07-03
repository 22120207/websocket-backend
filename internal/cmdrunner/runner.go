package cmdrunner

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"websocket-backend/internal/allowedcmds"
	"websocket-backend/internal/config"
	"websocket-backend/internal/customSSH"
	"websocket-backend/pkg/utils"
)

type CommandRunner struct {
	Config *config.Config
}

// NewCommandRunner creates and returns a new CommandRunner instance.
func NewCommandRunner(cfg *config.Config) *CommandRunner {
	return &CommandRunner{
		Config: cfg,
	}
}

// RunAndStream decodes the Base64-encoded command, executes it on the target host via SSH,
// and streams its stdout and stderr output back to the provided StreamSender.
// It respects the provided context for cancellation.
func (cr *CommandRunner) RunAndStream(
	ctx context.Context,
	encodedCmd string,
	targetHost string,
	wsClient StreamSender,
) error {
	decodedCmdBytes, err := base64.StdEncoding.DecodeString(encodedCmd)
	if err != nil {
		utils.Error("Failed to Base64 decode command:", err)
		return fmt.Errorf("failed to decode command: %w", err)
	}
	fullCmd := string(decodedCmdBytes)
	utils.Info("Attempting to run command:", fullCmd, "on target:", targetHost)

	parts := strings.Fields(fullCmd)
	if len(parts) == 0 {
		return fmt.Errorf("empty command received after decoding")
	}
	baseCmd := parts[0]

	if !allowedcmds.IsValidCommand(baseCmd) {
		utils.Error("Attempted to run disallowed command:", baseCmd)
		return fmt.Errorf("command '%s' is not allowed", baseCmd)
	}
	utils.Info("Command '" + baseCmd + "' is allowed. Proceeding with SSH execution.")

	// Get the SSHConfig for the target host from the main config
	sshConnConfig := cr.Config.GetSSHClientConfig(targetHost)
	if sshConnConfig == nil || sshConnConfig.Username == "" {
		return fmt.Errorf("SSH configuration for target '%s' not found or incomplete", targetHost)
	}

	sshClientConfig, err := customSSH.GetSSHClientConfig(sshConnConfig)
	if err != nil {
		utils.Error("Failed to get SSH client config:", err)
		return fmt.Errorf("failed to get SSH client config: %w", err)
	}

	sshConn, err := customSSH.Connect(sshConnConfig.Target, sshClientConfig)
	if err != nil {
		utils.Error("Failed to connect to SSH host", sshConnConfig.Target, ":", err)
		return fmt.Errorf("failed to connect to SSH host %s: %w", sshConnConfig.Target, err)
	}
	defer func() {
		utils.Info("Closing SSH connection to", sshConnConfig.Target)
		sshConn.Close()
	}()

	stdoutPipe, stderrPipe, session, err := customSSH.ExecuteStream(sshConn, fullCmd)
	if err != nil {
		utils.Error("Failed to execute command '"+fullCmd+"' on", sshConnConfig.Target, ":", err)
		return fmt.Errorf("failed to execute command '%s' on %s: %w", fullCmd, sshConnConfig.Target, err)
	}
	defer func() {
		utils.Info("Closing SSH session for command:", fullCmd)
		session.Close()
	}()

	var wg sync.WaitGroup
	buffer := make([]byte, 4096)

	wg.Add(1)
	go func() {
		defer wg.Done()
		_, err := io.CopyBuffer(&websocketStreamWriter{wsClient: wsClient, streamType: "stdout"}, stdoutPipe, buffer)
		if err != nil && err != io.EOF {
			utils.Error(fmt.Sprintf("Error streaming stdout for command '%s' on %s: %v", fullCmd, sshConnConfig.Target, err))
		}
		utils.Info("Stdout streaming finished for command:", fullCmd)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		_, err := io.CopyBuffer(&websocketStreamWriter{wsClient: wsClient, streamType: "stderr"}, stderrPipe, buffer)
		if err != nil && err != io.EOF {
			utils.Error(fmt.Sprintf("Error streaming stderr for command '%s' on %s: %v", fullCmd, sshConnConfig.Target, err))
		}
		utils.Info("Stderr streaming finished for command:", fullCmd)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		sessionDone := make(chan error, 1)
		go func() {
			sessionDone <- session.Wait()
		}()

		select {
		case <-ctx.Done():
			utils.Info("Client context cancelled for command:", fullCmd, ". Attempting to terminate remote command.")
			if err := session.Signal(customSSH.SIGTERM); err != nil {
				utils.Error("Failed to send SIGTERM to remote process for command '"+fullCmd+"':", err)
			}
			select {
			case <-time.After(5 * time.Second):
				utils.Info("Remote command did not terminate gracefully after SIGTERM. Sending SIGKILL.")
				if err := session.Signal(customSSH.SIGKILL); err != nil {
					utils.Error("Failed to send SIGKILL to remote process for command '"+fullCmd+"':", err)
				}
			case <-sessionDone:
				utils.Info("Remote command terminated gracefully after SIGTERM.")
			}
			return
		case <-time.After(10 * time.Minute):
			utils.Info("Command '" + fullCmd + "' timed out after 10 minutes. Terminating remote command.")
			if err := session.Signal(customSSH.SIGTERM); err != nil {
				utils.Error("Failed to send SIGTERM to remote process on timeout for command '"+fullCmd+"':", err)
			}
			return
		case err := <-sessionDone:
			if err != nil {
				utils.Error("Remote command '"+fullCmd+"' exited with error:", err)
			} else {
				utils.Info("Remote command '" + fullCmd + "' finished gracefully.")
			}
			return
		}
	}()

	wg.Wait()

	utils.Info("All streaming and command management goroutines for", fullCmd, "have completed.")
	return nil
}

type websocketStreamWriter struct {
	wsClient   StreamSender
	streamType string
}

func (w *websocketStreamWriter) Write(p []byte) (n int, err error) {
	w.wsClient.Send(p)
	return len(p), nil
}
