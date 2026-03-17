package roam

import (
	"fmt"
	"log"
	"os"
	"os/user"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

func HandleConfig(resetDefaults *bool) (*Config, error) {
	path, err := initConfigFile(resetDefaults)
	if err != nil {
		return nil, fmt.Errorf("initConfigFile: %w", err)
	}
	log.Printf("Config file path: %s", path)
	cfg, err := parseConfig(path)
	if err != nil {
		return nil, fmt.Errorf("parseConfig: %w", err)
	}
	return cfg, nil
}

func getConfigDir() (string, error) {
	if sudoUser := os.Getenv("SUDO_USER"); sudoUser != "" {
		log.Printf("Running as sudo...")
		u, err := user.Lookup(sudoUser)
		if err != nil {
			return "", fmt.Errorf("user.Lookup: %w", err)
		}
		return filepath.Join(u.HomeDir, ".config", "roamctl"), nil
	}
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("os.UserConfigDir: %w", err)
	}
	return filepath.Join(configDir, "roamctl"), nil
}

func initConfigFile(resetDefaults *bool) (string, error) {
	configDir, err := getConfigDir()
	configPath := filepath.Join(configDir, "config.toml")
	if err != nil {
		return "", fmt.Errorf("getConfigDir: %w", err)
	}
	if err = os.MkdirAll(configDir, 0755); err != nil {
		return "", fmt.Errorf("os.MkdirAll: %w", err)
	}
	if _, err = os.Stat(configPath); os.IsNotExist(err) {
		f, err := os.Create(configPath)
		if err != nil {
			return "", fmt.Errorf("os.Create: %w", err)
		}
		defer func() {
			if err = f.Close(); err != nil {
				log.Printf("error closing file: %v", err)
			}
		}()
		if _, err = f.WriteString(defaultConfigTemplate); err != nil {
			return "", fmt.Errorf("encodeConfig: %w", err)
		}
	}
	if *resetDefaults {
		log.Printf("Overwriting config.toml to default values...")
		err = os.WriteFile(configPath, []byte(defaultConfigTemplate), 0755)
	}
	return configPath, nil
}

func parseConfig(configPath string) (*Config, error) {
	var cfg Config
	_, err := toml.DecodeFile(configPath, &cfg)
	if err != nil {
		return nil, fmt.Errorf("toml.DecodeFile: %w", err)
	}
	return &cfg, nil
}
