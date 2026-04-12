// Package embeddednats wraps the NATS server library to provide an
// in-process NATS server for single-binary robot deployments.
package embeddednats

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	natsserver "github.com/nats-io/nats-server/v2/server"
)

const (
	defaultHost         = "127.0.0.1"
	defaultPort         = 4222
	defaultJetStreamDir = "./data/jetstream/"

	// portFreeEphemeral instructs the NATS server to pick a free port.
	portFreeEphemeral = -1

	readyPollInterval   = 50 * time.Millisecond
	serverStartDeadline = 10 * time.Second
)

// TLSConfig holds TLS certificate paths for the embedded NATS server.
type TLSConfig struct {
	CAFile   string
	CertFile string
	KeyFile  string
}

// Config holds configuration for the embedded NATS server.
type Config struct {
	Host         string
	Port         int
	JetStream    bool
	JetStreamDir string
	TLS          *TLSConfig
	Logger       *slog.Logger
}

// Server wraps a nats-server instance for in-process use.
type Server struct {
	natsServer *natsserver.Server
	config     Config
	logger     *slog.Logger
}

// New creates a new embedded NATS server but does not start it.
func New(config Config) (*Server, error) {
	if config.Host == "" {
		config.Host = defaultHost
	}
	if config.Port == 0 {
		config.Port = defaultPort
	}
	if config.JetStreamDir == "" {
		config.JetStreamDir = defaultJetStreamDir
	}
	if config.Logger == nil {
		config.Logger = slog.Default()
	}

	options := &natsserver.Options{
		Host:     config.Host,
		Port:     config.Port,
		NoLog:    true,
		NoSigs:   true,
		MaxPending: 64 * 1024 * 1024,
	}

	if config.JetStream {
		options.JetStream = true
		options.StoreDir = config.JetStreamDir
	}

	if config.TLS != nil {
		options.TLSCert = config.TLS.CertFile
		options.TLSKey = config.TLS.KeyFile
		options.TLSCaCert = config.TLS.CAFile
		options.TLS = true
	}

	natsServer, err := natsserver.NewServer(options)
	if err != nil {
		return nil, fmt.Errorf("failed to create NATS server: %w", err)
	}

	return &Server{
		natsServer: natsServer,
		config:     config,
		logger:     config.Logger,
	}, nil
}

// Start starts the embedded NATS server and blocks until it is ready
// to accept connections.
func (server *Server) Start() error {
	server.natsServer.Start()

	ctx, cancel := context.WithTimeout(context.Background(), serverStartDeadline)
	defer cancel()

	if err := server.WaitReady(ctx); err != nil {
		server.natsServer.Shutdown()
		return fmt.Errorf("NATS server failed to become ready: %w", err)
	}

	server.logger.Info("embedded NATS server started",
		"url", server.ClientURL(),
		"jetstream", server.config.JetStream,
	)
	return nil
}

// WaitReady blocks until the server is accepting connections or the
// context is cancelled.
func (server *Server) WaitReady(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if server.natsServer.ReadyForConnections(readyPollInterval) {
				return nil
			}
		}
	}
}

// ClientURL returns the NATS connection URL for clients.
func (server *Server) ClientURL() string {
	return server.natsServer.ClientURL()
}

// Shutdown drains all connections and stops the server.
func (server *Server) Shutdown() {
	server.logger.Info("shutting down embedded NATS server")
	server.natsServer.Shutdown()
	server.natsServer.WaitForShutdown()
	server.logger.Info("embedded NATS server stopped")
}

// IsRunning returns true if the server is currently accepting connections.
func (server *Server) IsRunning() bool {
	return server.natsServer.Running()
}
