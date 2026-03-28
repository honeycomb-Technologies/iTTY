package main

import (
	"context"
	"fmt"
	"net"
	"strconv"

	"github.com/honeycomb-Technologies/iTTY/daemon/internal/tailscale"
)

type tailscaleService interface {
	IsInstalled() bool
	IsRunning(context.Context) bool
	Hostname(context.Context) (string, error)
	ServePort(context.Context, int) error
}

type tailscaleServeResult struct {
	Enabled  bool
	Hostname string
}

func maybeConfigureTailscaleServe(ctx context.Context, client tailscaleService, listenAddr string) (tailscaleServeResult, error) {
	if client == nil || !client.IsInstalled() || !client.IsRunning(ctx) {
		return tailscaleServeResult{}, nil
	}

	port, err := listenPort(listenAddr)
	if err != nil {
		return tailscaleServeResult{}, err
	}

	if err := client.ServePort(ctx, port); err != nil {
		return tailscaleServeResult{}, err
	}

	result := tailscaleServeResult{Enabled: true}
	hostname, err := client.Hostname(ctx)
	if err == nil {
		result.Hostname = hostname
	}

	return result, nil
}

func listenPort(listenAddr string) (int, error) {
	_, portStr, err := net.SplitHostPort(listenAddr)
	if err != nil {
		return 0, fmt.Errorf("invalid listen address %q: %w", listenAddr, err)
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		return 0, fmt.Errorf("invalid listen port %q: %w", portStr, err)
	}
	if port < 1 || port > 65535 {
		return 0, fmt.Errorf("invalid listen port %d", port)
	}

	return port, nil
}

func newTailscaleClient() tailscaleService {
	return tailscale.NewClient()
}
