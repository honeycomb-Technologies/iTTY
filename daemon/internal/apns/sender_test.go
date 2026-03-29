package apns

import (
	"strings"
	"testing"
)

func TestValidateToken(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		token     string
		wantError bool
	}{
		{name: "valid 64-char token", token: strings.Repeat("a", 64), wantError: false},
		{name: "valid 32-char token", token: strings.Repeat("b", 32), wantError: false},
		{name: "too short", token: "abc123", wantError: true},
		{name: "empty", token: "", wantError: true},
		{name: "31 chars", token: strings.Repeat("c", 31), wantError: true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateToken(tt.token)
			if tt.wantError && err == nil {
				t.Fatalf("ValidateToken(%q) = nil, want error", tt.token)
			}
			if !tt.wantError && err != nil {
				t.Fatalf("ValidateToken() = %v, want nil", err)
			}
		})
	}
}

func TestRedactToken(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		token string
		want  string
	}{
		{name: "normal 64-char token", token: strings.Repeat("abcd", 16), want: "abcdabcd…abcd"},
		{name: "12 chars exactly", token: "123456789012", want: "12345678…9012"},
		{name: "short token", token: "abc", want: "<short>"},
		{name: "empty", token: "", want: "<short>"},
		{name: "11 chars", token: "12345678901", want: "<short>"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := redactToken(tt.token)
			if got != tt.want {
				t.Fatalf("redactToken(%q) = %q, want %q", tt.token, got, tt.want)
			}
		})
	}
}

func TestNewSenderNilOnEmptyConfig(t *testing.T) {
	t.Parallel()

	sender, err := NewSender("", "", "", "com.itty.app", false)
	if err != nil {
		t.Fatalf("NewSender() error = %v", err)
	}
	if sender != nil {
		t.Fatal("NewSender() with empty config should return nil")
	}
}

func TestDeviceStoreRegisterRejectsShort(t *testing.T) {
	t.Parallel()

	store := NewDeviceStore()
	err := store.Register("short")
	if err == nil {
		t.Fatal("Register() with short token should return error")
	}

	tokens := store.All()
	if len(tokens) != 0 {
		t.Fatalf("Store has %d tokens after rejected registration, want 0", len(tokens))
	}
}

func TestDeviceStoreRegisterAndAll(t *testing.T) {
	t.Parallel()

	store := NewDeviceStore()
	token := strings.Repeat("a", 64)
	if err := store.Register(token); err != nil {
		t.Fatalf("Register() = %v", err)
	}

	tokens := store.All()
	if len(tokens) != 1 || tokens[0] != token {
		t.Fatalf("All() = %v, want [%s]", tokens, token)
	}
}

func TestDeviceStoreUnregister(t *testing.T) {
	t.Parallel()

	store := NewDeviceStore()
	token := strings.Repeat("b", 64)
	_ = store.Register(token)
	store.Unregister(token)

	if len(store.All()) != 0 {
		t.Fatal("Unregister() did not remove token")
	}
}
