# CLI Reference

GoRAI command-line tools and scripts.

## GoRAI Scripts

| Script | Purpose |
|--------|---------|
| `scripts/start.sh` | Start NATS and development services |
| `scripts/stop.sh` | Stop all services |
| `scripts/hello.sh` | Run hello-sensor example |

## Go Commands

| Command | Purpose |
|---------|---------|
| `go build ./...` | Build all packages |
| `go test ./...` | Run unit tests |
| `go test -tags=component ./...` | Run component tests |
| `go test -race ./...` | Test with race detector |
| `go test -cover ./...` | Test with coverage |

## Makefile Targets

| Target | Purpose |
|--------|---------|
| `make build` | Build all binaries |
| `make test` | Run unit tests |
| `make test-quick` | Unit + component tests |
| `make test-all` | All test levels |
| `make proto` | Generate proto code |
| `make lint` | Run linters |

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `NATS_URL` | `nats://localhost:4222` | NATS server URL |
| `GORAI_LOG_LEVEL` | `info` | Log verbosity (debug, info, warn, error) |
| `GORAI_CONFIG` | `./config.yaml` | Configuration file path |
