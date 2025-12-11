---
title: "CLI Reference"
description: "Command-line interface documentation"
weight: 10
---

# CLI Reference

The `gorai` command-line tool provides utilities for development and deployment.

## Commands

### gorai init

Create a new Gorai project:

```bash
gorai init <project-name>
```

### gorai run

Run a Gorai application:

```bash
gorai run [--config config.yaml]
```

### gorai build

Build for deployment:

```bash
gorai build [--target linux/arm64]
```

### gorai version

Show version information:

```bash
gorai version
```

## Global Flags

| Flag | Description |
|------|-------------|
| `--config` | Configuration file path |
| `--log-level` | Log level (debug, info, warn, error) |
| `--nats-url` | NATS server URL |
