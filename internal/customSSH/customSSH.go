package customSSH

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"
	"websocket-backend/pkg/utils"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

type SSHConfig struct {
	Target   string
	Username string
	Password string
	KeyPath  string
	Timeout  time.Duration
}

func (s *SSHConfig) UsePrivateKey() bool {
	return s.KeyPath != ""
}

const (
	SIGTERM = ssh.Signal("TERM")
	SIGKILL = ssh.Signal("KILL")
)

// GetSSHClientConfig creates an *ssh.ClientConfig from the provided custom SSHConfig.
func GetSSHClientConfig(cfg *SSHConfig) (*ssh.ClientConfig, error) {
	var authMethods []ssh.AuthMethod

	if cfg.UsePrivateKey() {
		key, err := os.ReadFile(cfg.KeyPath)
		if err != nil {
			return nil, fmt.Errorf("unable to read private key from %s: %w", cfg.KeyPath, err)
		}

		signer, err := ssh.ParsePrivateKey(key)
		if err != nil {
			return nil, fmt.Errorf("unable to parse private key from %s: %w", cfg.KeyPath, err)
		}
		authMethods = append(authMethods, ssh.PublicKeys(signer))
		utils.Debug(fmt.Sprintf("Using private key authentication from %s for user %s", cfg.KeyPath, cfg.Username))
	} else if cfg.Password != "" {
		authMethods = append(authMethods, ssh.Password(cfg.Password))
		utils.Debug(fmt.Sprintf("Using password authentication for user %s", cfg.Username))
	} else {
		return nil, fmt.Errorf("no authentication method (password or private key) provided in SSHConfig")
	}

	knownHostsPath := os.ExpandEnv("$HOME/.ssh/known_hosts")
	hostKeyCallback, err := knownhosts.New(knownHostsPath)
	if err != nil {
		utils.Error("Failed to load known_hosts file from "+knownHostsPath+", defaulting to insecure host key check:", err)
		hostKeyCallback = ssh.InsecureIgnoreHostKey()
	} else {
		utils.Debug("Using " + knownHostsPath + " for host key verification.")
	}

	clientConfig := &ssh.ClientConfig{
		User:            cfg.Username,
		Auth:            authMethods,
		Timeout:         cfg.Timeout,
		HostKeyCallback: hostKeyCallback,
	}

	return clientConfig, nil
}

// Connect establishes an SSH connection to the specified host.
func Connect(host string, config *ssh.ClientConfig) (*ssh.Client, error) {
	utils.Info("Attempting to connect to SSH host:", host)
	client, err := ssh.Dial("tcp", host, config)
	if err != nil {
		return nil, fmt.Errorf("failed to dial SSH server %s: %w", host, err)
	}
	utils.Info("Successfully connected to SSH host:", host)
	return client, nil
}

// ExecuteStream opens a new SSH session, starts a command, and provides
// io.Readers for its standard output and standard error streams.
func ExecuteStream(conn *ssh.Client, command string) (stdout io.Reader, stderr io.Reader, session *ssh.Session, err error) {
	session, err = conn.NewSession()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create SSH session: %w", err)
	}

	// Request a pseudo-terminal (PTY)
	// This makes the remote command behave like it's running in a real terminal,
	modes := ssh.TerminalModes{
		ssh.ECHO:          0, // disable echoing
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}
	if err := session.RequestPty("xterm", 40, 80, modes); err != nil {
		session.Close()
		return nil, nil, nil, fmt.Errorf("failed to request pty: %w", err)
	}

	stdoutPipe, err := session.StdoutPipe()
	if err != nil {
		session.Close()
		return nil, nil, nil, fmt.Errorf("failed to get stdout pipe: %w", err)
	}

	stderrPipe, err := session.StderrPipe()
	if err != nil {
		session.Close()
		return nil, nil, nil, fmt.Errorf("failed to get stderr pipe: %w", err)
	}

	if err := session.Start(command); err != nil {
		session.Close()
		return nil, nil, nil, fmt.Errorf("failed to start remote command '%s': %w", command, err)
	}

	utils.Debug("Remote command '" + strings.ReplaceAll(command, "\n", " ") + "' started.")
	return stdoutPipe, stderrPipe, session, nil
}
