package config

import (
	"errors"
	"net/url"
	"os"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

const defaultServerURL = "localhost:8080"

type Config struct {
	ServerURL string `toml:"server_url"`
}

type NormalizedConfig struct {
	RawHost  string
	RESTBase string
	WSBase   string
}

func Load(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Config{ServerURL: defaultServerURL}, nil
		}
		return Config{}, err
	}

	var cfg Config
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}
	if strings.TrimSpace(cfg.ServerURL) == "" {
		cfg.ServerURL = defaultServerURL
	}
	return cfg, nil
}

func (c Config) Normalized() NormalizedConfig {
	raw := strings.TrimSpace(c.ServerURL)
	if raw == "" {
		raw = defaultServerURL
	}
	raw = strings.TrimRight(raw, "/")

	if !strings.Contains(raw, "://") {
		return NormalizedConfig{
			RawHost:  raw,
			RESTBase: "http://" + raw,
			WSBase:   "ws://" + raw,
		}
	}

	parsed, err := url.Parse(raw)
	if err != nil || parsed.Host == "" {
		return NormalizedConfig{
			RawHost:  raw,
			RESTBase: raw,
			WSBase:   raw,
		}
	}

	wsScheme := "ws"
	if parsed.Scheme == "https" {
		wsScheme = "wss"
	}
	ws := *parsed
	ws.Scheme = wsScheme

	return NormalizedConfig{
		RawHost:  parsed.Host,
		RESTBase: raw,
		WSBase:   ws.String(),
	}
}
