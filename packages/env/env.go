package env

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// Config holds all environment variables used by the application.
type Config struct {
	ProxyURL string
}

// Load reads environment variables and returns a Config.
// It first attempts to load from a .env file at the given path (if non-empty),
// then falls back to OS environment variables.
func Load(envFile string) (*Config, error) {
	if envFile != "" {
		if err := loadFile(envFile); err != nil {
			return nil, fmt.Errorf("failed to load env file %s: %w", envFile, err)
		}
	}

	cfg := &Config{
		ProxyURL: os.Getenv("PROXY_URL"),
	}

	if cfg.ProxyURL == "" {
		return nil, fmt.Errorf("PROXY_URL is required")
	}

	return cfg, nil
}

// loadFile reads a .env file and sets each KEY=VALUE pair as an environment variable.
func loadFile(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}

		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		value = strings.Trim(value, `"'`)

		os.Setenv(key, value)
	}

	return scanner.Err()
}
