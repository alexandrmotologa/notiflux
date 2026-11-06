package config

import (
	"os"
	"testing"
)

func TestLoadConfig_Defaults(t *testing.T) {
	cfg, err := LoadConfig([]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Port != 8085 {
		t.Errorf("expected port 8085, got %d", cfg.Port)
	}
	if cfg.DBPath != "notiflux.db" {
		t.Errorf("expected db path notiflux.db, got %s", cfg.DBPath)
	}
	if len(cfg.AllowedOrigins) != 1 || cfg.AllowedOrigins[0] != "*" {
		t.Errorf("expected allowed origins ['*'], got %v", cfg.AllowedOrigins)
	}
}

func TestLoadConfig_Flags(t *testing.T) {
	args := []string{
		"--port", "9090",
		"--db", "custom.db",
		"--api-key", "my-secret",
		"--allowed-origins", "https://app.example.com,https://admin.example.com",
		"--smtp-host", "smtp.postmarkapp.com",
		"--smtp-port", "2525",
	}

	cfg, err := LoadConfig(args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Port != 9090 {
		t.Errorf("expected port 9090, got %d", cfg.Port)
	}
	if cfg.DBPath != "custom.db" {
		t.Errorf("expected db custom.db, got %s", cfg.DBPath)
	}
	if cfg.APIKey != "my-secret" {
		t.Errorf("expected api-key my-secret, got %s", cfg.APIKey)
	}
	if len(cfg.AllowedOrigins) != 2 {
		t.Errorf("expected 2 allowed origins, got %d", len(cfg.AllowedOrigins))
	}
	if cfg.SMTPHost != "smtp.postmarkapp.com" {
		t.Errorf("expected smtp host smtp.postmarkapp.com, got %s", cfg.SMTPHost)
	}
	if cfg.SMTPPort != 2525 {
		t.Errorf("expected smtp port 2525, got %d", cfg.SMTPPort)
	}
}

func TestLoadConfig_EnvVars(t *testing.T) {
	os.Setenv("NOTIFLUX_PORT", "7070")
	os.Setenv("NOTIFLUX_DB_PATH", "env.db")
	os.Setenv("NOTIFLUX_API_KEY", "env-key")
	defer func() {
		os.Unsetenv("NOTIFLUX_PORT")
		os.Unsetenv("NOTIFLUX_DB_PATH")
		os.Unsetenv("NOTIFLUX_API_KEY")
	}()

	cfg, err := LoadConfig([]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Port != 7070 {
		t.Errorf("expected port 7070 from env, got %d", cfg.Port)
	}
	if cfg.DBPath != "env.db" {
		t.Errorf("expected db env.db from env, got %s", cfg.DBPath)
	}
	if cfg.APIKey != "env-key" {
		t.Errorf("expected api key env-key from env, got %s", cfg.APIKey)
	}
}
