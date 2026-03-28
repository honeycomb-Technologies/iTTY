// iTTY Daemon — Desktop companion for the iTTY iOS terminal app.
//
// The daemon manages tmux sessions, auto-configures the user's shell
// for tmux wrapping, and exposes session information via a REST API
// accessible over the user's Tailscale network.
//
// Usage:
//
//	itty              Start the daemon (default)
//	itty start        Start the daemon
//	itty status       Show daemon status
//	itty auto on      Enable auto-tmux shell wrapping
//	itty auto off     Disable auto-tmux shell wrapping
//	itty sessions     List tmux sessions (CLI)
//	itty version      Print version
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/honeycomb-Technologies/iTTY/daemon/internal/api"
	"github.com/honeycomb-Technologies/iTTY/daemon/internal/config"
	"github.com/honeycomb-Technologies/iTTY/daemon/internal/shell"
	"github.com/honeycomb-Technologies/iTTY/daemon/internal/tmux"
)

// version is set at build time via -ldflags "-X main.version=..."
var version = "dev"

func main() {
	cmd := "start"
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}

	var err error
	switch cmd {
	case "start", "":
		err = runDaemon()
	case "status":
		err = runStatus()
	case "auto":
		if len(os.Args) < 3 {
			fmt.Println("usage: itty auto <on|off>")
			os.Exit(1)
		}
		err = runAuto(os.Args[2])
	case "sessions":
		err = runSessions()
	case "version":
		fmt.Printf("itty %s\n", version)
	case "help", "-h", "--help":
		printHelp()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", cmd)
		printHelp()
		os.Exit(1)
	}

	if err != nil {
		log.Fatal(err)
	}
}

func runDaemon() error {
	cfg, err := config.LoadDefault()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	tmuxClient := tmux.NewClient()
	tmuxClient.TmuxPath = cfg.TmuxPath

	if !tmuxClient.IsInstalled(context.Background()) {
		return errors.New("tmux is not installed. install it with: sudo apt install tmux (Linux) or brew install tmux (macOS)")
	}

	api.Version = version
	srv := api.NewServer(tmuxClient, cfg)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		<-ctx.Done()

		log.Println("shutting down...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Printf("shutdown error: %v", err)
		}
	}()

	log.Printf("iTTY daemon %s starting (tmux: %s)", version, cfg.TmuxPath)
	if err := srv.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return nil
}

func runStatus() error {
	cfg, err := config.LoadDefault()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	tmuxClient := tmux.NewClient()
	tmuxClient.TmuxPath = cfg.TmuxPath
	ctx := context.Background()

	configPath, err := config.ConfigPath()
	if err != nil {
		return fmt.Errorf("resolving config path: %w", err)
	}

	fmt.Printf("iTTY daemon %s\n", version)
	fmt.Printf("config file:     %s\n", configPath)
	fmt.Printf("listen addr:     %s\n", cfg.ListenAddr)
	fmt.Printf("tmux path:       %s\n", cfg.TmuxPath)
	fmt.Printf("tmux installed:  %v\n", tmuxClient.IsInstalled(ctx))
	if v, err := tmuxClient.Version(ctx); err == nil {
		fmt.Printf("tmux version:    %s\n", v)
	}
	fmt.Printf("tmux running:    %v\n", tmuxClient.IsRunning(ctx))
	fmt.Printf("auto-wrap config:%v\n", cfg.AutoWrap)

	shellInfo, err := shell.Detect()
	if err == nil {
		fmt.Printf("shell:           %s (%s)\n", shellInfo.Type, shellInfo.Path)
		fmt.Printf("auto-wrap file:  %v\n", shell.IsConfigured(shellInfo))
		fmt.Printf("rc file:         %s\n", shellInfo.RCFile)
	}

	sessions, err := tmuxClient.ListSessions(ctx)
	if err != nil {
		return fmt.Errorf("listing sessions: %w", err)
	}

	fmt.Printf("sessions:        %d\n", len(sessions))
	for _, s := range sessions {
		status := " "
		if s.Attached {
			status = "*"
		}
		fmt.Printf("  %s %s (%d windows) [%s] %s\n", status, s.Name, s.Windows, s.LastPaneCmd, s.LastPanePath)
	}

	return nil
}

func runAuto(toggle string) error {
	cfg, err := config.LoadDefault()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	shellInfo, err := shell.Detect()
	if err != nil {
		return fmt.Errorf("cannot detect shell: %w", err)
	}

	switch toggle {
	case "on":
		if err := shell.Configure(shellInfo); err != nil {
			return fmt.Errorf("failed to configure auto-wrap: %w", err)
		}
		cfg.AutoWrap = true
		if err := config.SaveDefault(cfg); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}
		fmt.Printf("auto-wrap enabled in %s\n", shellInfo.RCFile)
		fmt.Println("new terminal windows will automatically use tmux sessions")
	case "off":
		if err := shell.Unconfigure(shellInfo); err != nil {
			return fmt.Errorf("failed to unconfigure auto-wrap: %w", err)
		}
		cfg.AutoWrap = false
		if err := config.SaveDefault(cfg); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}
		fmt.Printf("auto-wrap disabled in %s\n", shellInfo.RCFile)
	default:
		fmt.Println("usage: itty auto <on|off>")
		os.Exit(1)
	}

	return nil
}

func runSessions() error {
	cfg, err := config.LoadDefault()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	tmuxClient := tmux.NewClient()
	tmuxClient.TmuxPath = cfg.TmuxPath

	sessions, err := tmuxClient.ListSessions(context.Background())
	if err != nil {
		return fmt.Errorf("failed to list sessions: %w", err)
	}
	if len(sessions) == 0 {
		fmt.Println("no tmux sessions")
		return nil
	}

	for _, s := range sessions {
		status := " "
		if s.Attached {
			status = "*"
		}
		fmt.Printf("%s %-20s %d windows  [%s] %s\n", status, s.Name, s.Windows, s.LastPaneCmd, s.LastPanePath)
	}

	return nil
}

func printHelp() {
	fmt.Println(`iTTY Daemon — Desktop companion for the iTTY iOS terminal app

Usage:
  itty              Start the daemon (default)
  itty start        Start the daemon
  itty status       Show daemon and tmux status
  itty auto on      Enable auto-tmux shell wrapping
  itty auto off     Disable auto-tmux shell wrapping
  itty sessions     List tmux sessions
  itty version      Print version
  itty help         Show this help`)
}
