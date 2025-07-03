package config

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"websocket-backend/internal/customSSH"

	"github.com/joho/godotenv"
)

type Config struct {
	Port   string
	Target []customSSH.SSHConfig
}

var (
	config     *Config
	configOnce sync.Once
)

func LoadConfig() (*Config, error) {
	var err error

	configOnce.Do(func() {
		if envErr := godotenv.Load(); envErr != nil {
			err = fmt.Errorf("error loading .env file: %w", envErr)
			return
		}

		config = &Config{
			Port: os.Getenv("PORT"),
		}

		if config.Port == "" {
			config.Port = "8080"
		}

		inventoryPath := os.Getenv("INVENTORY_TARGET")
		if inventoryPath == "" {
			err = fmt.Errorf("INVENTORY_TARGET not set in environment")
			return
		}

		hosts, parseErr := parseInventoryFile(inventoryPath)
		if parseErr != nil {
			err = parseErr
			return
		}

		config.Target = hosts
	})

	return config, err
}

// parseInventoryFile reads and parses the inventory file.
func parseInventoryFile(path string) ([]customSSH.SSHConfig, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open inventory file: %w", err)
	}
	defer file.Close()

	var hosts []customSSH.SSHConfig
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "[") {
			continue
		}

		fields := strings.Fields(line)
		entry := customSSH.SSHConfig{}

		for _, field := range fields {
			kv := strings.SplitN(field, "=", 2)
			if len(kv) != 2 {
				continue
			}
			key := strings.ToLower(kv[0])
			value := kv[1]

			switch key {
			case "target":
				entry.Target = value + ":22"
			case "user":
				entry.Username = value
			case "password":
				entry.Password = value
			case "key":
				entry.KeyPath = value
			}
		}

		// Only add if target and username are present, and either password or keypath is present.
		if entry.Target != "" && entry.Username != "" && (entry.Password != "" || entry.KeyPath != "") {
			hosts = append(hosts, entry)
		} else {
			log.Printf("Skipping incomplete line in inventory: %s", line)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading inventory file: %w", err)
	}

	return hosts, nil
}

// This method will finds and returns an SSHConfig for a given target, or nil if not found.
func (c *Config) GetSSHClientConfig(target string) *customSSH.SSHConfig {
	target += ":22"

	for _, found := range c.Target {
		fmt.Println(found.Target)
		if found.Target == target {
			copy := found
			return &copy
		}
	}
	return nil
}
