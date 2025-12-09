---
title: "Contributing"
description: "How to contribute to Gorai"
weight: 10
---

# Contributing to Gorai

Thank you for your interest in contributing! Here's how to get started.

## Ways to Contribute

- **Bug reports** — Found a bug? Open an issue
- **Feature requests** — Have an idea? Start a discussion
- **Documentation** — Improve docs, fix typos
- **Code** — Fix bugs, implement features

## Development Setup

```bash
# Clone the repository
git clone https://github.com/gorai/gorai.git
cd gorai

# Install dependencies
go mod download

# Run tests
go test ./...
```

## Pull Request Process

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Run tests: `go test ./...`
5. Submit a pull request

## Code Style

- Follow standard Go conventions
- Run `go fmt` before committing
- Add tests for new functionality
- Update documentation as needed
