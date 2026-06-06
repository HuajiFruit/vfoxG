package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func (a *App) appConfigFile() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil || strings.TrimSpace(base) == "" {
		home, homeErr := os.UserHomeDir()
		if homeErr != nil || home == "" {
			if err != nil {
				return "", err
			}
			return "", fmt.Errorf("unable to resolve user config directory")
		}
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, "vfoxG", "config.json"), nil
}

func (a *App) readAppConfig() (AppConfig, error) {
	configFile, err := a.appConfigFile()
	if err != nil {
		return AppConfig{}, err
	}
	data, err := os.ReadFile(configFile)
	if err != nil {
		if os.IsNotExist(err) {
			return AppConfig{}, nil
		}
		return AppConfig{}, err
	}
	var config AppConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return AppConfig{}, err
	}
	return config, nil
}

func (a *App) saveAppConfig(config AppConfig) error {
	configFile, err := a.appConfigFile()
	if err != nil {
		return err
	}
	return a.writeJSONFile(configFile, config)
}
