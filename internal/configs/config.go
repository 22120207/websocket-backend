package configs

import (
	"encoding/json"
	"os"
)

type Config map[string]interface{}

func (c *Config) Load(configFile string) error {
	data, err := os.ReadFile(configFile)
	if err != nil {
		return err
	}

	err = json.Unmarshal(data, c)
	if err != nil {
		return err
	}

	return nil
}

func (c *Config) ChangeDedicatedStatus(status string) {
	(*c)["hw"].(map[string]interface{})["isDedicated"] = status
}

func (c *Config) WriteConfig(configFile string) error {
	data, err := json.MarshalIndent(c, "", "    ")
	if err != nil {
		return err
	}

	err = os.WriteFile(configFile, data, 0644)
	if err != nil {
		return err
	}

	return nil
}
