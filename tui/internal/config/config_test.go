package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"hive-tui/internal/config"
)

func TestNormalizeServerURLDefaultsToLocalhost(t *testing.T) {
	cfg := config.Config{ServerURL: ""}

	got := cfg.Normalized()

	if got.RawHost != "localhost:8080" {
		t.Fatalf("RawHost = %q", got.RawHost)
	}
	if got.RESTBase != "http://localhost:8080" {
		t.Fatalf("RESTBase = %q", got.RESTBase)
	}
	if got.WSBase != "ws://localhost:8080" {
		t.Fatalf("WSBase = %q", got.WSBase)
	}
}

func TestNormalizeServerURLTrimsTrailingSlashAndUsesWSSForHTTPS(t *testing.T) {
	cfg := config.Config{ServerURL: "https://chat.example.com/"}

	got := cfg.Normalized()

	if got.RawHost != "chat.example.com" {
		t.Fatalf("RawHost = %q", got.RawHost)
	}
	if got.RESTBase != "https://chat.example.com" {
		t.Fatalf("RESTBase = %q", got.RESTBase)
	}
	if got.WSBase != "wss://chat.example.com" {
		t.Fatalf("WSBase = %q", got.WSBase)
	}
}

func TestLoadReadsTOMLConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(path, []byte("server_url = \"127.0.0.1:9090\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}

	if cfg.ServerURL != "127.0.0.1:9090" {
		t.Fatalf("ServerURL = %q", cfg.ServerURL)
	}
}

func TestLoadMissingConfigUsesDefault(t *testing.T) {
	cfg, err := config.Load(filepath.Join(t.TempDir(), "missing.toml"))
	if err != nil {
		t.Fatal(err)
	}

	if cfg.Normalized().RESTBase != "http://localhost:8080" {
		t.Fatalf("RESTBase = %q", cfg.Normalized().RESTBase)
	}
}
