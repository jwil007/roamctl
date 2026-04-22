package config

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"

	"charm.land/log/v2"
	"github.com/BurntSushi/toml"
)

func HandleConfig(template *string, iface *string, edit *bool) (*Config, error) {
	path, err := initConfigFile(template, iface)
	if err != nil {
		return nil, fmt.Errorf("initConfigFile: %w", err)
	}
	if *edit {
		if err = editConfig(path); err != nil {
			return nil, fmt.Errorf("editConfig: %w", err)
		}
	}
	cfg, err := parseConfig(path)
	if err != nil {
		return nil, fmt.Errorf("parseConfig: %w", err)
	}
	return cfg, nil
}

func editConfig(path string) error {
	editors := []string{
		os.Getenv("EDITOR"),
		os.Getenv("VISUAL"),
		"nano",
		"vi",
	}
	for _, editor := range editors {
		if editor != "" {
			cmd := exec.Command(editor, path)
			cmd.Stdin = os.Stdin
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("cmd.Run: %w", err)
			}
			return nil
		}
	}
	return fmt.Errorf("no text editor found, edit file manually at %s", path)
}

func initConfigFile(template *string, iface *string) (string, error) {
	configDir := "/etc/roamctl"
	configPath := filepath.Join(configDir, *iface+".toml")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return "", fmt.Errorf("os.MkdirAll: %w", err)
	}
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
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
	switch *template {
	case "base":
		slog.Info("Overwriting config.toml to default values...")
		err := os.WriteFile(configPath,
			[]byte(defaultConfigTemplate), 0755)
		if err != nil {
			return "", fmt.Errorf("os.WriteFile: %w", err)
		}
		//case "macos":
		//	slog.Info("Overwriting config.toml to MacOS template...")
		//	err := os.WriteFile(configPath,
		//		[]byte(strings.Replace(macOSTemplate, `"wlan0"`, `"`+iface+`"`, 1)), 0755)
		//	if err != nil {
		//		return "", fmt.Errorf("os.WriteFile: %w", err)
		//	}
		//case "ios":
		//	slog.Info("Overwriting config.toml to iOS template...")
		//	err := os.WriteFile(configPath,
		//		[]byte(strings.Replace(iOSTemplate, `"wlan0"`, `"`+iface+`"`, 1)), 0755)
		//	if err != nil {
		//		return "", fmt.Errorf("os.WriteFile: %w", err)
		//	}
		//case "":
		//	f, err := os.ReadFile(configPath)
		//	if err != nil {
		//		return "", fmt.Errorf("os.ReadFile: %w", err)
		//	}
		//	updated := strings.Replace(string(f), `"wlan0"`, `"`+iface+`"`, 1)
		//	err = os.WriteFile(configPath, []byte(updated), 0755)
		//	if err != nil {
		//		return "", fmt.Errorf("os.WriteFile: %w", err)
		//	}
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

//func readIfaceFromFile(path string) string {
//	var c struct {
//		Preferences struct {
//			Interface string
//		}
//	}
//	if _, err := toml.DecodeFile(path, &c); err != nil {
//		return "wlan0"
//	}
//	if c.Preferences.Interface == "" {
//		return "wlan0"
//	}
//	return c.Preferences.Interface
//}
