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
//	itty stop         Stop the running daemon
//	itty status       Show daemon status
//	itty auto on      Enable auto-tmux shell wrapping
//	itty auto off     Disable auto-tmux shell wrapping
//	itty sessions     List tmux sessions (CLI)
//	itty version      Print version
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

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

	switch cmd {
	case "start", "":
		runDaemon()
	case "status":
		runStatus()
	case "auto":
		if len(os.Args) < 3 {
			fmt.Println("usage: itty auto <on|off>")
			os.Exit(1)
		}
		runAuto(os.Args[2])
	case "sessions":
		runSessions()
	case "version":
		fmt.Printf("itty %s\n", version)
	case "help", "-h", "--help":
		printHelp()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", cmd)
		printHelp()
		os.Exit(1)
	}
}

func runDaemon() {
	cfg := config.DefaultConfig()
	tmuxClient := tmux.NewClient()
	tmuxClient.TmuxPath = cfg.TmuxPath

	// Check tmux availability
	if !tmuxClient.IsInstalled(context.Background()) {
		log.Fatal("tmux is not installed. Install it with: sudo apt install tmux (Linux) or brew install tmux (macOS)")
	}

	// Set version for API responses
	api.Version = version

	// Start HTTP server
	srv := api.NewServer(tmuxClient, cfg)

	// Graceful shutdown on SIGINT/SIGTERM
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		if err := srv.Start(); err != nil {
			log.Printf("server stopped: %v", err)
		}
	}()

	log.Printf("iTTY daemon %s started (tmux: %s)", version, cfg.TmuxPath)
	<-ctx.Done()
	log.Println("shutting down...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5_000_000_000) // 5s
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown error: %v", err)
	}
}

func runStatus() {
	tmuxClient := tmux.NewClient()
	ctx := context.Background()

	fmt.Printf("iTTY daemon %s\n", version)
	fmt.Printf("tmux installed: %v\n", tmuxClient.IsInstalled(ctx))
	if v, err := tmuxClient.Version(ctx); err == nil {
		fmt.Printf("tmux version:   %s\n", v)
	}
	fmt.Printf("tmux running:   %v\n", tmuxClient.IsRunning(ctx))

	shellInfo, err := shell.Detect()
	if err == nil {
		fmt.Printf("shell:          %s (%s)\n", shellInfo.Type, shellInfo.Path)
		fmt.Printf("auto-wrap:      %v\n", shell.IsConfigured(shellInfo))
		fmt.Printf("rc file:        %s\n", shellInfo.RCFile)
	}

	sessions, err := tmuxClient.ListSessions(ctx)
	if err == nil {
		fmt.Printf("sessions:       %d\n", len(sessions))
		for _, s := range sessions {
			status := " "
			if s.Attached {
				status = "*"
			}
			fmt.Printf("  %s %s (%d windows) [%s] %s\n", status, s.Name, s.Windows, s.LastPaneCmd, s.LastPanePath)
		}
	}
}

func runAuto(toggle string) {
	shellInfo, err := shell.Detect()
	if err != nil {
		log.Fatalf("cannot detect shell: %v", err)
	}

	switch toggle {
	case "on":
		if err := shell.Configure(shellInfo); err != nil {
			log.Fatalf("failed to configure auto-wrap: %v", err)
		}
		fmt.Printf("auto-wrap enabled in %s\n", shellInfo.RCFile)
		fmt.Println("new terminal windows will automatically use tmux sessions")
	case "off":
		if err := shell.Unconfigure(shellInfo); err != nil {
			log.Fatalf("failed to unconfigure auto-wrap: %v", err)
		}
		fmt.Printf("auto-wrap disabled in %s\n", shellInfo.RCFile)
	default:
		fmt.Println("usage: itty auto <on|off>")
		os.Exit(1)
	}
}

func runSessions() {
	tmuxClient := tmux.NewClient()
	sessions, err := tmuxClient.ListSessions(context.Background())
	if err != nil {
		log.Fatalf("failed to list sessions: %v", err)
	}
	if len(sessions) == 0 {
		fmt.Println("no tmux sessions")
		return
	}
	for _, s := range sessions {
		status := " "
		if s.Attached {
			status = "*"
		}
		fmt.Printf("%s %-20s %d windows  [%s] %s\n", status, s.Name, s.Windows, s.LastPaneCmd, s.LastPanePath)
	}
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
