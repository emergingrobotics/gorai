# Understanding the Current Gorai Publishing System

**Date:** December 2024
**Purpose:** Document the current publishing implementation before migrating to new tools

---

## Executive Summary

The Gorai project currently uses a **dual-platform publishing system** that serves both a "lean-back" book experience and a "lean-forward" website experience from a **single canonical content source**. The system uses **symbolic links** to avoid content duplication and employs **containerized builds** for reproducibility.

### Current Tools
| Medium | Tool | Configuration |
|--------|------|---------------|
| Book | mdBook (Rust) | `/gorai/publish/book/book.toml` |
| Website | MkDocs Material (Python) | `/gorai/publish/website/mkdocs.yml` |

---

## 1. Directory Structure

```
/gorai/publish/
├── content/              # CANONICAL SOURCE (single source of truth)
│   ├── introduction.md
│   ├── part1-getting-started/
│   ├── part2-core-framework/
│   ├── part3-development/
│   ├── part4-advanced/
│   ├── appendices/
│   ├── reference/
│   └── examples/
│
├── book/                 # mdBook configuration
│   ├── book.toml         # mdBook config
│   ├── src/              # Contains SYMLINKS to content/
│   │   ├── README.md     # Book-specific intro
│   │   ├── SUMMARY.md    # Book table of contents
│   │   ├── introduction.md -> ../../content/introduction.md
│   │   ├── part1-getting-started -> ../../content/part1-getting-started/
│   │   ├── part2-core-framework -> ../../content/part2-core-framework/
│   │   ├── part3-development -> ../../content/part3-development/
│   │   ├── part4-advanced -> ../../content/part4-advanced/
│   │   ├── appendices -> ../../content/appendices/
│   │   ├── reference -> ../../content/reference/
│   │   └── examples -> ../../content/examples/
│   └── theme/            # Custom theme overrides
│
├── website/              # MkDocs configuration
│   ├── mkdocs.yml        # MkDocs config with Material theme
│   ├── docs/             # Contains mix of raw files + SYMLINKS
│   │   ├── index.md      # Website homepage (raw)
│   │   ├── getting-started/  # Website-specific guides (raw)
│   │   ├── guides/           # Website-specific guides (raw)
│   │   ├── community/        # Website-specific (raw)
│   │   ├── book/             # SYMLINKS to content/ parts
│   │   ├── examples/         # Mix of raw + symlinks
│   │   └── reference/        # SYMLINKS to content/reference/
│   └── overrides/        # Material theme customizations
│
├── dist/                 # Build output (gitignored)
│   ├── book/             # mdBook HTML output (~8.1 MB)
│   └── website/          # MkDocs HTML output (~9.5 MB)
│
├── scripts/              # Build automation
│   ├── build-all.sh
│   ├── build-book.sh
│   ├── build-website.sh
│   ├── setup-book-links.sh
│   ├── setup-website-links.sh
│   ├── verify-structure.sh
│   └── migrate-content.sh
│
├── container/            # Docker/Podman build
│   ├── Containerfile
│   └── entrypoint.sh
│
└── Makefile              # Main build targets
```

---

## 2. Content Sharing Strategy

### Single Source Principle

All authoritative content lives in `/gorai/publish/content/`. This directory contains ~20,284 lines of markdown across 118+ files.

### How Content is Shared

**Symbolic links** connect the canonical content to each tool's expected location:

```
content/part1-getting-started/
         ↑                    ↑
    book/src/part1...    website/docs/book/part1...
    (symlink)            (symlink)
```

**Key insight:** A single edit to a file in `content/` automatically appears in both the book and website builds.

### Tool-Specific Content

Some content is unique to each medium:

| Location | Purpose |
|----------|---------|
| `book/src/README.md` | Book-specific introduction |
| `book/src/SUMMARY.md` | Book table of contents (mdBook format) |
| `website/docs/index.md` | Website homepage |
| `website/docs/getting-started/` | Website-specific quick start guides |
| `website/docs/guides/` | Website-specific how-to guides |
| `website/docs/community/` | Contributing, support pages |

### Conditional Content Within Files

HTML comments allow same-file content to differ by platform:
```markdown
<!-- book-only -->
This appears only in the book.
<!-- /book-only -->

<!-- website-only -->
This appears only on the website.
<!-- /website-only -->
```

---

## 3. Current Tools & Configuration

### mdBook (Book)

**Configuration:** `/gorai/publish/book/book.toml`

```toml
[book]
title = "Gorai: Building Modern Robots with Go and NATS"
authors = ["Greg Herlein", "Luca Herlein"]
language = "en"
src = "src"

[build]
build-dir = "../dist/book"

[preprocessor.mermaid]
command = "mdbook-mermaid"

[preprocessor.toc]
command = "mdbook-toc"

[output.html]
default-theme = "light"
preferred-dark-theme = "ayu"
git-repository-url = "https://github.com/gherlein/gorai"
edit-url-template = "https://github.com/gherlein/gorai/edit/main/publish/book/src/{path}"

[output.html.search]
enable = true
limit-results = 30
```

**Features:**
- Sequential chapter navigation (controlled by SUMMARY.md)
- Mermaid diagram support
- Auto-generated table of contents
- Full-text search
- Print-to-PDF support via print.html
- Light/dark theme toggle
- Code copy buttons

### MkDocs Material (Website)

**Configuration:** `/gorai/publish/website/mkdocs.yml`

```yaml
site_name: Gorai
site_url: https://gorai.dev
repo_url: https://github.com/gherlein/gorai
repo_name: gherlein/gorai

theme:
  name: material
  palette:
    - scheme: default
      primary: purple
      accent: amber
      toggle:
        icon: material/brightness-7
        name: Switch to dark mode
    - scheme: slate
      primary: purple
      accent: amber
      toggle:
        icon: material/brightness-4
        name: Switch to light mode
  features:
    - navigation.tabs
    - navigation.sections
    - navigation.expand
    - search.suggest
    - search.highlight
    - content.code.copy
    - content.code.annotate

markdown_extensions:
  - pymdownx.highlight
  - pymdownx.superfences (mermaid support)
  - pymdownx.tabbed
  - pymdownx.details
  - admonition
  - footnotes
  - def_list
  - pymdownx.emoji

nav:
  - Home: index.md
  - Getting Started: [...]
  - Guides: [...]
  - The Book: [...]
  - Examples: [...]
  - Reference: [...]
  - Community: [...]
```

**Features:**
- Material design theme with dark/light toggle
- Tabbed navigation
- Auto-expanding sidebar sections
- Search with suggestions
- Code highlighting with line numbers
- Tabbed content groups
- Admonitions/callouts
- Mermaid diagram support
- Emoji support

### Container Build Environment

**File:** `/gorai/publish/container/Containerfile`

```dockerfile
FROM golang:1.24-bookworm

# Install Rust (for mdBook)
RUN curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y
RUN cargo install mdbook mdbook-mermaid mdbook-toc

# Install Python tools (for MkDocs)
RUN pip install mkdocs-material mkdocs-mermaid2-plugin mkdocs-minify-plugin

# Install pkgsite (for API docs)
RUN go install golang.org/x/pkgsite/cmd/pkgsite@latest
```

**Entry point commands:**
- `book` - Build mdBook
- `book-serve` - Live reload server (port 3000)
- `website` - Build MkDocs
- `website-serve` - Live reload server (port 8000)
- `api-serve` - Go API docs (port 6060)
- `all` - Build both

---

## 4. Build Process

### Makefile Targets

```makefile
# Container builds (recommended)
make container          # Build the publishing container
make docker-all         # Build book + website in container
make docker-book        # Build book only
make docker-web         # Build website only
make docker-serve-book  # Dev server port 3000
make docker-serve-web   # Dev server port 8000

# Local builds (requires tools installed)
make all                # Build both
make book               # Build book only
make website            # Build website only
make serve-book         # Dev server
make serve-web          # Dev server
make clean              # Remove dist/
```

### Build Steps

1. **Setup Symlinks**
   - `scripts/setup-book-links.sh` creates symlinks in `book/src/`
   - `scripts/setup-website-links.sh` creates symlinks in `website/docs/`

2. **Build**
   - mdBook: `cd publish/book && mdbook build`
   - MkDocs: `cd publish/website && mkdocs build --site-dir ../dist/website`

3. **Output**
   - Book: `/gorai/publish/dist/book/` (~8.1 MB)
   - Website: `/gorai/publish/dist/website/` (~9.5 MB)

### CI/CD Pipeline

**File:** `/.github/workflows/docs.yml`

```yaml
on:
  push:
    branches: [main]
    paths: ['publish/**', '.github/workflows/docs.yml']

jobs:
  build-book:
    - Setup mdBook (peaceiris/actions-mdbook@v2)
    - Run setup-book-links.sh
    - mdbook build
    - Upload artifact

  build-website:
    - Setup Python 3.11
    - pip install mkdocs-material
    - Run setup-website-links.sh
    - mkdocs build
    - Upload artifact

  deploy:
    - Download both artifacts
    - Merge: website at /, book at /book/
    - Deploy to GitHub Pages
```

---

## 5. Content State Assessment

### Overall Statistics
- **Total files:** 118+ markdown files
- **Total lines:** ~20,284 lines
- **Content distribution:** 4 parts, 17 chapters, appendices, reference, examples

### State by Section

| Section | Status | Notes |
|---------|--------|-------|
| **Part I: Getting Started** | ✓ Complete | Ch01-02 with full subsections |
| **Part II: Core Framework** | ✓ Mostly Complete | Ch03-09, NATS thoroughly covered |
| **Part III: Development** | ✓ Mostly Complete | Ch10-13, Hello Sensor tutorial done |
| **Part IV: Advanced** | ○ Structural | Ch14-17, overviews written |
| **Appendices** | ✓ Complete | Glossary, troubleshooting, hardware |
| **Reference** | ○ Basic | CLI, config, topics - minimal |
| **Examples** | ○ Present | 3 examples with varying depth |
| **Website-specific** | ✓ Structural | Getting started, guides, community |

### Content Depth

**Well-developed areas:**
- NATS messaging (8 detailed files covering fundamentals, patterns, QoS, JetStream, CLI)
- Sensors (interface, built-in, data types, fakes)
- Actuators (motors, servos, control patterns)
- Vision (cameras, types, data flow)
- Architecture concepts

**Areas with scaffolding:**
- AI/ML integration
- Project organization
- Some advanced topics
- Reference documentation

---

## 6. Key Architecture Decisions

### Why Symbolic Links?
- **Single source of truth:** One edit updates both outputs
- **No sync issues:** Cannot drift out of sync
- **Storage efficient:** No duplicated content
- **Clear ownership:** `content/` is authoritative

### Why Two Tools?
| Need | mdBook | MkDocs |
|------|--------|--------|
| Sequential reading | ✓ Excellent | ○ Possible |
| Quick reference | ○ Adequate | ✓ Excellent |
| Print support | ✓ Built-in | ○ Plugin |
| Material design | ✗ No | ✓ Native |
| Rust ecosystem | ✓ Yes | ✗ No |
| Python ecosystem | ✗ No | ✓ Yes |

### Why Container Builds?
- Reproducible across all environments
- No "works on my machine" issues
- Pinned tool versions
- Simplified CI/CD

---

## 7. File Reference

### Configuration Files
| File | Purpose |
|------|---------|
| `/gorai/publish/book/book.toml` | mdBook configuration |
| `/gorai/publish/website/mkdocs.yml` | MkDocs configuration |
| `/gorai/publish/Makefile` | Build orchestration |
| `/gorai/publish/container/Containerfile` | Container definition |
| `/gorai/.github/workflows/docs.yml` | CI/CD pipeline |

### Build Scripts
| Script | Purpose |
|--------|---------|
| `scripts/build-all.sh` | Orchestrate both builds |
| `scripts/build-book.sh` | Build mdBook |
| `scripts/build-website.sh` | Build MkDocs |
| `scripts/setup-book-links.sh` | Create book symlinks |
| `scripts/setup-website-links.sh` | Create website symlinks |
| `scripts/verify-structure.sh` | Validate setup |

### Canonical Content
| Path | Description |
|------|-------------|
| `content/introduction.md` | Book introduction |
| `content/part1-getting-started/` | Chapters 1-2 |
| `content/part2-core-framework/` | Chapters 3-9 |
| `content/part3-development/` | Chapters 10-13 |
| `content/part4-advanced/` | Chapters 14-17 |
| `content/appendices/` | Reference material |
| `content/reference/` | CLI, config, topics |
| `content/examples/` | Hello sensor, pan-tilt, skimmer |

---

## 8. Deployment Structure

After GitHub Actions deploys to GitHub Pages:

```
https://gorai.dev/
├── /                    # MkDocs website (root)
│   ├── index.html
│   ├── getting-started/
│   ├── guides/
│   ├── community/
│   ├── examples/
│   └── reference/
│
└── /book/               # mdBook nested under website
    ├── index.html
    ├── part1-getting-started/
    ├── part2-core-framework/
    ├── part3-development/
    ├── part4-advanced/
    └── reference/
```

---

## 9. Summary for Migration

### What Works Well
- Single source content model
- Symlink-based sharing
- Container-based builds
- CI/CD automation
- Content structure and organization

### Current Pain Points
- Two different tools with different ecosystems
- Rust dependency for mdBook
- Complex symlink management
- Different markdown feature sets between tools
- Container requires multiple language runtimes

### Content to Preserve
- All ~20,284 lines in `content/`
- Book structure (4 parts, 17 chapters)
- Website navigation structure
- Examples with code
- Appendices and reference material

### Infrastructure to Replace
- mdBook (book.toml, SUMMARY.md, theme/)
- MkDocs (mkdocs.yml, website-specific docs/)
- Container with Rust + Python
- Symlink setup scripts
- Current GitHub Actions workflow
