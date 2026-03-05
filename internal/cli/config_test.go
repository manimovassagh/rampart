package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveAndLoadConfig(t *testing.T) {
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	t.Setenv("HOME", tmpDir)
	defer func() { _ = os.Setenv("HOME", origHome) }()

	cfg := &Config{
		Issuer:       "http://localhost:8080",
		AccessToken:  "test-token",
		RefreshToken: "test-refresh",
	}

	if err := SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}

	// Verify file permissions
	path := filepath.Join(tmpDir, configDirName, configFileName)
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat config: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0600 {
		t.Errorf("config file permissions = %o, want 0600", perm)
	}

	loaded, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}

	if loaded.Issuer != cfg.Issuer {
		t.Errorf("Issuer = %q, want %q", loaded.Issuer, cfg.Issuer)
	}
	if loaded.AccessToken != cfg.AccessToken {
		t.Errorf("AccessToken = %q, want %q", loaded.AccessToken, cfg.AccessToken)
	}
	if loaded.RefreshToken != cfg.RefreshToken {
		t.Errorf("RefreshToken = %q, want %q", loaded.RefreshToken, cfg.RefreshToken)
	}
}

func TestLoadConfigMissing(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig on missing file: %v", err)
	}
	if cfg.Issuer != "" {
		t.Errorf("expected empty config, got issuer %q", cfg.Issuer)
	}
}
