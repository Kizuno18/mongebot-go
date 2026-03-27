// Package config - environment variable overrides and .env file loading.
package config

import (
	"bufio"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"
)

// LoadDotEnv reads a .env file and sets environment variables (does not override existing).
func LoadDotEnv(path string) error {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // .env is optional
		}
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip comments and empty lines
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, value, found := strings.Cut(line, "=")
		if !found {
			continue
		}

		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)

		// Remove surrounding quotes
		if len(value) >= 2 && ((value[0] == '"' && value[len(value)-1] == '"') || (value[0] == '\'' && value[len(value)-1] == '\'')) {
			value = value[1 : len(value)-1]
		}

		// Don't override existing env vars
		if _, exists := os.LookupEnv(key); !exists {
			os.Setenv(key, value)
		}
	}

	return scanner.Err()
}

// ApplyEnvOverrides applies environment variable overrides to the config.
// Environment variables take precedence over file config.
func ApplyEnvOverrides(cfg *AppConfig, logger *slog.Logger) {
	overrides := []struct {
		envKey string
		apply  func(string)
		desc   string
	}{
		{"MONGEBOT_MODE", nil, "Run mode (sidecar/headless)"},
		{"MONGEBOT_API_PORT", func(v string) {
			if port, err := strconv.Atoi(v); err == nil {
				cfg.API.Port = port
			}
		}, "API server port"},
		{"MONGEBOT_API_HOST", func(v string) {
			cfg.API.Host = v
		}, "API server host"},
		{"MONGEBOT_LOG_LEVEL", func(v string) {
			cfg.Logging.Level = v
		}, "Log level"},
		{"MONGEBOT_LOG_FILE", func(v string) {
			cfg.Logging.File = v
		}, "Log file path"},
		{"MONGEBOT_MAX_WORKERS", func(v string) {
			if n, err := strconv.Atoi(v); err == nil && n > 0 {
				cfg.Engine.MaxWorkers = n
			}
		}, "Max concurrent viewers"},
		{"MONGEBOT_PROXY_TIMEOUT", func(v string) {
			if d, err := ParseDurationString(v); err == nil {
				cfg.Engine.ProxyTimeout = Duration{d}
			}
		}, "Proxy connection timeout"},
		{"MONGEBOT_CHANNEL", nil, "Default channel (headless mode)"},
		{"MONGEBOT_PLATFORM", nil, "Default platform"},
		{"MONGEBOT_ENABLE_ADS", func(v string) {
			cfg.Engine.Features.Ads = v == "true" || v == "1"
		}, "Enable ad watching"},
		{"MONGEBOT_ENABLE_CHAT", func(v string) {
			cfg.Engine.Features.Chat = v == "true" || v == "1"
		}, "Enable IRC chat"},
		{"MONGEBOT_ENABLE_PUBSUB", func(v string) {
			cfg.Engine.Features.PubSub = v == "true" || v == "1"
		}, "Enable PubSub"},
	}

	applied := 0
	for _, o := range overrides {
		val, exists := os.LookupEnv(o.envKey)
		if !exists || o.apply == nil {
			continue
		}
		o.apply(val)
		applied++
		logger.Debug("env override applied", "key", o.envKey, "value", val)
	}

	if applied > 0 {
		logger.Info("environment overrides applied", "count", applied)
	}
}

// ParseDurationString wraps time.ParseDuration for config use.
func ParseDurationString(s string) (time.Duration, error) {
	return time.ParseDuration(s)
}

// GetEnvOrDefault returns the environment variable value or a default.
func GetEnvOrDefault(key, defaultValue string) string {
	if val, exists := os.LookupEnv(key); exists {
		return val
	}
	return defaultValue
}

// GetEnvIntOrDefault returns the environment variable as int or a default.
func GetEnvIntOrDefault(key string, defaultValue int) int {
	if val, exists := os.LookupEnv(key); exists {
		if n, err := strconv.Atoi(val); err == nil {
			return n
		}
	}
	return defaultValue
}

// GetEnvBoolOrDefault returns the environment variable as bool or a default.
func GetEnvBoolOrDefault(key string, defaultValue bool) bool {
	if val, exists := os.LookupEnv(key); exists {
		return val == "true" || val == "1" || val == "yes"
	}
	return defaultValue
}
