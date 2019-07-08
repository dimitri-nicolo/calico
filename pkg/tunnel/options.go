package tunnel

import (
	"time"
)

// Option is a common format for New() options
type Option func(*Tunnel) error

// WithKeepAliveSettings sets the Keep Alive settings for the tunnel.
func WithKeepAliveSettings(enable bool, intervalDuration time.Duration) Option {
	return func(t *Tunnel) error {
		t.keepAliveEnable = enable
		t.keepAliveInterval = intervalDuration
		return nil
	}
}
