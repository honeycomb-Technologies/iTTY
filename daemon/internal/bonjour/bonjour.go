// Package bonjour advertises the iTTY daemon as a Bonjour (DNS-SD)
// service so that iOS clients on the same local network can discover
// it without any manual configuration.
package bonjour

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// ServiceType is the Bonjour service type the daemon registers.
const ServiceType = "_itty._tcp"

// Advertiser manages a Bonjour service registration.
type Advertiser struct {
	port     int
	hostname string
	cancel   context.CancelFunc
	done     chan struct{}
}

// Start registers the daemon as a Bonjour service on the local network.
// The service is advertised as _itty._tcp on the given port.
// Call Stop to unregister.
func Start(listenAddr string) (*Advertiser, error) {
	port, err := parsePort(listenAddr)
	if err != nil {
		return nil, err
	}

	hostname, err := os.Hostname()
	if err != nil {
		hostname = "iTTY-daemon"
	}

	ctx, cancel := context.WithCancel(context.Background())
	adv := &Advertiser{
		port:     port,
		hostname: hostname,
		cancel:   cancel,
		done:     make(chan struct{}),
	}

	go adv.run(ctx)
	return adv, nil
}

// Stop unregisters the Bonjour service.
func (a *Advertiser) Stop() {
	a.cancel()
	<-a.done
}

func (a *Advertiser) run(ctx context.Context) {
	defer close(a.done)

	cmd, err := advertisementCommand(ctx, a.hostname, a.port, exec.LookPath)
	if err != nil {
		log.Printf("bonjour: failed to build advertisement command: %v", err)
		return
	}
	if cmd == nil {
		log.Printf("bonjour: neither dns-sd nor avahi-publish found, skipping service advertisement")
		return
	}

	if err := cmd.Start(); err != nil {
		log.Printf("bonjour: failed to start advertisement: %v", err)
		return
	}

	log.Printf("bonjour: advertising %s on port %d as %q", ServiceType, a.port, a.hostname)

	// Wait for context cancellation or process exit
	err = cmd.Wait()
	if err != nil && ctx.Err() == nil {
		log.Printf("bonjour: advertisement process exited: %v", err)
	}
}

func advertisementCommand(
	ctx context.Context,
	hostname string,
	port int,
	lookPath func(string) (string, error),
) (*exec.Cmd, error) {
	portArg := strconv.Itoa(port)
	txtVersion := "version=1"
	txtHost := fmt.Sprintf("hostname=%s", hostname)

	if path, err := lookPath("dns-sd"); err == nil {
		return exec.CommandContext(
			ctx,
			path,
			"-R", hostname,
			ServiceType, "local",
			portArg,
			txtVersion,
			txtHost,
		), nil
	} else if err != nil && !errors.Is(err, exec.ErrNotFound) {
		return nil, fmt.Errorf("looking up dns-sd: %w", err)
	}

	if path, err := lookPath("avahi-publish"); err == nil {
		return exec.CommandContext(
			ctx,
			path,
			"-s", hostname,
			ServiceType, portArg,
			txtVersion,
			txtHost,
		), nil
	} else if err != nil && !errors.Is(err, exec.ErrNotFound) {
		return nil, fmt.Errorf("looking up avahi-publish: %w", err)
	}

	return nil, nil
}

func parsePort(listenAddr string) (int, error) {
	_, portStr, err := net.SplitHostPort(listenAddr)
	if err != nil {
		return 0, fmt.Errorf("invalid listen address %q: %w", listenAddr, err)
	}

	port, err := strconv.Atoi(strings.TrimSpace(portStr))
	if err != nil || port < 1 || port > 65535 {
		return 0, fmt.Errorf("invalid port %q", portStr)
	}
	return port, nil
}
