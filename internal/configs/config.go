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
