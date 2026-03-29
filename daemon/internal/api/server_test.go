package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/honeycomb-Technologies/iTTY/daemon/internal/apns"
	"github.com/honeycomb-Technologies/iTTY/daemon/internal/config"
	"github.com/honeycomb-Technologies/iTTY/daemon/internal/platform"
	"github.com/honeycomb-Technologies/iTTY/daemon/internal/shell"
	"github.com/honeycomb-Technologies/iTTY/daemon/internal/tailscale"
	"github.com/honeycomb-Technologies/iTTY/daemon/internal/tmux"
)

type fakeTmux struct {
	installed     bool
	version       string
	sessions      []tmux.Session
	session       *tmux.SessionDetail
	sessionErr    error
	newSession    *tmux.SessionDetail
	newSessionErr error
	content       string
	contentErr    error
}

func (f fakeTmux) IsInstalled(context.Context) bool {
	return f.installed
}

func (f fakeTmux) Version(context.Context) (string, error) {
	return f.version, nil
}

func (f fakeTmux) ListSessions(context.Context) ([]tmux.Session, error) {
	return f.sessions, nil
}

func (f fakeTmux) GetSession(context.Context, string) (*tmux.SessionDetail, error) {
	return f.session, f.sessionErr
}

func (f fakeTmux) NewSession(_ context.Context, name string) (*tmux.SessionDetail, error) {
	if f.newSessionErr != nil {
		return nil, f.newSessionErr
	}
	if f.newSession != nil {
		return f.newSession, nil
	}
	return &tmux.SessionDetail{Session: tmux.Session{Name: name, Windows: 1}}, nil
}

func (f fakeTmux) CaptureSessionDefaultPane(context.Context, string) (string, error) {
	return f.content, f.contentErr
}

type fakeWindows struct {
	windows []platform.TerminalWindow
	err     error
}

func (f fakeWindows) ListWindows() ([]platform.TerminalWindow, error) {
	return f.windows, f.err
}

type fakeShell struct {
	info             *shell.ShellInfo
	detectErr        error
	configureErr     error
	unconfigureErr   error
	configureCalls   int
	unconfigureCalls int
}

func (f *fakeShell) Detect() (*shell.ShellInfo, error) {
	if f.detectErr != nil {
		return nil, f.detectErr
	}
	return f.info, nil
}

func (f *fakeShell) Configure(*shell.ShellInfo) error {
	f.configureCalls++
	return f.configureErr
}

func (f *fakeShell) Unconfigure(*shell.ShellInfo) error {
	f.unconfigureCalls++
	return f.unconfigureErr
}

type fakeConfigStore struct {
	saved []*config.Config
	err   error
}

func (f *fakeConfigStore) Save(cfg *config.Config) error {
	if f.err != nil {
		return f.err
	}

	copy := *cfg
	f.saved = append(f.saved, &copy)
	return nil
}

func TestGetConfigUsesExplicitJSONShape(t *testing.T) {
	server := NewServerWithDeps(
		fakeTmux{installed: true, version: "3.6a"},
		&config.Config{ListenAddr: ":3420", TmuxPath: "tmux", AutoWrap: true, TailscaleServe: true},
		nil,
		fakeWindows{},
		&fakeShell{info: &shell.ShellInfo{}},
		&fakeConfigStore{},
	)

	req := httptest.NewRequest(http.MethodGet, "/config", nil)
	rec := httptest.NewRecorder()
	server.mux.ServeHTTP(rec, req)

	body := rec.Body.String()
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if !strings.Contains(body, `"listenAddr"`) || strings.Contains(body, `"ListenAddr"`) {
		t.Fatalf("unexpected config JSON body: %s", body)
	}
}

func TestSetAutoWrapPersistsAndReturnsConfig(t *testing.T) {
	store := &fakeConfigStore{}
	sh := &fakeShell{info: &shell.ShellInfo{Type: shell.Bash, RCFile: "/tmp/.bashrc"}}
	cfg := &config.Config{ListenAddr: ":3420", TmuxPath: "tmux", AutoWrap: false, TailscaleServe: true}
	server := NewServerWithDeps(fakeTmux{}, cfg, nil, fakeWindows{}, sh, store)

	req := httptest.NewRequest(http.MethodPut, "/config/auto", strings.NewReader(`{"enabled":true}`))
	rec := httptest.NewRecorder()
	server.mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if sh.configureCalls != 1 || sh.unconfigureCalls != 0 {
		t.Fatalf("configureCalls=%d unconfigureCalls=%d, want 1 and 0", sh.configureCalls, sh.unconfigureCalls)
	}
	if len(store.saved) != 1 || !store.saved[0].AutoWrap {
		t.Fatalf("saved config = %#v, want AutoWrap=true", store.saved)
	}
	if !cfg.AutoWrap {
		t.Fatalf("cfg.AutoWrap = %v, want true", cfg.AutoWrap)
	}
}

func TestSetAutoWrapRejectsBadRequest(t *testing.T) {
	server := NewServerWithDeps(
		fakeTmux{},
		&config.Config{ListenAddr: ":3420", TmuxPath: "tmux"},
		nil,
		fakeWindows{},
		&fakeShell{info: &shell.ShellInfo{}},
		&fakeConfigStore{},
	)

	req := httptest.NewRequest(http.MethodPut, "/config/auto", strings.NewReader(`{"wrong":true}`))
	rec := httptest.NewRecorder()
	server.mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestGetWindowsReturnsDiscovererOutput(t *testing.T) {
	server := NewServerWithDeps(
		fakeTmux{},
		&config.Config{ListenAddr: ":3420", TmuxPath: "tmux"},
		nil,
		fakeWindows{windows: []platform.TerminalWindow{{ID: "1", Title: "ghostty", App: "ghostty", Focused: true}}},
		&fakeShell{info: &shell.ShellInfo{}},
		&fakeConfigStore{},
	)

	req := httptest.NewRequest(http.MethodGet, "/windows", nil)
	rec := httptest.NewRecorder()
	server.mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var windows []platform.TerminalWindow
	if err := json.Unmarshal(rec.Body.Bytes(), &windows); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if len(windows) != 1 || windows[0].ID != "1" {
		t.Fatalf("windows = %#v, want one window", windows)
	}
}

func TestGetSessionMapsNotFoundTo404(t *testing.T) {
	server := NewServerWithDeps(
		fakeTmux{sessionErr: fmt.Errorf("%w: missing", tmux.ErrSessionNotFound)},
		&config.Config{ListenAddr: ":3420", TmuxPath: "tmux"},
		nil,
		fakeWindows{},
		&fakeShell{info: &shell.ShellInfo{}},
		&fakeConfigStore{},
	)

	req := httptest.NewRequest(http.MethodGet, "/sessions/missing", nil)
	rec := httptest.NewRecorder()
	server.mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestStartReturnsBindError(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen() error = %v", err)
	}
	defer listener.Close()

	server := NewServerWithDeps(
		fakeTmux{},
		&config.Config{ListenAddr: listener.Addr().String(), TmuxPath: "tmux"},
		nil,
		fakeWindows{},
		&fakeShell{info: &shell.ShellInfo{}},
		&fakeConfigStore{},
	)

	err = server.Start()
	if err == nil {
		t.Fatal("Start() error = nil, want bind error")
	}
}

func TestShutdownBeforeStartIsSafe(t *testing.T) {
	server := NewServerWithDeps(
		fakeTmux{},
		&config.Config{ListenAddr: ":3420", TmuxPath: "tmux"},
		nil,
		fakeWindows{},
		&fakeShell{info: &shell.ShellInfo{}},
		&fakeConfigStore{},
	)

	if err := server.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}
}

func TestSetAutoWrapReturnsStoreError(t *testing.T) {
	store := &fakeConfigStore{err: errors.New("disk full")}
	server := NewServerWithDeps(
		fakeTmux{},
		&config.Config{ListenAddr: ":3420", TmuxPath: "tmux"},
		nil,
		fakeWindows{},
		&fakeShell{info: &shell.ShellInfo{}},
		store,
	)

	req := httptest.NewRequest(http.MethodPut, "/config/auto", strings.NewReader(`{"enabled":false}`))
	rec := httptest.NewRecorder()
	server.mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
}

type fakeTailscale struct {
	peers []tailscale.Peer
	err   error
}

func (f fakeTailscale) Peers(context.Context) ([]tailscale.Peer, error) {
	return f.peers, f.err
}

func TestGetPeersSuccess(t *testing.T) {
	ts := &fakeTailscale{
		peers: []tailscale.Peer{
			{ID: "self1", Hostname: "mac", DNSName: "mac.tail.ts.net", OS: "macOS", Online: true, Self: true},
			{ID: "peer1", Hostname: "desktop", DNSName: "desktop.tail.ts.net", OS: "linux", Online: true},
		},
	}
	server := NewServerWithDeps(
		fakeTmux{},
		&config.Config{ListenAddr: ":3420", TmuxPath: "tmux"},
		ts,
		fakeWindows{},
		&fakeShell{info: &shell.ShellInfo{}},
		&fakeConfigStore{},
	)

	req := httptest.NewRequest(http.MethodGet, "/peers", nil)
	rec := httptest.NewRecorder()
	server.mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var peers []tailscale.Peer
	if err := json.Unmarshal(rec.Body.Bytes(), &peers); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if len(peers) != 2 {
		t.Fatalf("got %d peers, want 2", len(peers))
	}
}

func TestGetPeersTailscaleNil(t *testing.T) {
	server := NewServerWithDeps(
		fakeTmux{},
		&config.Config{ListenAddr: ":3420", TmuxPath: "tmux"},
		nil,
		fakeWindows{},
		&fakeShell{info: &shell.ShellInfo{}},
		&fakeConfigStore{},
	)

	req := httptest.NewRequest(http.MethodGet, "/peers", nil)
	rec := httptest.NewRecorder()
	server.mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}
}

func TestGetPeersCLIError(t *testing.T) {
	ts := &fakeTailscale{err: errors.New("tailscale status: connection refused")}
	server := NewServerWithDeps(
		fakeTmux{},
		&config.Config{ListenAddr: ":3420", TmuxPath: "tmux"},
		ts,
		fakeWindows{},
		&fakeShell{info: &shell.ShellInfo{}},
		&fakeConfigStore{},
	)

	req := httptest.NewRequest(http.MethodGet, "/peers", nil)
	rec := httptest.NewRecorder()
	server.mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadGateway)
	}
}

func TestCreateSessionSuccess(t *testing.T) {
	server := NewServerWithDeps(
		fakeTmux{},
		&config.Config{ListenAddr: ":3420", TmuxPath: "tmux"},
		nil,
		fakeWindows{},
		&fakeShell{info: &shell.ShellInfo{}},
		&fakeConfigStore{},
	)

	req := httptest.NewRequest(http.MethodPost, "/sessions", strings.NewReader(`{"name":"itty-1"}`))
	rec := httptest.NewRecorder()
	server.mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusCreated, rec.Body.String())
	}

	var detail tmux.SessionDetail
	if err := json.Unmarshal(rec.Body.Bytes(), &detail); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if detail.Name != "itty-1" {
		t.Fatalf("session name = %q, want %q", detail.Name, "itty-1")
	}
}

func TestCreateSessionMissingName(t *testing.T) {
	server := NewServerWithDeps(
		fakeTmux{},
		&config.Config{ListenAddr: ":3420", TmuxPath: "tmux"},
		nil,
		fakeWindows{},
		&fakeShell{info: &shell.ShellInfo{}},
		&fakeConfigStore{},
	)

	req := httptest.NewRequest(http.MethodPost, "/sessions", strings.NewReader(`{}`))
	rec := httptest.NewRecorder()
	server.mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestCreateSessionRejectsUnknownFields(t *testing.T) {
	server := NewServerWithDeps(
		fakeTmux{},
		&config.Config{ListenAddr: ":3420", TmuxPath: "tmux"},
		nil,
		fakeWindows{},
		&fakeShell{info: &shell.ShellInfo{}},
		&fakeConfigStore{},
	)

	req := httptest.NewRequest(http.MethodPost, "/sessions", strings.NewReader(`{"name":"itty-1","extra":true}`))
	rec := httptest.NewRecorder()
	server.mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestCreateSessionRejectsTrailingJSON(t *testing.T) {
	server := NewServerWithDeps(
		fakeTmux{},
		&config.Config{ListenAddr: ":3420", TmuxPath: "tmux"},
		nil,
		fakeWindows{},
		&fakeShell{info: &shell.ShellInfo{}},
		&fakeConfigStore{},
	)

	req := httptest.NewRequest(http.MethodPost, "/sessions", strings.NewReader(`{"name":"itty-1"}{"name":"itty-2"}`))
	rec := httptest.NewRecorder()
	server.mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestCreateSessionMapsInvalidNameTo400(t *testing.T) {
	server := NewServerWithDeps(
		fakeTmux{newSessionErr: fmt.Errorf("%w: session name may only contain letters", tmux.ErrInvalidSessionName)},
		&config.Config{ListenAddr: ":3420", TmuxPath: "tmux"},
		nil,
		fakeWindows{},
		&fakeShell{info: &shell.ShellInfo{}},
		&fakeConfigStore{},
	)

	req := httptest.NewRequest(http.MethodPost, "/sessions", strings.NewReader(`{"name":"bad/name"}`))
	rec := httptest.NewRecorder()
	server.mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d body=%s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}

func TestCreateSessionMapsDuplicateTo409(t *testing.T) {
	server := NewServerWithDeps(
		fakeTmux{newSessionErr: fmt.Errorf("%w: itty-1", tmux.ErrSessionExists)},
		&config.Config{ListenAddr: ":3420", TmuxPath: "tmux"},
		nil,
		fakeWindows{},
		&fakeShell{info: &shell.ShellInfo{}},
		&fakeConfigStore{},
	)

	req := httptest.NewRequest(http.MethodPost, "/sessions", strings.NewReader(`{"name":"itty-1"}`))
	rec := httptest.NewRecorder()
	server.mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d body=%s", rec.Code, http.StatusConflict, rec.Body.String())
	}
}

func TestCaptureContentMapsNotFoundTo404(t *testing.T) {
	server := NewServerWithDeps(
		fakeTmux{contentErr: fmt.Errorf("%w: missing", tmux.ErrSessionNotFound)},
		&config.Config{ListenAddr: ":3420", TmuxPath: "tmux"},
		nil,
		fakeWindows{},
		&fakeShell{info: &shell.ShellInfo{}},
		&fakeConfigStore{},
	)

	req := httptest.NewRequest(http.MethodGet, "/sessions/missing/content", nil)
	rec := httptest.NewRecorder()
	server.mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestGetPeersEmptyReturnsArray(t *testing.T) {
	ts := &fakeTailscale{peers: []tailscale.Peer{}}
	server := NewServerWithDeps(
		fakeTmux{},
		&config.Config{ListenAddr: ":3420", TmuxPath: "tmux"},
		ts,
		fakeWindows{},
		&fakeShell{info: &shell.ShellInfo{}},
		&fakeConfigStore{},
	)

	req := httptest.NewRequest(http.MethodGet, "/peers", nil)
	rec := httptest.NewRecorder()
	server.mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	body := strings.TrimSpace(rec.Body.String())
	if body != "[]" {
		t.Fatalf("body = %q, want %q", body, "[]")
	}
}

func TestRegisterDeviceSuccess(t *testing.T) {
	server := NewServerWithDeps(
		fakeTmux{},
		&config.Config{ListenAddr: ":3420", TmuxPath: "tmux"},
		nil,
		fakeWindows{},
		&fakeShell{info: &shell.ShellInfo{}},
		&fakeConfigStore{},
	)
	server.ConfigureAPNs(nil)

	token := strings.Repeat("a", 64)
	req := httptest.NewRequest(http.MethodPost, "/devices", strings.NewReader(`{"token":"`+token+`"}`))
	rec := httptest.NewRecorder()
	server.mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
}

func TestRegisterDeviceEmptyToken(t *testing.T) {
	server := NewServerWithDeps(
		fakeTmux{},
		&config.Config{ListenAddr: ":3420", TmuxPath: "tmux"},
		nil,
		fakeWindows{},
		&fakeShell{info: &shell.ShellInfo{}},
		&fakeConfigStore{},
	)
	server.ConfigureAPNs(nil)

	req := httptest.NewRequest(http.MethodPost, "/devices", strings.NewReader(`{"token":""}`))
	rec := httptest.NewRecorder()
	server.mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestRegisterDeviceShortToken(t *testing.T) {
	server := NewServerWithDeps(
		fakeTmux{},
		&config.Config{ListenAddr: ":3420", TmuxPath: "tmux"},
		nil,
		fakeWindows{},
		&fakeShell{info: &shell.ShellInfo{}},
		&fakeConfigStore{},
	)
	server.ConfigureAPNs(nil)

	req := httptest.NewRequest(http.MethodPost, "/devices", strings.NewReader(`{"token":"tooshort"}`))
	rec := httptest.NewRecorder()
	server.mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestRegisterDeviceStoreNil(t *testing.T) {
	server := NewServerWithDeps(
		fakeTmux{},
		&config.Config{ListenAddr: ":3420", TmuxPath: "tmux"},
		nil,
		fakeWindows{},
		&fakeShell{info: &shell.ShellInfo{}},
		&fakeConfigStore{},
	)
	// Don't call ConfigureAPNs — deviceStore is nil

	req := httptest.NewRequest(http.MethodPost, "/devices", strings.NewReader(`{"token":"abc"}`))
	rec := httptest.NewRecorder()
	server.mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}
}

func TestUnregisterDeviceSuccess(t *testing.T) {
	server := NewServerWithDeps(
		fakeTmux{},
		&config.Config{ListenAddr: ":3420", TmuxPath: "tmux"},
		nil,
		fakeWindows{},
		&fakeShell{info: &shell.ShellInfo{}},
		&fakeConfigStore{},
	)
	server.ConfigureAPNs(nil)

	// Register first
	token := strings.Repeat("x", 64)
	_ = server.deviceStore.Register(token)

	req := httptest.NewRequest(http.MethodDelete, "/devices/"+token, nil)
	rec := httptest.NewRecorder()
	server.mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	if len(server.deviceStore.All()) != 0 {
		t.Fatal("device store should be empty after unregister")
	}
}

// Ensure the unused apns import is used.
var _ = apns.NewDeviceStore
