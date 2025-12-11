---
title: "Quick Start"
description: "Build your first Gorai component in 5 minutes"
weight: 20
---

# Quick Start

Let's build a simple temperature sensor that publishes readings to NATS.

## 1. Start NATS

In a terminal, start the NATS server:

```bash
nats-server
```

## 2. Create a Project

```bash
mkdir hello-sensor && cd hello-sensor
go mod init hello-sensor
```

## 3. Write the Sensor

Create `main.go`:

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

    fmt.Println("Connected to NATS, publishing temperature readings...")

    // Publish temperature readings every second
    ticker := time.NewTicker(time.Second)
    for range ticker.C {
        temp := 20.0 + rand.Float64()*10.0 // 20-30°C
        msg := fmt.Sprintf("%.2f", temp)

        nc.Publish("sensors.temperature.reading", []byte(msg))
        fmt.Printf("Published: %s°C\n", msg)
    }
}
```

## 4. Run It

```bash
go mod tidy
go run main.go
```

You should see:

```
Connected to NATS, publishing temperature readings...
Published: 24.56°C
Published: 21.33°C
...
```

## 5. Subscribe to Messages

In another terminal, use the NATS CLI to see messages:

```bash
nats sub "sensors.>"
```

## Next Steps

- Learn about [Core Concepts](../concepts/)
- Explore the [Components Guide](/docs/guides/components/)
- Try the [Hello Sensor Example](/examples/hello-sensor/)
