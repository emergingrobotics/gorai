package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/nats-io/nats.go"
)

// Message is the structure we'll receive
type Message struct {
	Text  string `json:"text"`
	Count int    `json:"count"`
}

// HealthStatus tracks the health of the application
type HealthStatus struct {
	mu           sync.RWMutex
	natsConn     *nats.Conn
	subscription *nats.Subscription
	connected    bool
	subscribed   bool
}

func (h *HealthStatus) setStatus(conn *nats.Conn, sub *nats.Subscription, connected, subscribed bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.natsConn = conn
	h.subscription = sub
	h.connected = connected
	h.subscribed = subscribed
}

func (h *HealthStatus) isHealthy() bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.connected &&
		h.natsConn != nil &&
		h.natsConn.IsConnected() &&
		h.subscribed &&
		h.subscription != nil &&
		h.subscription.IsValid()
}

func (h *HealthStatus) isReady() bool {
	// Same as healthy for subscriber
	return h.isHealthy()
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
			w.Write([]byte("Not healthy - NATS not connected or subscription invalid"))
		}
	})

	http.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		if health.isReady() {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Ready"))
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte("Not ready - NATS not connected or subscription invalid"))
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
				health.setStatus(nc, nil, false, false)
			}),
			nats.ReconnectHandler(func(nc *nats.Conn) {
				log.Printf("NATS reconnected")
				// Note: subscription will be re-established automatically by NATS client
				health.setStatus(nc, health.subscription, true, health.subscribed)
			}),
			nats.ClosedHandler(func(nc *nats.Conn) {
				log.Printf("NATS connection closed")
				health.setStatus(nc, nil, false, false)
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

	log.Println("Connected to NATS successfully")

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Subscribe to topic
	sub, err := nc.Subscribe("hello.messages", func(m *nats.Msg) {
		var msg Message
		if err := json.Unmarshal(m.Data, &msg); err != nil {
			log.Printf("Error unmarshaling message: %v", err)
			return
		}

		log.Printf("Received: %s (count=%d)", msg.Text, msg.Count)
	})

	if err != nil {
		log.Fatalf("Error subscribing: %v", err)
	}
	defer sub.Unsubscribe()

	// Mark as connected and subscribed
	health.setStatus(nc, sub, true, true)
	log.Println("Subscribed to 'hello.messages' (Ctrl+C to stop)...")
	log.Printf("Health endpoints available at http://localhost:%s/healthz and http://localhost:%s/readyz", healthPort, healthPort)

	// Wait for interrupt
	<-sigChan
	log.Println("Shutting down gracefully...")
	health.setStatus(nc, sub, false, false)
}
