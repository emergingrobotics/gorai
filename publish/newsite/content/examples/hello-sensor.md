---
title: "Hello Sensor"
description: "Build your first Gorai sensor"
weight: 10
---

# Hello Sensor Example

This example demonstrates a simple temperature sensor that publishes readings to NATS.

## Prerequisites

- Go 1.21+
- NATS server running locally

## Code

```go
package main

import (
    "fmt"
    "math/rand"
    "time"

    "github.com/nats-io/nats.go"
)

func main() {
    // Connect to NATS
    nc, err := nats.Connect(nats.DefaultURL)
    if err != nil {
        panic(err)
    }
    defer nc.Close()

    fmt.Println("Hello Sensor starting...")

    // Publish readings every second
    ticker := time.NewTicker(time.Second)
    for range ticker.C {
        temp := 20.0 + rand.Float64()*10.0
        msg := fmt.Sprintf("%.2f", temp)

        nc.Publish("sensors.hello.reading", []byte(msg))
        fmt.Printf("Temperature: %s°C\n", msg)
    }
}
```

## Running

```bash
# Terminal 1: Start NATS
nats-server

# Terminal 2: Run the sensor
go run main.go

# Terminal 3: Subscribe to readings
nats sub "sensors.>"
```

## What's Next?

- [Pan-Tilt Platform](../pan-tilt/)
- [Components Guide](/docs/guides/components/)
