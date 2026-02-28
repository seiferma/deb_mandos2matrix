package matrix

import (
	"encoding/json"
	"fmt"
	"os"
)

type InitialConfig struct {
	BaseConfig
	UserName string `json:"username"`
	Password string `json:"password"`
}

func ParseInitialConfigFromFile(filePath string) (*InitialConfig, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("Could not read config file %s: %w", filePath, err)
	}

	var config InitialConfig
	err = json.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("Could not parse config file %s: %w", filePath, err)
	}

	return &config, nil
}
