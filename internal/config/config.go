package config

import (
	"fmt"
	"strings"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

type Config struct {
	Server    ServerConfig              `koanf:"server"`
	Providers map[string]ProviderConfig `koanf:"providers"`
}

type ServerConfig struct {
	Host        string `koanf:"host"`
	ProxyPort   int    `koanf:"proxy_port"`
	MetricsPort int    `koanf:"metrics_port"`
}

type ProviderConfig struct {
	Upstream string `koanf:"upstream"`
	Enabled  bool   `koanf:"enabled"`
}

func Defaults() *Config {
	return &Config{
		Server: ServerConfig{
			Host:        "127.0.0.1",
			ProxyPort:   8080,
			MetricsPort: 9090,
		},
		Providers: map[string]ProviderConfig{
			"openai": {
				Upstream: "https://api.openai.com",
				Enabled:  true,
			},
			"anthropic": {
				Upstream: "https://api.anthropic.com",
				Enabled:  true,
			},
			"google": {
				Upstream: "https://generativelanguage.googleapis.com",
				Enabled:  false,
			},
		},
	}
}

func Load(configPath string) (*Config, error) {
	k := koanf.New(".")
	cfg := Defaults()

	// Load from YAML config file if provided.
	if configPath != "" {
		if err := k.Load(file.Provider(configPath), yaml.Parser()); err != nil {
			return nil, fmt.Errorf("loading config file %s: %w", configPath, err)
		}
	}

	// Load from environment variables (LLMWATCHER_SERVER_HOST, etc.).
	if err := k.Load(env.Provider("LLMWATCHER_", ".", func(s string) string {
		return strings.ToLower(strings.ReplaceAll(
			strings.TrimPrefix(s, "LLMWATCHER_"), "_", "."))
	}), nil); err != nil {
		return nil, fmt.Errorf("loading env vars: %w", err)
	}

	if err := k.Unmarshal("", cfg); err != nil {
		return nil, fmt.Errorf("unmarshalling config: %w", err)
	}

	return cfg, nil
}
