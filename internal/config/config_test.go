package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/alkalyne/alkalyne/internal/config"
)

func TestLoadCreatesDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.DataDir == "" {
		t.Error("expected non-empty DataDir")
	}
	if len(cfg.ListenAddrs) == 0 {
		t.Error("expected at least one listen addr")
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("config file should exist after Load")
	}
}

func TestLoadAndSave(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	cfg.Nickname = "testuser"
	if err := config.Save(path, cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	reloaded, err := config.Load(path)
	if err != nil {
		t.Fatalf("Reload: %v", err)
	}

	if reloaded.Nickname != "testuser" {
		t.Errorf("expected nickname 'testuser', got %q", reloaded.Nickname)
	}
}
