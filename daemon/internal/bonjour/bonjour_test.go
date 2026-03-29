package bonjour

import (
	"context"
	"errors"
	"os/exec"
	"testing"
)

func TestParsePort(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		listenAddr string
		want       int
		wantErr    bool
	}{
		{name: "wildcard host", listenAddr: ":3420", want: 3420},
		{name: "explicit host", listenAddr: "127.0.0.1:8080", want: 8080},
		{name: "invalid address", listenAddr: "3420", wantErr: true},
		{name: "invalid port", listenAddr: ":0", wantErr: true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := parsePort(tt.listenAddr)
			if tt.wantErr {
				if err == nil {
					t.Fatal("parsePort() error = nil, want failure")
				}
				return
			}

			if err != nil {
				t.Fatalf("parsePort() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("parsePort() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestAdvertisementCommandPrefersDNSSD(t *testing.T) {
	t.Parallel()

	cmd, err := advertisementCommand(context.Background(), "itty-host", 3420, func(name string) (string, error) {
		switch name {
		case "dns-sd":
			return "/usr/bin/dns-sd", nil
		case "avahi-publish":
			return "/usr/bin/avahi-publish", nil
		default:
			return "", exec.ErrNotFound
		}
	})
	if err != nil {
		t.Fatalf("advertisementCommand() error = %v", err)
	}
	if cmd == nil {
		t.Fatal("advertisementCommand() = nil, want dns-sd command")
	}
	if got := cmd.Path; got != "/usr/bin/dns-sd" {
		t.Fatalf("cmd.Path = %q, want %q", got, "/usr/bin/dns-sd")
	}
}

func TestAdvertisementCommandFallsBackToAvahi(t *testing.T) {
	t.Parallel()

	cmd, err := advertisementCommand(context.Background(), "itty-host", 3420, func(name string) (string, error) {
		if name == "avahi-publish" {
			return "/usr/bin/avahi-publish", nil
		}
		return "", exec.ErrNotFound
	})
	if err != nil {
		t.Fatalf("advertisementCommand() error = %v", err)
	}
	if cmd == nil {
		t.Fatal("advertisementCommand() = nil, want avahi command")
	}
	if got := cmd.Path; got != "/usr/bin/avahi-publish" {
		t.Fatalf("cmd.Path = %q, want %q", got, "/usr/bin/avahi-publish")
	}
}

func TestAdvertisementCommandReturnsNilWhenUnavailable(t *testing.T) {
	t.Parallel()

	cmd, err := advertisementCommand(context.Background(), "itty-host", 3420, func(string) (string, error) {
		return "", exec.ErrNotFound
	})
	if err != nil {
		t.Fatalf("advertisementCommand() error = %v", err)
	}
	if cmd != nil {
		t.Fatalf("advertisementCommand() = %#v, want nil", cmd)
	}
}

func TestAdvertisementCommandPropagatesUnexpectedLookPathError(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("lookup failed")
	_, err := advertisementCommand(context.Background(), "itty-host", 3420, func(string) (string, error) {
		return "", wantErr
	})
	if err == nil {
		t.Fatal("advertisementCommand() error = nil, want failure")
	}
	if !errors.Is(err, wantErr) {
		t.Fatalf("advertisementCommand() error = %v, want %v", err, wantErr)
	}
}
