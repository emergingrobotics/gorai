# Gorai Publication Specification

**Version 0.1.0**

This specification defines the tooling, folder structure, and workflow for publishing Gorai documentation, including the book, developer website, and API reference.

*Pronounced "go-ray" (like "sting-ray")*

---

## Table of Contents

1. [Overview](#overview)
2. [Publication Components](#publication-components)
3. [Directory Structure](#directory-structure)
4. [Publishing Container](#publishing-container)
5. [Tool Configuration](#tool-configuration)
   - [mdBook Configuration](#mdbook-configuration)
   - [Material for MkDocs Configuration](#material-for-mkdocs-configuration)
   - [pkgsite Configuration](#pkgsite-configuration)
6. [Build Workflow](#build-workflow)
7. [Makefile Integration](#makefile-integration)
8. [Deployment](#deployment)
9. [Content Guidelines](#content-guidelines)

---

## Overview

Gorai documentation is published through three specialized tools, each optimized for its purpose:

| Content Type | Tool | Purpose |
|-------------|------|---------|
| **Book** | mdBook | Linear, tutorial-style content for learning Gorai |
| **Website** | Material for MkDocs | Developer portal with guides, examples, and navigation |
| **API Reference** | pkgsite | Go package documentation extracted from source |

All tools run inside a single unified container, ensuring:
- **No host dependencies** beyond Podman
- **Reproducible builds** across development machines and CI
- **Consistent output** regardless of build environment

---

## Publication Components

### The Book (mdBook)

The Gorai book is a comprehensive guide for learning the framework, structured as sequential chapters.

**Characteristics:**
- Linear reading experience (Chapter 1 → Chapter 15)
- Tutorial-focused with progressive complexity
- Complete code examples with explanations
- Cross-references between chapters

**Source:** `publish/book/src/`
**Output:** `publish/dist/book/`

### The Website (Material for MkDocs)

The developer website serves as the primary entry point for Gorai users.

**Characteristics:**
- Project landing page and overview
- Quick-start guides and installation
- Searchable documentation
- Links to book chapters and API reference
- Example projects and recipes
- Community and contribution guides

**Source:** `publish/website/docs/`
**Output:** `publish/dist/website/`

### API Reference (pkgsite)

Auto-generated Go package documentation from source code comments.

**Characteristics:**
- Extracted from godoc comments in source
- Examples from `_test.go` files
- Package hierarchy navigation
- Type, function, and method documentation

**Source:** Go source files in `pkg/`, `components/`, `services/`, etc.
**Output:** `publish/dist/api/`

---

## Directory Structure

```
gorai/
├── publish/                      # All publication materials
│   ├── container/                # Publishing container definition
│   │   ├── Containerfile         # Unified container with all tools
│   │   └── entrypoint.sh         # Container entrypoint script
│   │
│   ├── book/                     # mdBook source
│   │   ├── book.toml             # mdBook configuration
│   │   ├── src/                  # Book chapters (markdown)
│   │   │   ├── SUMMARY.md        # Table of contents
│   │   │   ├── ch01/             # Chapter 1: Why Gorai?
│   │   │   │   ├── README.md     # Chapter introduction
│   │   │   │   ├── landscape.md  # Section 1.1
│   │   │   │   ├── philosophy.md # Section 1.2
│   │   │   │   └── ...
│   │   │   ├── ch02/             # Chapter 2: Mental Model
│   │   │   └── ...
│   │   └── theme/                # Custom theme overrides (optional)
│   │
│   ├── website/                  # Material for MkDocs source
│   │   ├── mkdocs.yml            # MkDocs configuration
│   │   ├── docs/                 # Website pages (markdown)
│   │   │   ├── index.md          # Landing page
│   │   │   ├── getting-started/  # Quick start guides
│   │   │   ├── guides/           # How-to guides
│   │   │   ├── examples/         # Example projects
│   │   │   ├── reference/        # Reference materials
│   │   │   └── community/        # Contributing, support
│   │   └── overrides/            # Theme customizations
│   │
│   └── dist/                     # Build output (gitignored)
│       ├── book/                 # Built book HTML
│       ├── website/              # Built website HTML
│       └── api/                  # Built API reference HTML
│
├── book/                         # LEGACY - migrate to publish/book/
│   └── ...
│
├── docs/                         # Design documents (not published directly)
│   ├── general-designs.md
│   ├── ros2-design.md
│   └── ...
│
├── specs/                        # Specifications (referenced by website)
│   ├── gorai-framework-specification.md
│   ├── publication.md            # This file
│   └── ...
│
└── plans/                        # Planning documents
    ├── book.md                   # Book outline (reference for publish/book/)
    └── ...
```

### Directory Purposes

| Directory | Purpose | Published? |
|-----------|---------|------------|
| `publish/book/` | mdBook source files | Yes → `dist/book/` |
| `publish/website/` | MkDocs source files | Yes → `dist/website/` |
| `publish/container/` | Container definition | No |
| `publish/dist/` | Build outputs | No (gitignored) |
| `docs/` | Internal design documents | Selectively linked |
| `specs/` | Technical specifications | Selectively linked |
| `plans/` | Planning documents | No |
| `book/` | Legacy location | Migrate to `publish/book/` |

---

## Publishing Container

A single container includes all publishing tools for consistent builds.

### Containerfile

**Location:** `publish/container/Containerfile`

```dockerfile
# Gorai Publishing Container
# Includes: mdBook, Material for MkDocs, pkgsite
#
# Build:
#   podman build -t gorai-publish publish/container/
#
# Usage:
#   podman run --rm -v ${PWD}:/workspace gorai-publish <command>

FROM docker.io/library/golang:1.23-bookworm AS golang-base

# Install Rust for mdBook
RUN curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y
ENV PATH="/root/.cargo/bin:${PATH}"

# Install mdBook and plugins
RUN cargo install mdbook mdbook-mermaid mdbook-toc

# Install Python and Material for MkDocs
RUN apt-get update && apt-get install -y \
    python3 \
    python3-pip \
    python3-venv \
    && rm -rf /var/lib/apt/lists/*

RUN pip3 install --break-system-packages \
    mkdocs-material \
    mkdocs-mermaid2-plugin \
    mkdocs-minify-plugin \
    mkdocs-redirects

# Install pkgsite
RUN go install golang.org/x/pkgsite/cmd/pkgsite@latest

# Set up workspace
WORKDIR /workspace

# Copy entrypoint
COPY entrypoint.sh /usr/local/bin/
RUN chmod +x /usr/local/bin/entrypoint.sh

ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]
CMD ["help"]
```

### Entrypoint Script

**Location:** `publish/container/entrypoint.sh`

```bash
#!/bin/bash
set -e

case "$1" in
    book)
        # Build the book with mdBook
        cd /workspace/publish/book
        mdbook-mermaid install .
        mdbook build --dest-dir /workspace/publish/dist/book
        echo "Book built to publish/dist/book/"
        ;;

    book-serve)
        # Serve book with live reload
        cd /workspace/publish/book
        mdbook-mermaid install .
        mdbook serve --hostname 0.0.0.0 --port 3000
        ;;

    website)
        # Build website with MkDocs
        cd /workspace/publish/website
        mkdocs build --site-dir /workspace/publish/dist/website
        echo "Website built to publish/dist/website/"
        ;;

    website-serve)
        # Serve website with live reload
        cd /workspace/publish/website
        mkdocs serve --dev-addr 0.0.0.0:8000
        ;;

    api)
        # Build API reference with pkgsite
        cd /workspace
        # Generate static pages for local packages
        pkgsite -http=:6060 -list=false . &
        sleep 5
        # TODO: Static export when pkgsite supports it
        # For now, serve dynamically
        echo "API server running on port 6060"
        wait
        ;;

    api-serve)
        # Serve API reference dynamically
        cd /workspace
        pkgsite -http=0.0.0.0:6060 .
        ;;

    all)
        # Build everything
        $0 book
        $0 website
        echo "All documentation built to publish/dist/"
        ;;

    help|*)
        echo "Gorai Publishing Container"
        echo ""
        echo "Commands:"
        echo "  book         Build the mdBook book"
        echo "  book-serve   Serve book with live reload (port 3000)"
        echo "  website      Build the MkDocs website"
        echo "  website-serve Serve website with live reload (port 8000)"
        echo "  api-serve    Serve API reference (port 6060)"
        echo "  all          Build book and website"
        echo "  help         Show this message"
        ;;
esac
```

### Container Image Details

| Component | Version | Purpose |
|-----------|---------|---------|
| Go | 1.23 | pkgsite, base image |
| Rust | stable | mdBook compilation |
| mdBook | latest | Book generation |
| mdbook-mermaid | latest | Mermaid diagram support |
| mdbook-toc | latest | Table of contents generation |
| Python | 3.11+ | MkDocs runtime |
| mkdocs-material | latest | Website theme |
| mkdocs-mermaid2-plugin | latest | Mermaid in website |
| pkgsite | latest | API documentation |

---

## Tool Configuration

### mdBook Configuration

**Location:** `publish/book/book.toml`

```toml
[book]
title = "Gorai: Building Modern Robots with Go and NATS"
authors = ["The Gorai Authors"]
description = "A comprehensive guide to building robots with the Gorai framework"
language = "en"
multilingual = false
src = "src"

[build]
build-dir = "../dist/book"
create-missing = false

[output.html]
default-theme = "light"
preferred-dark-theme = "ayu"
git-repository-url = "https://github.com/gorai/gorai"
edit-url-template = "https://github.com/gorai/gorai/edit/main/publish/book/{path}"
site-url = "/book/"
cname = "gorai.dev"

[output.html.fold]
enable = true
level = 1

[output.html.playground]
editable = false
copyable = true
line-numbers = true

[output.html.search]
enable = true
limit-results = 30
use-hierarchical-headings = true

[preprocessor.mermaid]
command = "mdbook-mermaid"

[preprocessor.toc]
command = "mdbook-toc"
renderer = ["html"]
```

### mdBook SUMMARY.md Structure

**Location:** `publish/book/src/SUMMARY.md`

```markdown
# Summary

[Introduction](README.md)

# Getting Started

- [Why Gorai?](ch01/README.md)
    - [The Robotics Landscape](ch01/landscape.md)
    - [Design Philosophy](ch01/philosophy.md)
    - [Who Should Use Gorai](ch01/audience.md)
    - [What You'll Build](ch01/whatyoullbuild.md)
    - [Prerequisites](ch01/prerequisites.md)

- [Mental Model & Architecture](ch02/README.md)
    - [The Big Picture](ch02/bigpicture.md)
    - [Core Concepts](ch02/coreconcepts.md)
    - [Distributed Systems](ch02/distributed.md)
    - [Configuration](ch02/config.md)
    - [NWS/NWC Pattern](ch02/nwsnwc.md)

# Core Framework

- [NATS: The Communication Backbone](ch03/README.md)
    - [Why NATS?](ch03/whynats.md)
    - [NATS Fundamentals](ch03/fundamentals.md)
    - [Communication Patterns](ch03/patterns.md)
    - [Quality of Service](ch03/qos.md)
    - [JetStream](ch03/jetstream.md)
    - [CLI Tools](ch03/cli.md)

- [Components: Sensors](ch04/README.md)
- [Components: Actuators](ch05/README.md)
- [Components: Vision](ch06/README.md)
- [Services](ch07/README.md)

# Development

- [Development Environment](ch08/README.md)
- [Hello Sensor Deep Dive](ch09/README.md)
- [Building Custom Components](ch10/README.md)
- [Testing Strategies](ch11/README.md)

# Advanced Topics

- [AI/ML Integration](ch12/README.md)
- [Project Organization](ch13/README.md)
- [AI-Assisted Development](ch14/README.md)

# Conclusion

- [Conclusion & Next Steps](ch15/README.md)

---

[Appendices](appendices/README.md)
```

### Material for MkDocs Configuration

**Location:** `publish/website/mkdocs.yml`

```yaml
site_name: Gorai
site_url: https://gorai.dev
site_description: A lightweight, Go-based robotics framework built on NATS.io
site_author: The Gorai Authors

repo_name: gorai/gorai
repo_url: https://github.com/gorai/gorai
edit_uri: edit/main/publish/website/docs/

theme:
  name: material
  custom_dir: overrides
  language: en

  palette:
    - media: "(prefers-color-scheme: light)"
      scheme: default
      primary: deep purple
      accent: amber
      toggle:
        icon: material/brightness-7
        name: Switch to dark mode
    - media: "(prefers-color-scheme: dark)"
      scheme: slate
      primary: deep purple
      accent: amber
      toggle:
        icon: material/brightness-4
        name: Switch to light mode

  font:
    text: Inter
    code: JetBrains Mono

  features:
    - navigation.instant
    - navigation.tracking
    - navigation.tabs
    - navigation.sections
    - navigation.expand
    - navigation.top
    - search.suggest
    - search.highlight
    - content.code.copy
    - content.code.annotate
    - content.action.edit

  icon:
    repo: fontawesome/brands/github
    logo: material/robot

plugins:
  - search
  - mermaid2
  - minify:
      minify_html: true

markdown_extensions:
  - pymdownx.highlight:
      anchor_linenums: true
      line_spans: __span
      pygments_lang_class: true
  - pymdownx.inlinehilite
  - pymdownx.snippets
  - pymdownx.superfences:
      custom_fences:
        - name: mermaid
          class: mermaid
          format: !!python/name:pymdownx.superfences.fence_code_format
  - pymdownx.tabbed:
      alternate_style: true
  - pymdownx.details
  - pymdownx.critic
  - pymdownx.caret
  - pymdownx.keys
  - pymdownx.mark
  - pymdownx.tilde
  - admonition
  - tables
  - attr_list
  - md_in_html
  - def_list
  - footnotes
  - toc:
      permalink: true

nav:
  - Home: index.md
  - Getting Started:
    - Installation: getting-started/installation.md
    - Quick Start: getting-started/quickstart.md
    - First Robot: getting-started/first-robot.md
    - Concepts: getting-started/concepts.md
  - Guides:
    - Components: guides/components.md
    - Services: guides/services.md
    - NATS Messaging: guides/nats.md
    - Configuration: guides/configuration.md
    - Testing: guides/testing.md
  - Examples:
    - Hello Sensor: examples/hello-sensor.md
    - Pan-Tilt Platform: examples/pan-tilt.md
    - Surface Vehicle: examples/skimmer.md
  - Reference:
    - Framework Spec: reference/framework-spec.md
    - API Reference: reference/api.md
    - CLI Reference: reference/cli.md
  - Community:
    - Contributing: community/contributing.md
    - Code of Conduct: community/code-of-conduct.md
    - Support: community/support.md
  - Book: book/index.md

extra:
  social:
    - icon: fontawesome/brands/github
      link: https://github.com/gorai/gorai

  version:
    provider: mike

copyright: Copyright &copy; 2024 The Gorai Authors. Apache 2.0 License.
```

### pkgsite Configuration

pkgsite requires no configuration file—it reads Go module structure and godoc comments directly from source.

**Best Practices for API Documentation:**

1. **Package comments** at the top of each package's primary file:
   ```go
   // Package motor provides motor control interfaces and implementations.
   //
   // Motors are actuators that convert electrical energy into rotational motion.
   // This package defines the Motor interface and provides reference implementations
   // for common motor types including DC motors, stepper motors, and servos.
   package motor
   ```

2. **Function and type documentation**:
   ```go
   // SetPower sets the motor power as a percentage from -1.0 to 1.0.
   // Negative values reverse the motor direction.
   // Returns an error if the motor is not initialized or if power is out of range.
   func (m *DCMotor) SetPower(ctx context.Context, power float64) error {
   ```

3. **Examples in test files**:
   ```go
   func ExampleDCMotor_SetPower() {
       motor, _ := NewDCMotor(Config{Pin: 18})
       motor.SetPower(context.Background(), 0.5) // 50% forward
       // Output:
   }
   ```

---

## Build Workflow

### Building the Container

```bash
# Build the publishing container
podman build -t gorai-publish publish/container/

# Or use the Makefile
make publish-container
```

### Building Documentation

```bash
# Build everything
podman run --rm -v ${PWD}:/workspace:Z gorai-publish all

# Build only the book
podman run --rm -v ${PWD}:/workspace:Z gorai-publish book

# Build only the website
podman run --rm -v ${PWD}:/workspace:Z gorai-publish website
```

### Development with Live Reload

```bash
# Serve the book (port 3000)
podman run --rm -it -p 3000:3000 -v ${PWD}:/workspace:Z gorai-publish book-serve

# Serve the website (port 8000)
podman run --rm -it -p 8000:8000 -v ${PWD}:/workspace:Z gorai-publish website-serve

# Serve the API reference (port 6060)
podman run --rm -it -p 6060:6060 -v ${PWD}:/workspace:Z gorai-publish api-serve
```

### Combined Development

For simultaneous development, use podman-compose or run containers in parallel:

```bash
# Terminal 1: Book
podman run --rm -it -p 3000:3000 -v ${PWD}:/workspace:Z gorai-publish book-serve

# Terminal 2: Website
podman run --rm -it -p 8000:8000 -v ${PWD}:/workspace:Z gorai-publish website-serve

# Terminal 3: API
podman run --rm -it -p 6060:6060 -v ${PWD}:/workspace:Z gorai-publish api-serve
```

---

## Makefile Integration

Add to the project Makefile:

```makefile
# =============================================================================
# Publishing
# =============================================================================

PUBLISH_IMAGE := gorai-publish
PUBLISH_DIR := publish

.PHONY: publish-container
publish-container: ## Build the publishing container
	podman build -t $(PUBLISH_IMAGE) $(PUBLISH_DIR)/container/

.PHONY: publish-all
publish-all: publish-container ## Build all documentation
	podman run --rm -v ${PWD}:/workspace:Z $(PUBLISH_IMAGE) all

.PHONY: publish-book
publish-book: publish-container ## Build the book
	podman run --rm -v ${PWD}:/workspace:Z $(PUBLISH_IMAGE) book

.PHONY: publish-website
publish-website: publish-container ## Build the website
	podman run --rm -v ${PWD}:/workspace:Z $(PUBLISH_IMAGE) website

.PHONY: serve-book
serve-book: publish-container ## Serve book with live reload (port 3000)
	podman run --rm -it -p 3000:3000 -v ${PWD}:/workspace:Z $(PUBLISH_IMAGE) book-serve

.PHONY: serve-website
serve-website: publish-container ## Serve website with live reload (port 8000)
	podman run --rm -it -p 8000:8000 -v ${PWD}:/workspace:Z $(PUBLISH_IMAGE) website-serve

.PHONY: serve-api
serve-api: publish-container ## Serve API reference (port 6060)
	podman run --rm -it -p 6060:6060 -v ${PWD}:/workspace:Z $(PUBLISH_IMAGE) api-serve

.PHONY: publish-clean
publish-clean: ## Clean build outputs
	rm -rf $(PUBLISH_DIR)/dist
```

---

## Deployment

### Static Hosting

The built documentation is static HTML suitable for any hosting platform:

| Platform | Configuration |
|----------|---------------|
| GitHub Pages | Deploy `publish/dist/` to `gh-pages` branch |
| Netlify | Build command: `make publish-all`, publish: `publish/dist/website` |
| Cloudflare Pages | Same as Netlify |
| Self-hosted | Serve `publish/dist/` with nginx, caddy, etc. |

### URL Structure

Recommended URL structure for `gorai.dev`:

| Path | Content | Source |
|------|---------|--------|
| `/` | Website landing page | `publish/dist/website/` |
| `/book/` | The Gorai book | `publish/dist/book/` |
| `/api/` | API reference | `publish/dist/api/` or dynamic pkgsite |

### CI/CD Integration

Example GitHub Actions workflow:

```yaml
name: Publish Documentation

on:
  push:
    branches: [main]
    paths:
      - 'publish/**'
      - 'pkg/**'
      - 'component/**'
      - 'service/**'

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Build publishing container
        run: podman build -t gorai-publish publish/container/

      - name: Build documentation
        run: podman run --rm -v ${PWD}:/workspace gorai-publish all

      - name: Deploy to GitHub Pages
        uses: peaceiris/actions-gh-pages@v3
        with:
          github_token: ${{ secrets.GITHUB_TOKEN }}
          publish_dir: ./publish/dist/website
```

---

## Content Guidelines

### Book Content (mdBook)

- **Progressive complexity**: Start simple, build up
- **Complete examples**: Runnable code, not snippets
- **Cross-references**: Link between chapters
- **Mermaid diagrams**: Use for architecture and flow
- **One concept per section**: Keep focused

### Website Content (MkDocs)

- **Scannable**: Headers, bullets, tables
- **Task-oriented**: "How to..." structure for guides
- **Searchable**: Good keywords and descriptions
- **Current**: Keep in sync with framework changes
- **Links to book**: Reference book chapters for deep dives

### API Documentation (pkgsite)

- **Every exported type**: Document all public APIs
- **Examples**: Include runnable examples in tests
- **Package overview**: Describe package purpose and usage
- **Error conditions**: Document when errors occur
- **Thread safety**: Note concurrency considerations

---

## Migration Plan

### From `book/` to `publish/book/`

1. Create the new directory structure
2. Move content from `book/tmp/` to `publish/book/src/`
3. Create `SUMMARY.md` from chapter files
4. Create `book.toml` configuration
5. Verify build with container
6. Remove legacy `book/` directory

### Timeline

| Phase | Task | Status |
|-------|------|--------|
| 1 | Create `publish/` directory structure | Pending |
| 2 | Create container definition | Pending |
| 3 | Migrate book content | Pending |
| 4 | Create initial website content | Pending |
| 5 | Integrate with Makefile | Pending |
| 6 | Set up CI/CD | Pending |
| 7 | Deploy to gorai.dev | Pending |

---

## References

- [mdBook Documentation](https://rust-lang.github.io/mdBook/)
- [Material for MkDocs](https://squidfunk.github.io/mkdocs-material/)
- [pkgsite](https://pkg.go.dev/golang.org/x/pkgsite)
- [Podman](https://podman.io/)
