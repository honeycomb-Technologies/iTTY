package tailscale

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestIsRunning(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		script string
		want   bool
	}{
		{
			name: "running backend state",
			script: `#!/bin/sh
if [ "$1" = "status" ] && [ "$2" = "--json" ]; then
  printf '{"BackendState":"Running"}\n'
  exit 0
fi
echo "unexpected args: $@" >&2
exit 1
`,
			want: true,
		},
		{
			name: "stopped backend state",
			script: `#!/bin/sh
if [ "$1" = "status" ] && [ "$2" = "--json" ]; then
  printf '{"BackendState":"Stopped"}\n'
  exit 0
fi
echo "unexpected args: $@" >&2
exit 1
`,
			want: false,
		},
		{
			name: "invalid json",
			script: `#!/bin/sh
if [ "$1" = "status" ] && [ "$2" = "--json" ]; then
  printf 'not-json\n'
  exit 0
fi
echo "unexpected args: $@" >&2
exit 1
`,
			want: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			client := NewClient()
			client.BinPath = writeFakeTailscale(t, tt.script)

			if got := client.IsRunning(context.Background()); got != tt.want {
				t.Fatalf("IsRunning() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHostname(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		script    string
		want      string
		wantError string
	}{
		{
			name: "trims trailing dot",
			script: `#!/bin/sh
if [ "$1" = "status" ] && [ "$2" = "--self" ] && [ "$3" = "--json" ]; then
  printf '{"DNSName":"phone-demo.tail123.ts.net."}\n'
  exit 0
fi
echo "unexpected args: $@" >&2
exit 1
`,
			want: "phone-demo.tail123.ts.net",
		},
		{
			name: "missing dns name",
			script: `#!/bin/sh
if [ "$1" = "status" ] && [ "$2" = "--self" ] && [ "$3" = "--json" ]; then
  printf '{"Hostname":"demo"}\n'
  exit 0
fi
echo "unexpected args: $@" >&2
exit 1
`,
			wantError: "could not determine tailscale hostname",
		},
		{
			name: "malformed json",
			script: `#!/bin/sh
if [ "$1" = "status" ] && [ "$2" = "--self" ] && [ "$3" = "--json" ]; then
  printf '{"DNSName":'
  exit 0
fi
echo "unexpected args: $@" >&2
exit 1
`,
			wantError: "decoding tailscale status",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			client := NewClient()
			client.BinPath = writeFakeTailscale(t, tt.script)

			got, err := client.Hostname(context.Background())
			if tt.wantError != "" {
				if err == nil {
					t.Fatalf("Hostname() error = nil, want %q", tt.wantError)
				}
				if !strings.Contains(err.Error(), tt.wantError) {
					t.Fatalf("Hostname() error = %v, want substring %q", err, tt.wantError)
				}
				return
			}
			if err != nil {
				t.Fatalf("Hostname() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("Hostname() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestServePortReturnsStderr(t *testing.T) {
	t.Parallel()

	client := NewClient()
	client.BinPath = writeFakeTailscale(t, `#!/bin/sh
if [ "$1" = "serve" ] && [ "$2" = "--bg" ] && [ "$3" = "8080" ]; then
  echo "permission denied" >&2
  exit 1
fi
echo "unexpected args: $@" >&2
exit 1
`)

	err := client.ServePort(context.Background(), 8080)
	if err == nil {
		t.Fatal("ServePort() error = nil, want failure")
	}
	if !strings.Contains(err.Error(), "permission denied") {
		t.Fatalf("ServePort() error = %v, want stderr details", err)
	}
}

func TestRunTimesOut(t *testing.T) {
	t.Parallel()

	client := NewClient()
	client.BinPath = writeFakeTailscale(t, `#!/bin/sh
sleep 1
`)
	client.CommandTimeout = 50 * time.Millisecond

	_, err := client.run(context.Background(), "status", "--json")
	if err == nil {
		t.Fatal("run() error = nil, want timeout")
	}
	if !strings.Contains(err.Error(), context.DeadlineExceeded.Error()) {
		t.Fatalf("run() error = %v, want deadline exceeded", err)
	}
}

func writeFakeTailscale(t *testing.T, script string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "tailscale")
	if err := os.WriteFile(path, []byte(strings.TrimSpace(script)+"\n"), 0o755); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	return path
}
