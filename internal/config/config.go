package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	DiffModeAll      = "all"
	DiffModeStaged   = "staged"
	DiffModeUnstaged = "unstaged"
)

const (
	DiffStyleDefault = "default"
	DiffStylePatch   = "patch"
	DiffStyleFilled  = "filled"
)

const (
	AIProviderOpenAI    = "openai"
	AIProviderAnthropic = "anthropic"
	AIProviderLocal     = "local"
)

type Config struct {
	Interactive      *bool  `json:"interactive,omitempty"`
	Unified          *bool  `json:"unified,omitempty"`
	NoColor          *bool  `json:"no_color,omitempty"`
	DiffMode         string `json:"diff_mode,omitempty"`
	DiffStyle        string `json:"diff_style,omitempty"`
	AddedTextColor   string `json:"added_text_color,omitempty"`
	DeletedTextColor string `json:"deleted_text_color,omitempty"`

	// AI Configuration
	AIEnabled       *bool  `json:"ai_enabled,omitempty"`
	AIProvider      string `json:"ai_provider,omitempty"`
	AIAPIKey        string `json:"ai_api_key,omitempty"`
	AIModel         string `json:"ai_model,omitempty"`
	AutoAnalyze     *bool  `json:"auto_analyze,omitempty"`
	ShowSuggestions *bool  `json:"show_suggestions,omitempty"`
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
	} else {
		switch mode {
		case DiffModeAll, DiffModeStaged, DiffModeUnstaged:
			c.DiffMode = mode
		default:
			return fmt.Errorf("invalid diff_mode %q", c.DiffMode)
		}
	}

	style := strings.ToLower(strings.TrimSpace(c.DiffStyle))
	switch style {
	case "", DiffStyleDefault, DiffStylePatch, DiffStyleFilled:
		c.DiffStyle = style
	default:
		return fmt.Errorf("invalid diff_style %q", c.DiffStyle)
	}

	added, err := normalizeHexColor(c.AddedTextColor)
	if err != nil {
		return fmt.Errorf("invalid added_text_color %q: %w", c.AddedTextColor, err)
	}
	c.AddedTextColor = added

	deleted, err := normalizeHexColor(c.DeletedTextColor)
	if err != nil {
		return fmt.Errorf("invalid deleted_text_color %q: %w", c.DeletedTextColor, err)
	}
	c.DeletedTextColor = deleted

	// AI configuration normalization
	if c.AIProvider != "" {
		provider := strings.ToLower(strings.TrimSpace(c.AIProvider))
		switch provider {
		case AIProviderOpenAI, AIProviderAnthropic, AIProviderLocal:
			c.AIProvider = provider
		default:
			return fmt.Errorf("invalid ai_provider %q", c.AIProvider)
		}
	} else {
		// Default to OpenAI
		c.AIProvider = AIProviderOpenAI
	}

	// Set default models based on provider
	if c.AIModel == "" {
		switch c.AIProvider {
		case AIProviderOpenAI:
			c.AIModel = "gpt-4o-mini"
		case AIProviderAnthropic:
			c.AIModel = "claude-3-5-haiku-20241022"
		}
	}

	// Check for environment variable overrides
	if envProvider := os.Getenv("CRITICA_AI_PROVIDER"); envProvider != "" {
		c.AIProvider = strings.ToLower(envProvider)
	}
	if envModel := os.Getenv("CRITICA_AI_MODEL"); envModel != "" {
		c.AIModel = envModel
	}
	if envKey := os.Getenv("CRITICA_AI_API_KEY"); envKey != "" {
		c.AIAPIKey = envKey
	}

	return nil
}

func normalizeHexColor(value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", nil
	}

	trimmed = strings.TrimPrefix(trimmed, "#")

	if len(trimmed) != 6 {
		return "", errors.New("must be a 6-digit hex color")
	}

	if _, err := strconv.ParseUint(trimmed, 16, 32); err != nil {
		return "", errors.New("must contain only hexadecimal digits")
	}

	return "#" + strings.ToLower(trimmed), nil
}
