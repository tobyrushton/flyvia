package env

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_FromEnvFile(t *testing.T) {
	dir := t.TempDir()
	envFile := filepath.Join(dir, ".env")
	os.WriteFile(envFile, []byte("PROXY_URL=\"http://proxy.example.com:8080\"\n"), 0644)

	os.Unsetenv("PROXY_URL")

	cfg, err := Load(envFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.ProxyURL != "http://proxy.example.com:8080" {
		t.Errorf("expected http://proxy.example.com:8080, got %s", cfg.ProxyURL)
	}
}

func TestLoad_FromOSEnv(t *testing.T) {
	os.Setenv("PROXY_URL", "http://os-proxy.example.com:3128")
	defer os.Unsetenv("PROXY_URL")

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.ProxyURL != "http://os-proxy.example.com:3128" {
		t.Errorf("expected http://os-proxy.example.com:3128, got %s", cfg.ProxyURL)
	}
}

func TestLoad_MissingProxyURL(t *testing.T) {
	os.Unsetenv("PROXY_URL")

	_, err := Load("")
	if err == nil {
		t.Fatal("expected error for missing PROXY_URL")
	}
}

func TestLoad_EnvFileNotFound(t *testing.T) {
	os.Unsetenv("PROXY_URL")

	_, err := Load("/nonexistent/.env")
	if err == nil {
		t.Fatal("expected error for missing env file")
	}
}

func TestLoad_EnvFileWithComments(t *testing.T) {
	dir := t.TempDir()
	envFile := filepath.Join(dir, ".env")
	content := "# This is a comment\nPROXY_URL=http://proxy.example.com:8080\n\n# Another comment\n"
	os.WriteFile(envFile, []byte(content), 0644)
	os.Unsetenv("PROXY_URL")

	cfg, err := Load(envFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.ProxyURL != "http://proxy.example.com:8080" {
		t.Errorf("expected http://proxy.example.com:8080, got %s", cfg.ProxyURL)
	}
}

func TestLoad_SingleQuotedValue(t *testing.T) {
	dir := t.TempDir()
	envFile := filepath.Join(dir, ".env")
	os.WriteFile(envFile, []byte("PROXY_URL='http://proxy.example.com:8080'\n"), 0644)
	os.Unsetenv("PROXY_URL")

	cfg, err := Load(envFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.ProxyURL != "http://proxy.example.com:8080" {
		t.Errorf("expected http://proxy.example.com:8080, got %s", cfg.ProxyURL)
	}
}

func TestLoad_EnvFileOverridesOS(t *testing.T) {
	os.Setenv("PROXY_URL", "http://old-proxy.example.com:3128")
	defer os.Unsetenv("PROXY_URL")

	dir := t.TempDir()
	envFile := filepath.Join(dir, ".env")
	os.WriteFile(envFile, []byte("PROXY_URL=\"http://new-proxy.example.com:8080\"\n"), 0644)

	cfg, err := Load(envFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.ProxyURL != "http://new-proxy.example.com:8080" {
		t.Errorf("expected http://new-proxy.example.com:8080, got %s", cfg.ProxyURL)
	}
}
