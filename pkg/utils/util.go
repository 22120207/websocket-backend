package utils

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"websocket-backend/models"

	log "github.com/sirupsen/logrus"
)

// Set up logger
func SetupLogger() {
	log.SetOutput(os.Stdout)
	log.SetLevel(log.DebugLevel)
}

// Parse the file that contain info about targets/hosts
func ParseInventoryFile(path string) ([]models.SSHConfig, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open inventory file: %w", err)
	}
	defer file.Close()

	var hosts []models.SSHConfig
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "[") {
			continue
		}

		fields := strings.Fields(line)
		entry := models.SSHConfig{}

		for _, field := range fields {
			kv := strings.SplitN(field, "=", 2)
			if len(kv) != 2 {
				continue
			}
			key := strings.ToLower(kv[0])
			value := kv[1]

			switch key {
			case "target":
				entry.Target = value
			case "user":
				entry.Username = value
			case "password":
				entry.Password = value
			case "key":
				entry.KeyPath = value
			}
		}

		if entry.Target != "" && entry.Username != "" && (entry.Password != "" || entry.KeyPath != "") {
			hosts = append(hosts, entry)
		} else {
			log.Printf("Skipping incomplete line: %s", line)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading inventory file: %w", err)
	}

	return hosts, nil
}
