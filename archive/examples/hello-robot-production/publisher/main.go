package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/nats-io/nats.go"
)

// Message is the structure we'll publish
type Message struct {
	Text  string `json:"text"`
	Count int    `json:"count"`
}

// HealthStatus tracks the health of the application
type HealthStatus struct {
	mu        sync.RWMutex
	natsConn  *nats.Conn
	connected bool
}

func (h *HealthStatus) setConnected(conn *nats.Conn, connected bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.natsConn = conn
	h.connected = connected
}

func (h *HealthStatus) isHealthy() bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.connected && h.natsConn != nil && h.natsConn.IsConnected()
}

func main() {
	// Get NATS URL from environment or use default
	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		natsURL = "nats://localhost:4222"
	}

	// Get health check port from environment or use default
	healthPort := os.Getenv("HEALTH_PORT")
	if healthPort == "" {
		healthPort = "8080"
	}

	// Create health status tracker
	health := &HealthStatus{}

	// Setup health check HTTP server
	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		if health.isHealthy() {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte("NATS not connected"))
		}
	})

	http.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		if health.isHealthy() {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Ready"))
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte("Not ready - NATS not connected"))
		}
	})

	// Start HTTP server in background
	go func() {
		log.Printf("Starting health check server on :%s", healthPort)
		if err := http.ListenAndServe(":"+healthPort, nil); err != nil {
			log.Printf("Health check server error: %v", err)
		}
	}()

	log.Printf("Connecting to NATS at %s...", natsURL)

	// Connect to NATS with retry logic
	var nc *nats.Conn
	var err error
	for i := 0; i < 30; i++ {
		nc, err = nats.Connect(natsURL,
			nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
				log.Printf("NATS disconnected: %v", err)
				health.setConnected(nc, false)
			}),
			nats.ReconnectHandler(func(nc *nats.Conn) {
				log.Printf("NATS reconnected")
				health.setConnected(nc, true)
			}),
			nats.ClosedHandler(func(nc *nats.Conn) {
				log.Printf("NATS connection closed")
				health.setConnected(nc, false)
			}),
		)
		if err == nil {
			break
		}
		log.Printf("Connection attempt %d failed: %v (retrying...)", i+1, err)
		time.Sleep(2 * time.Second)
	}

	if err != nil {
		log.Fatalf("Failed to connect to NATS after 30 attempts: %v", err)
	}
	defer nc.Close()

	// Mark as connected
	health.setConnected(nc, true)
	log.Println("Connected to NATS successfully")

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Publish counter
	count := 0

	// Create ticker for publishing
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	log.Println("Starting publisher (Ctrl+C to stop)...")
	log.Printf("Health endpoints available at http://localhost:%s/healthz and http://localhost:%s/readyz", healthPort, healthPort)

	for {
		select {
		case <-ticker.C:
			count++
			msg := Message{
				Text:  fmt.Sprintf("Hello #%d", count),
				Count: count,
			}

			// Marshal to JSON
			data, err := json.Marshal(msg)
			if err != nil {
				log.Printf("Error marshaling message: %v", err)
				continue
			}

			// Publish to NATS
			if err := nc.Publish("hello.messages", data); err != nil {
				log.Printf("Error publishing message: %v", err)
				// Mark as unhealthy if publish fails
				if !nc.IsConnected() {
					health.setConnected(nc, false)
				}
				continue
			}

			log.Printf("Published: %s", msg.Text)

		case sig := <-sigChan:
			log.Printf("Received signal %v, shutting down gracefully...", sig)
			health.setConnected(nc, false)
			return
		}
	}
}
