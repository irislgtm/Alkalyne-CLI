package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/alkalyne/alkalyne/internal/models"
)

func Load(path string) (*models.Config, error) {
	cfg := models.DefaultConfig()
	if path == "" {
		path = defaultPath()
	}
	cfg.DataDir = filepath.Dir(path)

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(cfg.DataDir, 0700); err != nil {
				return nil, fmt.Errorf("config: create dir: %w", err)
			}
			if err := Save(path, cfg); err != nil {
				return nil, fmt.Errorf("config: save defaults: %w", err)
			}
			return cfg, nil
		}
		return nil, fmt.Errorf("config: read: %w", err)
	}

	if err := toml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("config: parse: %w", err)
	}
	return cfg, nil
}

func Save(path string, cfg *models.Config) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("config: open: %w", err)
	}
	defer func() { _ = f.Close() }()

	enc := toml.NewEncoder(f)
	if err := enc.Encode(cfg); err != nil {
		return fmt.Errorf("config: encode: %w", err)
	}
	return nil
}

func defaultPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "~/.alkalyne/config.toml"
	}
	return filepath.Join(home, ".alkalyne", "config.toml")
}

func DataDir(cfg *models.Config) string {
	dir := cfg.DataDir
	if len(dir) > 0 && dir[0] == '~' {
		home, err := os.UserHomeDir()
		if err == nil {
			dir = filepath.Join(home, dir[1:])
		}
	}
	return dir
}
