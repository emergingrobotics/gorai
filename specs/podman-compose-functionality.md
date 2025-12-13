# Gorai Container Orchestration with Podman

**Version**: 1.0
**Date**: 2025-12-13
**Status**: Specification

## Overview

Gorai uses **Podman** as its container runtime for orchestrating robot components and services. This specification defines how the Robot Definition Language (RDL) integrates with Podman to build, run, and manage containerized robot systems.

## Design Principles

### 1. Go-First Language Policy

Gorai's primary implementation language is **Go**. However, specific components may use other languages when required by technology constraints:

| Language | Use Case | Justification |
|----------|----------|---------------|
| **Go** | All core components, CLI, services | Primary language |
| **Python** | ML inference (Hailo, TensorFlow) | Best NPU/ML library support |
| **C/C++** | Hardware drivers (if no Go option) | Low-level hardware access |
| **TinyGo** | Microcontroller firmware | Embedded systems |

When a non-Go language is required, that component runs in an **isolated OCI container**.

### 2. Container Isolation Strategy

```
┌─────────────────────────────────────────────────────────────┐
│ Host System (Raspberry Pi 5)                                │
│                                                              │
│  ┌────────────────────────────────────────────────────────┐│
│  │ gorai-core container (Go)                              ││
│  │  - Camera component                                     ││
│  │  - Motor component                                      ││
│  │  - Dashboard service                                    ││
│  │  - NATS client                                         ││
│  └────────────────────────────────────────────────────────┘│
│                         │ NATS (localhost:4222)            │
│  ┌────────────────────────────────────────────────────────┐│
│  │ gorai-hailo container (Python)                         ││
│  │  - Object detection service                            ││
│  │  - Hailo runtime                                       ││
│  │  - /dev/hailo0 passthrough                             ││
│  └────────────────────────────────────────────────────────┘│
│                         │ NATS                             │
│  ┌────────────────────────────────────────────────────────┐│
│  │ nats container                                         ││
│  │  - NATS server                                         ││
│  │  - Port 4222, 8222                                     ││
│  └────────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────────┘
```

### 3. Why Podman (Not Docker)

- **Rootless by default**: Better security for robot systems
- **Systemd integration**: Native service management
- **Daemonless**: No background daemon required
- **Docker-compatible**: Same CLI, Dockerfile support
- **Quadlet**: Generates systemd units from containers
- **Pod support**: Group related containers

---

## RDL Container Schema

### Extended RDL Structure

```json
{
  "version": "1",
  "robot": {
    "name": "watchbot",
    "description": "Person detection robot"
  },

  "nats": {
    "url": "nats://nats:4222",
    "container": "nats"
  },

  "containers": {
    "nats": {
      "image": "docker.io/nats:2.10-alpine",
      "ports": ["4222:4222", "8222:8222"],
      "restart": "always",
      "healthcheck": {
        "test": ["CMD", "wget", "-q", "--spider", "http://localhost:8222/healthz"],
        "interval": "10s",
        "timeout": "5s",
        "retries": 3
      }
    },

    "gorai-core": {
      "build": {
        "context": ".",
        "dockerfile": "Containerfile"
      },
      "image": "localhost/watchbot-core:latest",
      "depends_on": {
        "nats": {
          "condition": "service_healthy"
        }
      },
      "environment": {
        "GORAI_ROBOT_NAME": "${robot.name}",
        "NATS_URL": "nats://nats:4222"
      },
      "volumes": [
        "./robot.json:/etc/gorai/robot.json:ro",
        "/dev/video0:/dev/video0"
      ],
      "devices": [
        "/dev/video0"
      ],
      "ports": ["8080:8080"],
      "restart": "unless-stopped",
      "components": ["main_camera"],
      "services": ["dashboard"]
    },

    "gorai-hailo": {
      "build": {
        "context": "./services/object-detection",
        "dockerfile": "Containerfile.hailo"
      },
      "image": "localhost/watchbot-hailo:latest",
      "depends_on": {
        "nats": {
          "condition": "service_healthy"
        },
        "gorai-core": {
          "condition": "service_started"
        }
      },
      "environment": {
        "NATS_URL": "nats://nats:4222",
        "MODEL_PATH": "/models/yolox_s_leaky.hef"
      },
      "volumes": [
        "./models:/models:ro",
        "./robot.json:/etc/gorai/robot.json:ro"
      ],
      "devices": [
        "/dev/hailo0"
      ],
      "security_opt": ["label=disable"],
      "restart": "unless-stopped",
      "services": ["person_detector"]
    }
  },

  "components": [
    {
      "name": "main_camera",
      "type": "camera",
      "model": "v4l2",
      "container": "gorai-core",
      "attributes": {
        "device": "/dev/video0",
        "width": 640,
        "height": 480,
        "frame_rate": 30
      }
    }
  ],

  "services": [
    {
      "name": "dashboard",
      "type": "dashboard",
      "model": "web",
      "container": "gorai-core",
      "attributes": {
        "listen": ":8080"
      }
    },
    {
      "name": "person_detector",
      "type": "object_detection",
      "model": "hailo_yolox",
      "container": "gorai-hailo",
      "attributes": {
        "model_path": "/models/yolox_s_leaky.hef",
        "input_topic": "gorai.watchbot.main_camera.data",
        "output_topic": "gorai.watchbot.person_detector.detections",
        "confidence_threshold": 0.5,
        "classes": ["person"]
      }
    }
  ],

  "dashboard": {
    "enabled": true,
    "listen": ":8080"
  }
}
```

### Container Definition Schema

```json
{
  "containers": {
    "<container-name>": {
      // Image specification (one of: image, build)
      "image": "string",           // Pre-built image to pull
      "build": {                   // Build from source
        "context": "string",       // Build context path
        "dockerfile": "string",    // Containerfile path (default: Containerfile)
        "args": {                  // Build arguments
          "key": "value"
        },
        "target": "string"         // Multi-stage build target
      },

      // Dependencies
      "depends_on": {
        "<container-name>": {
          "condition": "service_started | service_healthy | service_completed_successfully"
        }
      },

      // Runtime configuration
      "environment": {
        "KEY": "value"
      },
      "env_file": ["path/to/.env"],

      // Storage
      "volumes": [
        "host-path:container-path:options",
        "named-volume:container-path"
      ],

      // Devices (for hardware access)
      "devices": [
        "/dev/video0",
        "/dev/hailo0",
        "/dev/i2c-1"
      ],

      // Networking
      "ports": ["host:container"],
      "network_mode": "bridge | host | none",
      "networks": ["network-name"],

      // Security
      "privileged": false,
      "security_opt": ["label=disable"],
      "cap_add": ["SYS_RAWIO"],
      "cap_drop": ["ALL"],

      // Resource limits
      "resources": {
        "limits": {
          "cpus": "2.0",
          "memory": "512M"
        },
        "reservations": {
          "cpus": "0.5",
          "memory": "256M"
        }
      },

      // Lifecycle
      "restart": "no | always | on-failure | unless-stopped",
      "stop_grace_period": "10s",

      // Health checking
      "healthcheck": {
        "test": ["CMD", "command", "args"],
        "interval": "30s",
        "timeout": "10s",
        "retries": 3,
        "start_period": "5s"
      },

      // Gorai-specific: which components/services run here
      "components": ["component-name"],
      "services": ["service-name"]
    }
  }
}
```

---

## Dependency Management

### Dependency Conditions

Inspired by Docker Compose v2, Gorai supports three dependency conditions:

| Condition | Description |
|-----------|-------------|
| `service_started` | Container has started (default) |
| `service_healthy` | Container passes health check |
| `service_completed_successfully` | Container exited with code 0 |

### Dependency Graph Example

```
                    ┌─────────┐
                    │  nats   │
                    └────┬────┘
                         │ service_healthy
            ┌────────────┼────────────┐
            │            │            │
            ▼            ▼            ▼
     ┌───────────┐ ┌───────────┐ ┌───────────┐
     │gorai-core │ │gorai-hailo│ │prometheus │
     └───────────┘ └─────┬─────┘ └───────────┘
                         │ service_started
                         │ (needs camera frames)
                         ▼
                  [starts after gorai-core]
```

### Implicit Dependencies

Gorai automatically infers dependencies from:

1. **NATS URL**: If `nats.container` is set, all containers depend on it
2. **Topic subscriptions**: Services subscribing to topics from other containers
3. **Component references**: Services referencing components in other containers

---

## Gorai CLI Commands

### `gorai start`

Starts all containers defined in the RDL.

```bash
# Start robot from RDL file
gorai start --config robot.json

# Start in detached mode
gorai start --config robot.json --detach

# Start specific containers only
gorai start --config robot.json --containers gorai-core,nats

# Rebuild images before starting
gorai start --config robot.json --build

# Force recreate containers
gorai start --config robot.json --force-recreate
```

**Under the hood**:
1. Parse RDL and extract container definitions
2. Generate `podman-compose.yaml` in temp directory
3. Run `podman-compose up` with appropriate flags

### `gorai stop`

Stops all running containers.

```bash
# Stop all containers
gorai stop --config robot.json

# Stop and remove containers
gorai stop --config robot.json --remove

# Stop with timeout
gorai stop --config robot.json --timeout 30s
```

**Under the hood**:
1. Run `podman-compose down`
2. Optionally remove volumes with `--volumes`

### `gorai status`

Shows status of all containers.

```bash
gorai status --config robot.json
```

**Output**:
```
CONTAINER       IMAGE                           STATUS          HEALTH
nats            docker.io/nats:2.10-alpine      Up 5 minutes    healthy
gorai-core      localhost/watchbot-core:latest  Up 5 minutes    -
gorai-hailo     localhost/watchbot-hailo:latest Up 4 minutes    -

COMPONENTS      CONTAINER       STATUS
main_camera     gorai-core      running

SERVICES        CONTAINER       STATUS
dashboard       gorai-core      running
person_detector gorai-hailo     running
```

### `gorai logs`

Stream logs from containers.

```bash
# All container logs
gorai logs --config robot.json

# Specific container
gorai logs --config robot.json --container gorai-hailo

# Follow logs
gorai logs --config robot.json --follow

# Last N lines
gorai logs --config robot.json --tail 100
```

### `gorai build`

Build container images.

```bash
# Build all images
gorai build --config robot.json

# Build specific container
gorai build --config robot.json --container gorai-hailo

# No cache
gorai build --config robot.json --no-cache
```

### `gorai exec`

Execute command in running container.

```bash
# Interactive shell
gorai exec --config robot.json gorai-core /bin/sh

# Run command
gorai exec --config robot.json gorai-hailo python --version
```

---

## Generated Podman Compose File

Gorai generates a `podman-compose.yaml` from the RDL:

```yaml
# Generated by gorai from robot.json
# DO NOT EDIT - regenerated on each 'gorai start'

version: "3.8"

services:
  nats:
    image: docker.io/nats:2.10-alpine
    container_name: watchbot-nats
    ports:
      - "4222:4222"
      - "8222:8222"
    restart: always
    healthcheck:
      test: ["CMD", "wget", "-q", "--spider", "http://localhost:8222/healthz"]
      interval: 10s
      timeout: 5s
      retries: 3

  gorai-core:
    build:
      context: .
      dockerfile: Containerfile
    image: localhost/watchbot-core:latest
    container_name: watchbot-gorai-core
    depends_on:
      nats:
        condition: service_healthy
    environment:
      GORAI_ROBOT_NAME: watchbot
      NATS_URL: nats://nats:4222
      GORAI_COMPONENTS: main_camera
      GORAI_SERVICES: dashboard
    volumes:
      - ./robot.json:/etc/gorai/robot.json:ro
    devices:
      - /dev/video0:/dev/video0
    ports:
      - "8080:8080"
    restart: unless-stopped

  gorai-hailo:
    build:
      context: ./services/object-detection
      dockerfile: Containerfile.hailo
    image: localhost/watchbot-hailo:latest
    container_name: watchbot-gorai-hailo
    depends_on:
      nats:
        condition: service_healthy
      gorai-core:
        condition: service_started
    environment:
      NATS_URL: nats://nats:4222
      MODEL_PATH: /models/yolox_s_leaky.hef
      GORAI_SERVICES: person_detector
    volumes:
      - ./models:/models:ro
      - ./robot.json:/etc/gorai/robot.json:ro
    devices:
      - /dev/hailo0:/dev/hailo0
    security_opt:
      - label=disable
    restart: unless-stopped

networks:
  default:
    name: watchbot-network
```

---

## Containerfiles

### Go Core Container (Containerfile)

```dockerfile
# Build stage
FROM docker.io/golang:1.22-alpine AS builder

WORKDIR /build

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Build
COPY . .
RUN CGO_ENABLED=0 go build -o gorai-robot ./cmd/gorai-robot

# Runtime stage
FROM docker.io/alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /build/gorai-robot /usr/local/bin/

# Create non-root user
RUN adduser -D -u 1000 gorai
USER gorai

ENTRYPOINT ["/usr/local/bin/gorai-robot"]
CMD ["--config", "/etc/gorai/robot.json"]
```

### Python Hailo Container (Containerfile.hailo)

```dockerfile
# Hailo runtime base image
FROM docker.io/python:3.11-slim-bookworm

# Install Hailo runtime dependencies
RUN apt-get update && apt-get install -y \
    libhailort0 \
    hailort \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Install Python dependencies
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Copy application
COPY . .

# Create non-root user with access to hailo device
RUN useradd -m -u 1000 gorai && \
    usermod -a -G video gorai

USER gorai

ENTRYPOINT ["python", "main.py"]
CMD ["--config", "/etc/gorai/robot.json"]
```

---

## Device Passthrough

### Video Devices

```json
{
  "containers": {
    "gorai-core": {
      "devices": ["/dev/video0", "/dev/video1"],
      "volumes": ["/dev/video0:/dev/video0"]
    }
  }
}
```

**Note**: Device passthrough requires appropriate permissions. On Raspberry Pi:
```bash
# Add user to video group
sudo usermod -a -G video $USER

# For rootless podman
podman unshare chown 1000:44 /dev/video0
```

### Hailo NPU

```json
{
  "containers": {
    "gorai-hailo": {
      "devices": ["/dev/hailo0"],
      "security_opt": ["label=disable"],
      "cap_add": ["SYS_RAWIO"]
    }
  }
}
```

### I2C/SPI (Motor Controllers)

```json
{
  "containers": {
    "gorai-core": {
      "devices": ["/dev/i2c-1", "/dev/spidev0.0"],
      "cap_add": ["SYS_RAWIO"],
      "group_add": ["i2c", "spi", "gpio"]
    }
  }
}
```

### GPIO

```json
{
  "containers": {
    "gorai-core": {
      "devices": ["/dev/gpiochip0"],
      "volumes": ["/sys/class/gpio:/sys/class/gpio"],
      "privileged": false,
      "cap_add": ["SYS_RAWIO"]
    }
  }
}
```

---

## Networking

### Default: Bridge Network

All containers share a bridge network named `{robot-name}-network`:

```json
{
  "robot": {"name": "watchbot"},
  "containers": {
    "nats": { ... },
    "gorai-core": { ... }
  }
}
```

Containers can reach each other by name: `nats://nats:4222`

### Host Network Mode

For low-latency or complex networking:

```json
{
  "containers": {
    "gorai-core": {
      "network_mode": "host"
    }
  }
}
```

### Custom Networks

```json
{
  "networks": {
    "robot-internal": {
      "driver": "bridge",
      "internal": true
    },
    "robot-external": {
      "driver": "bridge"
    }
  },
  "containers": {
    "nats": {
      "networks": ["robot-internal", "robot-external"]
    },
    "gorai-core": {
      "networks": ["robot-internal"]
    }
  }
}
```

---

## Health Checks

### NATS Health Check

```json
{
  "containers": {
    "nats": {
      "healthcheck": {
        "test": ["CMD", "wget", "-q", "--spider", "http://localhost:8222/healthz"],
        "interval": "10s",
        "timeout": "5s",
        "retries": 3,
        "start_period": "5s"
      }
    }
  }
}
```

### Go Service Health Check

```json
{
  "containers": {
    "gorai-core": {
      "healthcheck": {
        "test": ["CMD", "wget", "-q", "--spider", "http://localhost:8080/health"],
        "interval": "30s",
        "timeout": "10s",
        "retries": 3
      }
    }
  }
}
```

### Python Service Health Check

```json
{
  "containers": {
    "gorai-hailo": {
      "healthcheck": {
        "test": ["CMD", "python", "-c", "import nats; exit(0)"],
        "interval": "30s",
        "timeout": "10s",
        "retries": 3
      }
    }
  }
}
```

---

## Resource Limits

### Memory and CPU

```json
{
  "containers": {
    "gorai-hailo": {
      "resources": {
        "limits": {
          "cpus": "2.0",
          "memory": "1G"
        },
        "reservations": {
          "cpus": "1.0",
          "memory": "512M"
        }
      }
    }
  }
}
```

### GPU/NPU Resources

For Hailo-8L:

```json
{
  "containers": {
    "gorai-hailo": {
      "devices": ["/dev/hailo0"],
      "resources": {
        "reservations": {
          "devices": [
            {
              "driver": "hailo",
              "count": 1,
              "capabilities": ["npu"]
            }
          ]
        }
      }
    }
  }
}
```

---

## Volumes and Persistence

### Named Volumes

```json
{
  "volumes": {
    "model-cache": {},
    "log-data": {
      "driver": "local",
      "driver_opts": {
        "type": "none",
        "o": "bind",
        "device": "/var/log/gorai"
      }
    }
  },
  "containers": {
    "gorai-hailo": {
      "volumes": [
        "model-cache:/app/cache",
        "log-data:/var/log"
      ]
    }
  }
}
```

### Bind Mounts

```json
{
  "containers": {
    "gorai-core": {
      "volumes": [
        "./robot.json:/etc/gorai/robot.json:ro",
        "./data:/var/gorai/data:rw",
        "/tmp/gorai:/tmp/gorai"
      ]
    }
  }
}
```

---

## Environment Variables

### Static Values

```json
{
  "containers": {
    "gorai-core": {
      "environment": {
        "LOG_LEVEL": "info",
        "NATS_URL": "nats://nats:4222"
      }
    }
  }
}
```

### RDL Variable Interpolation

```json
{
  "robot": {"name": "watchbot"},
  "containers": {
    "gorai-core": {
      "environment": {
        "GORAI_ROBOT_NAME": "${robot.name}",
        "GORAI_NATS_URL": "${nats.url}"
      }
    }
  }
}
```

### Environment Files

```json
{
  "containers": {
    "gorai-hailo": {
      "env_file": [
        ".env",
        ".env.local"
      ]
    }
  }
}
```

### Host Environment Passthrough

```json
{
  "containers": {
    "gorai-core": {
      "environment": {
        "HOME": null,
        "USER": null
      }
    }
  }
}
```

---

## Implementation

### Package Structure

```
/gorai/cmd/gorai/commands/
├── start.go         # gorai start command
├── stop.go          # gorai stop command
├── status.go        # gorai status command
├── logs.go          # gorai logs command
├── build.go         # gorai build command
├── exec.go          # gorai exec command
└── compose/
    ├── generate.go  # RDL → podman-compose.yaml
    ├── runner.go    # Executes podman-compose
    └── status.go    # Parses container status
```

### RDL to Compose Generation

```go
// compose/generate.go

type ComposeGenerator struct {
    rdl *config.RDL
}

func (g *ComposeGenerator) Generate() (*ComposeFile, error) {
    compose := &ComposeFile{
        Version:  "3.8",
        Services: make(map[string]Service),
        Networks: make(map[string]Network),
        Volumes:  make(map[string]Volume),
    }

    // Add default network
    compose.Networks["default"] = Network{
        Name: fmt.Sprintf("%s-network", g.rdl.Robot.Name),
    }

    // Convert containers
    for name, container := range g.rdl.Containers {
        service := g.convertContainer(name, container)
        compose.Services[name] = service
    }

    return compose, nil
}

func (g *ComposeGenerator) WriteYAML(path string) error {
    compose, err := g.Generate()
    if err != nil {
        return err
    }

    data, err := yaml.Marshal(compose)
    if err != nil {
        return err
    }

    return os.WriteFile(path, data, 0644)
}
```

### Start Command Implementation

```go
// commands/start.go

func runStart(cmd *cobra.Command, args []string) error {
    cfg, err := config.Load(configPath)
    if err != nil {
        return err
    }

    // Generate compose file
    gen := compose.NewGenerator(cfg)
    composePath := filepath.Join(os.TempDir(), "gorai-compose.yaml")
    if err := gen.WriteYAML(composePath); err != nil {
        return err
    }

    // Build images if requested
    if buildFlag {
        if err := runPodmanCompose(composePath, "build"); err != nil {
            return err
        }
    }

    // Start containers
    composeArgs := []string{"up"}
    if detachFlag {
        composeArgs = append(composeArgs, "-d")
    }
    if forceRecreateFlag {
        composeArgs = append(composeArgs, "--force-recreate")
    }

    return runPodmanCompose(composePath, composeArgs...)
}

func runPodmanCompose(composePath string, args ...string) error {
    cmdArgs := append([]string{"-f", composePath}, args...)
    cmd := exec.Command("podman-compose", cmdArgs...)
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr
    return cmd.Run()
}
```

---

## Systemd Integration

### Quadlet Support

Podman Quadlet generates systemd units. Gorai can generate Quadlet files:

```bash
gorai generate-systemd --config robot.json --output /etc/containers/systemd/
```

**Generated files**:

```ini
# /etc/containers/systemd/watchbot-nats.container
[Container]
Image=docker.io/nats:2.10-alpine
ContainerName=watchbot-nats
PublishPort=4222:4222
PublishPort=8222:8222

[Service]
Restart=always

[Install]
WantedBy=default.target
```

```ini
# /etc/containers/systemd/watchbot-gorai-core.container
[Container]
Image=localhost/watchbot-core:latest
ContainerName=watchbot-gorai-core
Environment=NATS_URL=nats://nats:4222
Volume=./robot.json:/etc/gorai/robot.json:ro
AddDevice=/dev/video0
PublishPort=8080:8080

[Service]
Restart=unless-stopped

[Unit]
Requires=watchbot-nats.service
After=watchbot-nats.service

[Install]
WantedBy=default.target
```

---

## Security Considerations

### Rootless Containers

Gorai containers run rootless by default:

```bash
# Enable linger for user
loginctl enable-linger $USER

# Start containers as user
gorai start --config robot.json
```

### Device Permissions

For device access in rootless mode:

```bash
# Add rules for video devices
echo 'SUBSYSTEM=="video4linux", GROUP="video", MODE="0660"' | \
  sudo tee /etc/udev/rules.d/99-gorai-video.rules

# Add rules for Hailo
echo 'SUBSYSTEM=="hailo", GROUP="hailo", MODE="0660"' | \
  sudo tee /etc/udev/rules.d/99-gorai-hailo.rules

# Reload udev
sudo udevadm control --reload-rules
sudo udevadm trigger
```

### SELinux/AppArmor

For systems with SELinux:

```json
{
  "containers": {
    "gorai-hailo": {
      "security_opt": ["label=disable"]
    }
  }
}
```

---

## Complete Example

### Directory Structure

```
/opt/watchbot/
├── robot.json              # RDL configuration
├── Containerfile           # Go core container
├── models/
│   └── yolox_s_leaky.hef   # ML model
├── services/
│   └── object-detection/
│       ├── Containerfile.hailo
│       ├── main.py
│       └── requirements.txt
└── data/                   # Runtime data
```

### Full RDL (robot.json)

```json
{
  "version": "1",
  "robot": {
    "name": "watchbot",
    "description": "Person detection security robot"
  },

  "nats": {
    "url": "nats://nats:4222",
    "container": "nats"
  },

  "containers": {
    "nats": {
      "image": "docker.io/nats:2.10-alpine",
      "ports": ["4222:4222", "8222:8222"],
      "restart": "always",
      "healthcheck": {
        "test": ["CMD", "wget", "-q", "--spider", "http://localhost:8222/healthz"],
        "interval": "10s",
        "timeout": "5s",
        "retries": 3
      }
    },

    "gorai-core": {
      "build": {
        "context": ".",
        "dockerfile": "Containerfile"
      },
      "image": "localhost/watchbot-core:latest",
      "depends_on": {
        "nats": {"condition": "service_healthy"}
      },
      "environment": {
        "NATS_URL": "nats://nats:4222",
        "LOG_LEVEL": "info"
      },
      "volumes": [
        "./robot.json:/etc/gorai/robot.json:ro"
      ],
      "devices": ["/dev/video0"],
      "ports": ["8080:8080"],
      "restart": "unless-stopped",
      "healthcheck": {
        "test": ["CMD", "wget", "-q", "--spider", "http://localhost:8080/health"],
        "interval": "30s",
        "timeout": "10s",
        "retries": 3
      },
      "components": ["main_camera"],
      "services": ["dashboard"]
    },

    "gorai-hailo": {
      "build": {
        "context": "./services/object-detection",
        "dockerfile": "Containerfile.hailo"
      },
      "image": "localhost/watchbot-hailo:latest",
      "depends_on": {
        "nats": {"condition": "service_healthy"},
        "gorai-core": {"condition": "service_started"}
      },
      "environment": {
        "NATS_URL": "nats://nats:4222",
        "MODEL_PATH": "/models/yolox_s_leaky.hef",
        "CONFIDENCE_THRESHOLD": "0.5"
      },
      "volumes": [
        "./models:/models:ro",
        "./robot.json:/etc/gorai/robot.json:ro"
      ],
      "devices": ["/dev/hailo0"],
      "security_opt": ["label=disable"],
      "restart": "unless-stopped",
      "resources": {
        "limits": {
          "memory": "1G"
        }
      },
      "services": ["person_detector"]
    }
  },

  "components": [
    {
      "name": "main_camera",
      "type": "camera",
      "model": "v4l2",
      "container": "gorai-core",
      "attributes": {
        "device": "/dev/video0",
        "width": 640,
        "height": 480,
        "frame_rate": 30,
        "jpeg_quality": 80
      }
    }
  ],

  "services": [
    {
      "name": "dashboard",
      "type": "dashboard",
      "model": "web",
      "container": "gorai-core",
      "attributes": {
        "listen": ":8080"
      }
    },
    {
      "name": "person_detector",
      "type": "object_detection",
      "model": "hailo_yolox",
      "container": "gorai-hailo",
      "attributes": {
        "model_path": "/models/yolox_s_leaky.hef",
        "input_topic": "gorai.watchbot.main_camera.data",
        "output_topic_annotated": "gorai.watchbot.person_detector.annotated",
        "output_topic_detections": "gorai.watchbot.person_detector.detections",
        "confidence_threshold": 0.5,
        "classes": ["person"]
      }
    }
  ],

  "dashboard": {
    "enabled": true,
    "listen": ":8080"
  }
}
```

### Usage

```bash
# Build and start robot
cd /opt/watchbot
gorai start --config robot.json --build --detach

# Check status
gorai status --config robot.json

# View logs
gorai logs --config robot.json --follow

# Stop robot
gorai stop --config robot.json
```

---

## Summary

This specification defines how Gorai uses Podman for container orchestration:

1. **Go-first policy**: Primary language is Go, containers isolate other languages
2. **RDL extensions**: Container definitions with dependencies, health checks, resources
3. **CLI commands**: `gorai start/stop/status/logs/build/exec`
4. **Podman compose**: Generated from RDL, manages container lifecycle
5. **Device passthrough**: Support for cameras, NPUs, GPIO
6. **Dependency ordering**: `depends_on` with health conditions
7. **Systemd integration**: Optional Quadlet generation

This approach provides:
- **Isolation**: Each language/runtime in its own container
- **Reproducibility**: Identical containers across deployments
- **Portability**: Standard OCI containers
- **Security**: Rootless containers by default
- **Manageability**: Single CLI for all container operations
