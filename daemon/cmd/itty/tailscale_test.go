package main

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestListenPort(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		listen    string
		want      int
		wantError string
	}{
		{name: "wildcard address", listen: ":8080", want: 8080},
		{name: "loopback address", listen: "127.0.0.1:9090", want: 9090},
		{name: "missing port", listen: "127.0.0.1", wantError: "invalid listen address"},
		{name: "non numeric port", listen: ":abc", wantError: "invalid listen port"},
		{name: "out of range port", listen: ":70000", wantError: "invalid listen port"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := listenPort(tt.listen)
			if tt.wantError != "" {
				if err == nil {
					t.Fatalf("listenPort() error = nil, want %q", tt.wantError)
				}
				if got != 0 {
					t.Fatalf("listenPort() = %d, want 0 on error", got)
				}
				if !strings.Contains(err.Error(), tt.wantError) {
					t.Fatalf("listenPort() error = %v, want substring %q", err, tt.wantError)
				}
				return
			}
			if err != nil {
				t.Fatalf("listenPort() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("listenPort() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestMaybeConfigureTailscaleServe(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		client    fakeTailscale
		listen    string
		want      tailscaleServeResult
		wantError error
	}{
		{
			name:   "skips when tailscale not installed",
			client: fakeTailscale{},
			listen: ":8080",
			want:   tailscaleServeResult{},
		},
		{
			name:   "skips when tailscale not running",
			client: fakeTailscale{installed: true},
			listen: ":8080",
			want:   tailscaleServeResult{},
		},
		{
			name:   "configures serve and returns hostname",
			client: fakeTailscale{installed: true, running: true, hostname: "demo.ts.net"},
			listen: ":8080",
			want:   tailscaleServeResult{Enabled: true, Hostname: "demo.ts.net"},
		},
		{
			name:   "hostname failure does not roll back configured serve",
			client: fakeTailscale{installed: true, running: true, hostnameErr: errors.New("no hostname")},
			listen: ":8080",
			want:   tailscaleServeResult{Enabled: true},
		},
		{
			name:      "serve failure is returned",
			client:    fakeTailscale{installed: true, running: true, serveErr: errors.New("boom")},
			listen:    ":8080",
			wantError: errors.New("boom"),
		},
		{
			name:      "invalid listen address is returned",
			client:    fakeTailscale{installed: true, running: true},
			listen:    "8080",
			wantError: errors.New("invalid listen address"),
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := maybeConfigureTailscaleServe(context.Background(), &tt.client, tt.listen)
			if tt.wantError != nil {
				if err == nil {
					t.Fatalf("maybeConfigureTailscaleServe() error = nil, want %v", tt.wantError)
				}
				if !strings.Contains(err.Error(), tt.wantError.Error()) {
					t.Fatalf("maybeConfigureTailscaleServe() error = %v, want substring %q", err, tt.wantError)
				}
				return
			}
			if err != nil {
				t.Fatalf("maybeConfigureTailscaleServe() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("maybeConfigureTailscaleServe() = %#v, want %#v", got, tt.want)
			}
			if tt.want.Enabled && tt.client.servedPort != 8080 {
				t.Fatalf("ServePort() called with port %d, want 8080", tt.client.servedPort)
			}
		})
	}
}

type fakeTailscale struct {
	installed   bool
	running     bool
	hostname    string
	hostnameErr error
	serveErr    error
	servedPort  int
}

func (f *fakeTailscale) IsInstalled() bool {
	return f.installed
}

func (f *fakeTailscale) IsRunning(context.Context) bool {
	return f.running
}

func (f *fakeTailscale) Hostname(context.Context) (string, error) {
	if f.hostnameErr != nil {
		return "", f.hostnameErr
	}
	return f.hostname, nil
}

func (f *fakeTailscale) ServePort(_ context.Context, port int) error {
	f.servedPort = port
	return f.serveErr
}
