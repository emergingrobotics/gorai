package main

import (
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nats-io/nats.go"
)

// Message is the structure we'll receive
type Message struct {
	Text  string `json:"text"`
	Count int    `json:"count"`
}

func main() {
	// Get NATS URL from environment or use default
	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		natsURL = "nats://localhost:4222"
	}

	log.Printf("Connecting to NATS at %s...", natsURL)

	// Connect to NATS with retry logic
	var nc *nats.Conn
	var err error
	for i := 0; i < 30; i++ {
		nc, err = nats.Connect(natsURL)
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

	log.Println("Subscribed to 'hello.messages' (Ctrl+C to stop)...")

	// Wait for interrupt
	<-sigChan
	log.Println("Shutting down...")
}
