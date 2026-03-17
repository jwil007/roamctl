package roam

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/BurntSushi/toml"
)

func HandleConfig() (*Config, error) {
	path, err := initConfigFile()
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

func initConfigFile() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("os.UserConfigDir: %w", err)
	}
	configPath := filepath.Join(configDir, "roamctl", "config.toml")
	if err = os.MkdirAll(filepath.Join(configDir, "roamctl"), 0755); err != nil {
		return "", fmt.Errorf("os.MkdirAll: %w", err)
	}
	if _, err = os.Stat(configPath); os.IsNotExist(err) {
		f, err := os.Create(configPath)
		if err != nil {
			return "", fmt.Errorf("os.Create: %w", err)
		}
		defer func() {
			if err := f.Close(); err != nil {
				log.Printf("error closing file: %v", err)
			}
		}()
		if err = encodeConfig(f, defaultConfig()); err != nil {
			return "", fmt.Errorf("encodeConfig: %w", err)
		}
	}
	return configPath, nil
}

func defaultConfig() Config {
	thresholds := Thresholds{
		RSSI:       -67,
		DataRate:   0,
		ScoreDelta: 5,
	}
	timing := Timing{
		SuccessBackoffTime:      6 * time.Second,
		FailureBackoffTime:      2 * time.Second,
		NoCandidatesBackoffTime: 3 * time.Second,
		SigPollInterval:         300 * time.Millisecond,
		BGScanInterval:          30 * time.Second,
		MaxScanAge:              10 * time.Second,
	}
	scoreWeights := ScoreWeights{
		RSSI:         100,
		MinRSSI:      -80,
		MaxRSSI:      -40,
		SNR:          0,
		MinSNR:       0,
		MaxSNR:       0,
		Band:         50,
		ChannelWidth: 0,
		EstThruput:   0,
		QBSSUtil:     25,
		QBSSStaCt:    0,
		PHYType:      15,
	}
	preferences := Preferences{
		Interface: "wlan0",
	}
	return Config{
		Thresholds:   thresholds,
		ScoreWeights: scoreWeights,
		Timing:       timing,
		Preferences:  preferences,
	}
}

func encodeConfig(f *os.File, cfg Config) error {
	err := toml.NewEncoder(f).Encode(cfg)
	if err != nil {
		return fmt.Errorf("encode error: %w", err)
	}
	return nil
}

func parseConfig(configPath string) (*Config, error) {
	var cfg Config
	_, err := toml.DecodeFile(configPath, &cfg)
	if err != nil {
		return nil, fmt.Errorf("toml.DecodeFile: %w", err)
	}
	return &cfg, nil
}
