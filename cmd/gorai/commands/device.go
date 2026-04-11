package commands

import (
	"context"
	"fmt"
	"os"
	"time"

	gorainats "github.com/gorai/gorai/pkg/nats"
)

func cmdDevice() error {
	args := os.Args[2:]
	if len(args) == 0 {
		return printDeviceUsage()
	}

	switch args[0] {
	case "reset":
		return cmdDeviceReset(args[1:])
	case "-h", "--help":
		return printDeviceUsage()
	default:
		return fmt.Errorf("unknown device subcommand: %s\n\nRun 'gorai device --help' for usage.", args[0])
	}
}

func cmdDeviceReset(args []string) error {
	var natsPrefix string
	var deviceID string
	var natsURL string

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--nats-prefix":
			if i+1 >= len(args) {
				return fmt.Errorf("--nats-prefix requires a value")
			}
			i++
			natsPrefix = args[i]
		case "--device-id":
			if i+1 >= len(args) {
				return fmt.Errorf("--device-id requires a value")
			}
			i++
			deviceID = args[i]
		case "--nats-url":
			if i+1 >= len(args) {
				return fmt.Errorf("--nats-url requires a value")
			}
			i++
			natsURL = args[i]
		case "-h", "--help":
			return printDeviceResetUsage()
		default:
			return fmt.Errorf("unknown flag: %s", args[i])
		}
	}

	if natsPrefix == "" {
		return fmt.Errorf("--nats-prefix is required")
	}
	if deviceID == "" {
		return fmt.Errorf("--device-id is required")
	}

	if natsURL == "" {
		natsURL = os.Getenv("NATS_URL")
	}
	if natsURL == "" {
		natsURL = "nats://localhost:4222"
	}

	topic := fmt.Sprintf("%s.%s.tx.system.reset", natsPrefix, deviceID)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client, err := gorainats.Connect(ctx, &gorainats.Config{
		URL:            natsURL,
		Name:           "gorai-device-reset",
		ConnectTimeout: 5 * time.Second,
		ReconnectWait:  1 * time.Second,
		MaxReconnects:  0,
	})
	if err != nil {
		return fmt.Errorf("failed to connect to NATS at %s: %w", natsURL, err)
	}
	defer client.Close()

	resetPayload := []byte(`{"subsystem":0}`)
	if err := client.Publish(topic, resetPayload); err != nil {
		return fmt.Errorf("failed to publish reset to %s: %w", topic, err)
	}

	time.Sleep(500 * time.Millisecond)

	fmt.Fprintf(os.Stderr, "Reset sent to %s\n", topic)
	return nil
}

func printDeviceUsage() error {
	fmt.Println(`gorai device - Device management commands

Usage:
  gorai device <subcommand> [flags]

Subcommands:
  reset    Send a reset command to a device via NATS

Use "gorai device <subcommand> --help" for more information.`)
	return nil
}

func printDeviceResetUsage() error {
	fmt.Println(`gorai device reset - Send a reset command to a device

Usage:
  gorai device reset --nats-prefix <prefix> --device-id <id> [flags]

Connects to NATS and publishes an empty message to the device reset topic:
  <nats_prefix>.<device_id>.tx.system.reset

Flags:
  --nats-prefix <prefix>   NATS topic prefix for the device (required)
  --device-id <id>         Device identifier (required)
  --nats-url <url>         NATS server URL (default: $NATS_URL or nats://localhost:4222)
  -h, --help               Show this help message

Examples:
  gorai device reset --nats-prefix robot1 --device-id pwm-controller
  gorai device reset --nats-prefix robot1 --device-id pwm-controller --nats-url nats://10.0.0.1:4222`)
	return nil
}
