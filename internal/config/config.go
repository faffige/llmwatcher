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

	// AWS is optional config for providers that use AWS Sig V4 signing (e.g. Bedrock).
	// If Region is empty, the default AWS credential chain is used for region too.
	AWS *AWSConfig `koanf:"aws"`
}

// AWSConfig holds AWS-specific settings for Bedrock.
// If AccessKeyID and SecretAccessKey are empty, the default credential chain
// is used (env vars, IAM roles, SSO, shared config, etc.).
type AWSConfig struct {
	Region          string `koanf:"region"`
	Profile         string `koanf:"profile"`
	AccessKeyID     string `koanf:"access_key_id"`
	SecretAccessKey string `koanf:"secret_access_key"`
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
			"bedrock": {
				Upstream: "https://bedrock-runtime.us-east-1.amazonaws.com",
				Enabled:  false,
				AWS: &AWSConfig{
					Region: "us-east-1",
				},
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
