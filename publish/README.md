# Gorai Publishing

This directory contains the build system for Gorai documentation:

- **Book** — PDF and ePub generated with Pandoc
- **Website** — Hugo static site with the hugo-book theme
- **API Reference** — Go package documentation via pkgsite

All builds run inside a container, so no local tool installation is required beyond Podman (or Docker).

## Quick Start

```bash
# Build the container (one-time setup)
make container

# Build everything
make all

# Or build individually
make book      # PDF + ePub
make website   # Hugo site
```

Output is written to `dist/`.

## Container Tools

The publishing container includes:

| Tool | Version | Purpose |
|------|---------|---------|
| Hugo Extended | 0.146.0 | Static site generator |
| Pandoc | 3.1.11 | Document converter |
| TeX Live | Latest | PDF generation via XeLaTeX |
| ImageMagick | Latest | Image processing |
| GraphViz | Latest | Diagram generation |
| PlantUML | Latest | UML diagrams |
| Go | 1.24 | pkgsite for API docs |

## Make Targets

### Setup

```bash
make container    # Build the publishing container
```

### Book

```bash
make book         # Build PDF and ePub
make book-pdf     # Build PDF only
make book-epub    # Build ePub only
make serve-book   # Preview HTML (port 8000)
```

### Website

```bash
make website      # Build Hugo site
make serve-website # Dev server (port 1313)
```

### Combined

```bash
make all              # Build book + website
make website-with-book # Book + downloads + website
```

### API Documentation

```bash
make serve-api    # Start pkgsite (port 6060)
```

### Deployment

```bash
make publish DEST=user@host:/var/www/gorai
```

### Utility

```bash
make versions     # Show tool versions
make shell        # Interactive shell in container
make clean        # Remove build artifacts
make validate     # Check workspace structure
```

## Directory Structure

```
publish/
├── Makefile            # Main build orchestration
├── container/          # Container definition
│   ├── Containerfile   # Multi-tool container image
│   └── entrypoint.sh   # Command dispatcher
├── book/               # Pandoc book source
│   ├── Makefile        # Standalone book builds
│   ├── metadata.yaml   # Book metadata
│   ├── chapters/       # Markdown chapters
│   └── templates/      # PDF/ePub templates
├── website/            # Hugo website source
│   ├── Makefile        # Standalone website builds
│   ├── hugo.toml       # Hugo configuration
│   ├── content/        # Markdown pages
│   └── themes/         # hugo-book theme (submodule)
└── dist/               # Build output (gitignored)
    ├── book/           # gorai-book.pdf, gorai-book.epub
    └── website/        # Static HTML
```

## Development Workflow

### Editing the Website

```bash
# Start dev server with live reload
make serve-website

# Open http://localhost:1313
# Edit files in website/content/
# Browser refreshes automatically
```

### Editing the Book

```bash
# Build and preview
make serve-book

# Edit files in book/chapters/
# Refresh browser to see changes
```

### Working Without Container

For quick iterations, you can use the tools directly if installed locally:

```bash
# Website (requires hugo 0.146.0+)
cd website && hugo server --buildDrafts

# Book (requires pandoc, texlive-xetex)
cd book && make pdf
```

## Container Details

### Building the Container

```bash
# Using Podman (recommended)
podman build -t gorai-publish container/

# Using Docker
docker build -t gorai-publish container/
```

### Running Commands Directly

```bash
# Podman
podman run --rm -v $(pwd)/..:/workspace:Z gorai-publish website

# Docker
docker run --rm -v $(pwd)/..:/workspace gorai-publish website
```

### Interactive Shell

```bash
make shell
# Now inside container:
hugo version
pandoc --version
```

## Requirements

- **Podman** (recommended) or **Docker**
- **Git** (for theme submodule)
- **Make**

That's it! All other tools (Hugo, Pandoc, TeX Live, etc.) are provided by the container.

## Troubleshooting

### Container Not Found

```bash
# Error: Container image not found
make container  # Build it first
```

### SELinux Issues (Fedora/RHEL)

The Makefile automatically adds `:Z` for Podman on SELinux systems. If you still have issues:

```bash
# Check SELinux status
getenforce

# Temporarily set permissive (for testing)
sudo setenforce 0
```

### Hugo Theme Not Found

```bash
# Initialize submodule
cd website
git submodule update --init --recursive
```

### Port Already in Use

```bash
# Find what's using the port
lsof -i :1313

# Kill it or use a different port
podman run --rm -it -p 8080:1313 -v ...
```
