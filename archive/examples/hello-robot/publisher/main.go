package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nats-io/nats.go"
)

// Message is the structure we'll publish
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

	// Publish counter
	count := 0

	// Create ticker for publishing
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	log.Println("Starting publisher (Ctrl+C to stop)...")

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
				continue
			}

			log.Printf("Published: %s", msg.Text)

		case sig := <-sigChan:
			log.Printf("Received signal %v, shutting down...", sig)
			return
		}
	}
}
