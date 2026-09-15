package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const (
	ProviderOllama = "ollama"
	ProviderOpenAI = "openai"

	DefaultProvider = ProviderOllama
	DefaultModel    = "deepseek-v4-pro:cloud"
	DefaultHost     = "http://localhost:11434"
)

type OllamaConfig struct {
	Model string `yaml:"model"`
	Host  string `yaml:"host"`
}

type OpenAIConfig struct {
	Model   string `yaml:"model"`
	BaseURL string `yaml:"base_url"`
	APIKey  string `yaml:"api_key"`
}

type Config struct {
	Provider string       `yaml:"provider"`
	Ollama   OllamaConfig `yaml:"ollama"`
	OpenAI   OpenAIConfig `yaml:"openai"`
}

func defaultConfig() Config {
	return Config{
		Provider: DefaultProvider,
		Ollama: OllamaConfig{
			Model: DefaultModel,
			Host:  DefaultHost,
		},
		OpenAI: OpenAIConfig{
			Model:   "",
			BaseURL: "",
			APIKey:  "",
		},
	}
}

func Path() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "dict-cli", "config.yml"), nil
}

// Load reads the config file at ~/.config/dict-cli/config.yml, creating it
// with default values if it doesn't exist yet.
func Load() (Config, error) {
	path, err := Path()
	if err != nil {
		return defaultConfig(), err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			cfg := defaultConfig()
			if writeErr := save(path, cfg); writeErr != nil {
				return cfg, writeErr
			}
			return cfg, nil
		}
		return defaultConfig(), err
	}

	cfg := defaultConfig()
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return defaultConfig(), err
	}

	if cfg.Provider == "" {
		cfg.Provider = DefaultProvider
	}
	if cfg.Ollama.Model == "" {
		cfg.Ollama.Model = DefaultModel
	}
	if cfg.Ollama.Host == "" {
		cfg.Ollama.Host = DefaultHost
	}

	return cfg, nil
}

func save(path string, cfg Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0o644)
}

// Validate checks that the config has the fields required for the selected provider.
func (c Config) Validate() error {
	switch c.Provider {
	case ProviderOllama:
		return nil
	case ProviderOpenAI:
		if c.OpenAI.APIKey == "" {
			return fmt.Errorf("openai.api_key is required when provider is %q (set it in the config file or OPENAI_API_KEY env var)", ProviderOpenAI)
		}
		if c.OpenAI.Model == "" {
			return fmt.Errorf("openai.model is required when provider is %q", ProviderOpenAI)
		}
		return nil
	default:
		return fmt.Errorf("unknown provider %q (expected %q or %q)", c.Provider, ProviderOllama, ProviderOpenAI)
	}
}
