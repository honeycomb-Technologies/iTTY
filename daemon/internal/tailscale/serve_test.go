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

func TestPeers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		script    string
		wantLen   int
		wantSelf  string
		wantPeer  string
		wantError string
	}{
		{
			name: "happy path with self and two peers",
			script: `#!/bin/sh
if [ "$1" = "status" ] && [ "$2" = "--json" ]; then
  cat <<'EOF'
{"BackendState":"Running","Self":{"ID":"self1","HostName":"my-mac","DNSName":"my-mac.tail.ts.net.","OS":"macOS","Online":true,"TailscaleIPs":["100.1.1.1"]},"Peer":{"nodekey:abc":{"ID":"peer1","HostName":"desktop","DNSName":"desktop.tail.ts.net.","OS":"linux","Online":true,"TailscaleIPs":["100.1.1.2"]},"nodekey:def":{"ID":"peer2","HostName":"laptop","DNSName":"laptop.tail.ts.net.","OS":"windows","Online":false,"TailscaleIPs":["100.1.1.3"]}}}
EOF
  exit 0
fi
exit 1
`,
			wantLen:  3,
			wantSelf: "my-mac.tail.ts.net",
			wantPeer: "desktop.tail.ts.net",
		},
		{
			name: "self only no peers",
			script: `#!/bin/sh
if [ "$1" = "status" ] && [ "$2" = "--json" ]; then
  printf '{"BackendState":"Running","Self":{"ID":"s","HostName":"solo","DNSName":"solo.tail.ts.net.","OS":"linux","Online":true,"TailscaleIPs":["100.1.1.1"]},"Peer":{}}\n'
  exit 0
fi
exit 1
`,
			wantLen:  1,
			wantSelf: "solo.tail.ts.net",
		},
		{
			name: "cli failure returns error",
			script: `#!/bin/sh
echo "tailscale not running" >&2
exit 1
`,
			wantError: "tailscale status",
		},
		{
			name: "stopped backend returns empty",
			script: `#!/bin/sh
if [ "$1" = "status" ] && [ "$2" = "--json" ]; then
  printf '{"BackendState":"Stopped","Self":{},"Peer":{}}\n'
  exit 0
fi
exit 1
`,
			wantLen: 0,
		},
		{
			name: "malformed json returns error",
			script: `#!/bin/sh
if [ "$1" = "status" ] && [ "$2" = "--json" ]; then
  printf '{"BackendState":'
  exit 0
fi
exit 1
`,
			wantError: "decoding tailscale status",
		},
		{
			name: "dns name trailing dot stripped",
			script: `#!/bin/sh
if [ "$1" = "status" ] && [ "$2" = "--json" ]; then
  printf '{"BackendState":"Running","Self":{"ID":"s","HostName":"dotty","DNSName":"dotty.tail.ts.net.","OS":"linux","Online":true,"TailscaleIPs":[]},"Peer":{}}\n'
  exit 0
fi
exit 1
`,
			wantLen:  1,
			wantSelf: "dotty.tail.ts.net",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			client := NewClient()
			client.BinPath = writeFakeTailscale(t, tt.script)

			peers, err := client.Peers(context.Background())
			if tt.wantError != "" {
				if err == nil {
					t.Fatalf("Peers() error = nil, want %q", tt.wantError)
				}
				if !strings.Contains(err.Error(), tt.wantError) {
					t.Fatalf("Peers() error = %v, want substring %q", err, tt.wantError)
				}
				return
			}
			if err != nil {
				t.Fatalf("Peers() error = %v", err)
			}
			if len(peers) != tt.wantLen {
				t.Fatalf("Peers() returned %d peers, want %d", len(peers), tt.wantLen)
			}

			if tt.wantSelf != "" {
				var found bool
				for _, p := range peers {
					if p.Self && p.DNSName == tt.wantSelf {
						found = true
						break
					}
				}
				if !found {
					t.Fatalf("Peers() missing self peer with DNSName=%q", tt.wantSelf)
				}
			}

			if tt.wantPeer != "" {
				var found bool
				for _, p := range peers {
					if !p.Self && p.DNSName == tt.wantPeer {
						found = true
						break
					}
				}
				if !found {
					t.Fatalf("Peers() missing peer with DNSName=%q", tt.wantPeer)
				}
			}
		})
	}
}

func TestPeersOfflinePeerPropagated(t *testing.T) {
	t.Parallel()

	client := NewClient()
	client.BinPath = writeFakeTailscale(t, `#!/bin/sh
if [ "$1" = "status" ] && [ "$2" = "--json" ]; then
  cat <<'EOF'
{"BackendState":"Running","Self":{"ID":"s","HostName":"me","DNSName":"me.tail.ts.net.","OS":"linux","Online":true,"TailscaleIPs":[]},"Peer":{"nodekey:a":{"ID":"p1","HostName":"offline-box","DNSName":"offline-box.tail.ts.net.","OS":"linux","Online":false,"TailscaleIPs":["100.1.1.2"]}}}
EOF
  exit 0
fi
exit 1
`)

	peers, err := client.Peers(context.Background())
	if err != nil {
		t.Fatalf("Peers() error = %v", err)
	}

	for _, p := range peers {
		if p.Hostname == "offline-box" {
			if p.Online {
				t.Fatalf("offline peer Online = true, want false")
			}
			return
		}
	}
	t.Fatal("offline-box peer not found")
}

func writeFakeTailscale(t *testing.T, script string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "tailscale")
	if err := os.WriteFile(path, []byte(strings.TrimSpace(script)+"\n"), 0o755); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	return path
}
