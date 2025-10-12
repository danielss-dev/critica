package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	DiffModeAll      = "all"
	DiffModeStaged   = "staged"
	DiffModeUnstaged = "unstaged"
)

type Config struct {
	Interactive *bool  `json:"interactive,omitempty"`
	Unified     *bool  `json:"unified,omitempty"`
	NoColor     *bool  `json:"no_color,omitempty"`
	DiffMode    string `json:"diff_mode,omitempty"`
}

func Load() (*Config, error) {
	path, err := DefaultPath()
	if err != nil {
		return &Config{}, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &Config{}, nil
		}
		return &Config{}, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return &Config{}, fmt.Errorf("parse config: %w", err)
	}

	if err := cfg.normalize(); err != nil {
		return &Config{}, err
	}

	return &cfg, nil
}

func DefaultPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "critica", "config.json"), nil
}

func (c *Config) normalize() error {
	mode := strings.ToLower(strings.TrimSpace(c.DiffMode))
	if mode == "" {
		c.DiffMode = ""
		return nil
	}

	switch mode {
	case DiffModeAll, DiffModeStaged, DiffModeUnstaged:
		c.DiffMode = mode
		return nil
	default:
		return fmt.Errorf("invalid diff_mode %q", c.DiffMode)
	}
}
