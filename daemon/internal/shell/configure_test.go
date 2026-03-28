package shell

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfigureIsIdempotentAndCreatesParentDir(t *testing.T) {
	rcPath := filepath.Join(t.TempDir(), ".config", "fish", "config.fish")
	info := &ShellInfo{
		Type:   Fish,
		Path:   "/usr/bin/fish",
		RCFile: rcPath,
	}

	if err := Configure(info); err != nil {
		t.Fatalf("Configure() first call error = %v", err)
	}
	if err := Configure(info); err != nil {
		t.Fatalf("Configure() second call error = %v", err)
	}

	content, err := os.ReadFile(rcPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	if count := strings.Count(string(content), markerStart); count != 1 {
		t.Fatalf("markerStart count = %d, want 1", count)
	}
	if !IsConfigured(info) {
		t.Fatal("IsConfigured() = false, want true")
	}
}

func TestUnconfigureRemovesManagedBlockAndKeepsUserContent(t *testing.T) {
	rcPath := filepath.Join(t.TempDir(), ".bashrc")
	info := &ShellInfo{
		Type:   Bash,
		Path:   "/bin/bash",
		RCFile: rcPath,
	}

	original := "export PATH=/usr/local/bin:$PATH\n"
	if err := os.WriteFile(rcPath, []byte(original), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if err := Configure(info); err != nil {
		t.Fatalf("Configure() error = %v", err)
	}
	if err := Unconfigure(info); err != nil {
		t.Fatalf("Unconfigure() error = %v", err)
	}

	content, err := os.ReadFile(rcPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	if got := string(content); got != original {
		t.Fatalf("final rc file = %q, want %q", got, original)
	}
	if IsConfigured(info) {
		t.Fatal("IsConfigured() = true, want false")
	}
}
