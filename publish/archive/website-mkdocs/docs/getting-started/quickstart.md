# Quick Start

Build your first GoRAI node in minutes.

## Create a New Project

```bash
mkdir my-robot && cd my-robot
go mod init my-robot
go get github.com/gorai/gorai
```

## Write Your First Node

Create `main.go`:

```go
package main

import (
    "context"
    "log"

    "github.com/gorai/gorai/pkg/node"
)

func main() {
    ctx := context.Background()

    n, err := node.New("my_robot", node.WithNATS("nats://localhost:4222"))
    if err != nil {
        log.Fatal(err)
    }
    defer n.Close()

    log.Println("Robot node started")
    n.Spin(ctx)
}
```

## Run It

```bash
go run main.go
```

## Next Steps

- [Concepts](concepts.md) - Understand nodes, resources, and messaging
- [Hello Sensor](../examples/hello-sensor.md) - Build a complete sensor component
