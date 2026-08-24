package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadRefreshInterval(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, []byte(`{"refresh_interval":"30m","channels":[]}`), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got := cfg.GetRefreshInterval(); got != 30*time.Minute {
		t.Fatalf("GetRefreshInterval() = %v, want %v", got, 30*time.Minute)
	}
}

func TestLoadRejectsNonPositiveRefreshInterval(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(`{"refresh_interval":"0s"}`), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("Load() error = nil, want error for a non-positive refresh interval")
	}
}
