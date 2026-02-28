package matrix

import (
	"encoding/json"
	"fmt"
	"os"
)

type RuntimeConfig struct {
	BaseConfig
	UserId      string `json:"user_id"`
	AccessToken string `json:"access_token"`
	DeviceId    string `json:"device_id"`
}

func ParseRuntimeConfigFromFile(filePath string) (*RuntimeConfig, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("Could not read config file %s: %w", filePath, err)
	}

	var config RuntimeConfig
	err = json.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("Could not parse config file %s: %w", filePath, err)
	}

	return &config, nil
}

func (rc *RuntimeConfig) SerializeToFile(filePath string) error {
	data, err := json.Marshal(rc)
	if err != nil {
		return fmt.Errorf("Could not serialize config to JSON: %w", err)
	}

	err = os.WriteFile(filePath, data, 0600)
	if err != nil {
		return fmt.Errorf("Could not write config to file %s: %w", filePath, err)
	}

	return nil
}
