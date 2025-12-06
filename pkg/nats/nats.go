// Package nats provides NATS connection management for Gorai.
package nats

import (
	"fmt"

	"github.com/nats-io/nats.go"
)

// Options configures a NATS connection.
type Options struct {
	URL      string
	Name     string
	User     string
	Password string
	Token    string
}

// Connect establishes a connection to NATS.
func Connect(opts Options) (*nats.Conn, error) {
	natsOpts := []nats.Option{
		nats.Name(opts.Name),
	}

	if opts.User != "" && opts.Password != "" {
		natsOpts = append(natsOpts, nats.UserInfo(opts.User, opts.Password))
	}

	if opts.Token != "" {
		natsOpts = append(natsOpts, nats.Token(opts.Token))
	}

	nc, err := nats.Connect(opts.URL, natsOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	return nc, nil
}

// MustConnect connects to NATS or panics.
func MustConnect(opts Options) *nats.Conn {
	nc, err := Connect(opts)
	if err != nil {
		panic(err)
	}
	return nc
}
