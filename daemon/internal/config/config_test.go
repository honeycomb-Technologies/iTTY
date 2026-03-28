package config

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestLoadDefaultReturnsDefaultsWhenMissing(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	cfg, err := LoadDefault()
	if err != nil {
		t.Fatalf("LoadDefault() error = %v", err)
	}

	want := DefaultConfig()
	if !reflect.DeepEqual(cfg, want) {
		t.Fatalf("LoadDefault() = %#v, want %#v", cfg, want)
	}
}

func TestSaveDefaultRoundTrip(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	want := &Config{
		ListenAddr:     "127.0.0.1:9090",
		TmuxPath:       "/opt/homebrew/bin/tmux",
		AutoWrap:       false,
		TailscaleServe: false,
		APNsKeyPath:    "/tmp/key.p8",
		APNsKeyID:      "KEY123",
		APNsTeamID:     "TEAM123",
	}

	if err := SaveDefault(want); err != nil {
		t.Fatalf("SaveDefault() error = %v", err)
	}

	got, err := LoadDefault()
	if err != nil {
		t.Fatalf("LoadDefault() error = %v", err)
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("round trip = %#v, want %#v", got, want)
	}
}

func TestSaveRejectsInvalidConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")

	err := Save(path, &Config{})
	if err == nil {
		t.Fatal("Save() error = nil, want validation error")
	}
}
