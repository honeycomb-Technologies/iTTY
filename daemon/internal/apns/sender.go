// Package apns sends push notifications to iOS devices via Apple Push
// Notification service. It wraps the apns2 library and loads credentials
// from the daemon config.
package apns

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"

	"github.com/sideshow/apns2"
	"github.com/sideshow/apns2/payload"
	"github.com/sideshow/apns2/token"
)

// minTokenLength is the minimum acceptable APNs device token length.
// Real tokens are 64 hex characters; this guards against obviously invalid values.
const minTokenLength = 32

// ErrInvalidToken indicates a device token that is too short or empty.
var ErrInvalidToken = errors.New("invalid device token")

// Sender manages APNs connections and sends push notifications.
type Sender struct {
	client *apns2.Client
	topic  string
}

// NewSender creates a Sender from an APNs .p8 key file.
// Returns nil if any required config is empty (graceful no-op).
// Set production to true for App Store / TestFlight builds.
func NewSender(keyPath, keyID, teamID, bundleID string, production bool) (*Sender, error) {
	if keyPath == "" || keyID == "" || teamID == "" {
		return nil, nil
	}

	authKey, err := token.AuthKeyFromFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("loading APNs key from %s: %w", keyPath, err)
	}

	tok := &token.Token{
		AuthKey: authKey,
		KeyID:   keyID,
		TeamID:  teamID,
	}

	client := apns2.NewTokenClient(tok)
	if production {
		client = client.Production()
	} else {
		client = client.Development()
	}

	return &Sender{
		client: client,
		topic:  bundleID,
	}, nil
}

// Send pushes a notification to the given device token.
func (s *Sender) Send(ctx context.Context, deviceToken string, title, body string) error {
	p := payload.NewPayload().
		AlertTitle(title).
		AlertBody(body).
		Sound("default").
		Badge(1)

	notification := &apns2.Notification{
		DeviceToken: deviceToken,
		Topic:       s.topic,
		Payload:     p,
	}

	resp, err := s.client.PushWithContext(ctx, notification)
	if err != nil {
		return fmt.Errorf("APNs push: %w", err)
	}

	if !resp.Sent() {
		return fmt.Errorf("APNs rejected: %d %s", resp.StatusCode, resp.Reason)
	}

	return nil
}

// ValidateToken checks that a device token meets minimum length requirements.
func ValidateToken(token string) error {
	if len(token) < minTokenLength {
		return fmt.Errorf("%w: must be at least %d characters, got %d", ErrInvalidToken, minTokenLength, len(token))
	}
	return nil
}

// redactToken returns a safely truncated token for logging.
func redactToken(token string) string {
	if len(token) < 12 {
		return "<short>"
	}
	return token[:8] + "…" + token[len(token)-4:]
}

// DeviceStore is a simple in-memory store for APNs device tokens.
// Tokens are lost on daemon restart — acceptable for v1.
type DeviceStore struct {
	mu     sync.RWMutex
	tokens map[string]struct{}
}

// NewDeviceStore creates an empty device store.
func NewDeviceStore() *DeviceStore {
	return &DeviceStore{
		tokens: make(map[string]struct{}),
	}
}

// Register validates and adds a device token. Returns an error if the token
// is too short.
func (d *DeviceStore) Register(token string) error {
	if err := ValidateToken(token); err != nil {
		return err
	}

	d.mu.Lock()
	defer d.mu.Unlock()
	d.tokens[token] = struct{}{}
	log.Printf("apns: registered device %s", redactToken(token))
	return nil
}

// Unregister removes a device token.
func (d *DeviceStore) Unregister(token string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	delete(d.tokens, token)
}

// All returns all registered device tokens.
func (d *DeviceStore) All() []string {
	d.mu.RLock()
	defer d.mu.RUnlock()
	tokens := make([]string, 0, len(d.tokens))
	for t := range d.tokens {
		tokens = append(tokens, t)
	}
	return tokens
}

// NotifyAll sends a push notification to all registered devices.
// Errors are logged but not returned — best-effort delivery.
func NotifyAll(ctx context.Context, sender *Sender, store *DeviceStore, title, body string) {
	if sender == nil || store == nil {
		return
	}

	for _, token := range store.All() {
		if err := sender.Send(ctx, token, title, body); err != nil {
			log.Printf("apns: push to %s failed: %v", redactToken(token), err)
		}
	}
}
