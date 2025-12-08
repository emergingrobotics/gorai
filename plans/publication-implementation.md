# Publication Implementation Plan

**Implements:** `specs/publication.md`
**Created:** 2024-12-08
**Updated:** 2024-12-08

This plan details the implementation steps for the Gorai publication infrastructure, with a focus on creating both "lean-back" (book) and "lean-forward" (website) experiences from the same content.

---

## Executive Summary

Gorai documentation will be published in two formats optimized for different consumption modes:

| Format | Tool | Experience | Use Case |
|--------|------|------------|----------|
| **Book** | mdBook | Lean-back | Reading on tablet/e-reader, PDF print, sequential learning |
| **Website** | Material for MkDocs | Lean-forward | Quick reference, copy-paste code, search, cross-navigation |

**Key Insight:** Both formats contain the **same depth of information**, but structured differently for their consumption mode. This is achieved through a **single-source architecture** where content is authored once and transformed for each output.

---

## Content Philosophy

### Lean-Back Experience (Book)

The book is optimized for **sequential reading** and **deep understanding**:

- **Linear narrative**: Chapters flow from introduction through advanced topics
- **Complete context**: Each section assumes you've read previous sections
- **Prose-heavy**: Explanations are written for reading, not scanning
- **Embedded examples**: Code is presented with surrounding explanation
- **PDF/ePub friendly**: Works offline, printable, works on e-readers
- **No interactivity required**: All information is self-contained

**User persona**: Reading on a tablet during commute, or printed for study

### Lean-Forward Experience (Website)

The website is optimized for **task completion** and **quick reference**:

- **Non-linear navigation**: Jump directly to what you need
- **Scannable**: Headers, bullets, tables for fast comprehension
- **Copy-paste ready**: Code blocks with copy buttons
- **Heavy cross-linking**: Related topics linked inline
- **Search-first**: Find information by keyword
- **Interactive elements**: Tabs, expandable sections, syntax highlighting
- **API integration**: Direct links to pkgsite documentation

**User persona**: Developer at keyboard, building a robot, needs specific information

---

## Single-Source Architecture

### The Problem

Maintaining two separate copies of documentation leads to:
- Content drift (book says one thing, website says another)
- Double maintenance burden
- Inconsistent examples
- Version skew

### The Solution: Canonical Source with Format-Specific Rendering

```
┌─────────────────────────────────────────────────────────────────┐
│                    CANONICAL SOURCE                              │
│                   publish/content/                               │
│                                                                  │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐          │
│  │ ch01/        │  │ ch02/        │  │ ...          │          │
│  │  intro.md    │  │  overview.md │  │              │          │
│  │  section1.md │  │  concepts.md │  │              │          │
│  │  section2.md │  │  ...         │  │              │          │
│  └──────────────┘  └──────────────┘  └──────────────┘          │
└─────────────────────────────────────────────────────────────────┘
                              │
                              │ Build Process
                              ▼
         ┌────────────────────┴────────────────────┐
         │                                         │
         ▼                                         ▼
┌─────────────────────┐               ┌─────────────────────┐
│      BOOK           │               │      WEBSITE        │
│   publish/book/     │               │  publish/website/   │
│                     │               │                     │
│ - SUMMARY.md        │               │ - mkdocs.yml        │
│ - book.toml         │               │ - nav structure     │
│ - Symlinks/includes │               │ - Symlinks/includes │
│   to content/       │               │   to content/       │
│                     │               │                     │
│ Output: PDF, ePub,  │               │ Output: Interactive │
│ static HTML book    │               │ searchable website  │
└─────────────────────┘               └─────────────────────┘
```

### Implementation Approach

#### Option A: Symbolic Links (Recommended)

Content lives in `publish/content/`. Both `publish/book/src/` and `publish/website/docs/` contain symbolic links to the canonical files.

**Pros:**
- Simple to understand
- No build-time processing
- Edit once, both update
- Standard Unix approach

**Cons:**
- Windows compatibility (though Gorai targets Linux)
- Requires careful path management

#### Option B: Build-Time Copy

A build script copies content from `publish/content/` to both destinations, applying format-specific transformations.

**Pros:**
- Can apply transformations (e.g., different frontmatter)
- No symlink issues
- Can add format-specific content

**Cons:**
- More complex build process
- Must rebuild after content changes

#### Option C: MkDocs Snippets + mdBook Includes

Use each tool's native include mechanism to pull from a shared location.

**Pros:**
- Native to each tool
- Can include partial content
- Fine-grained control

**Cons:**
- Different syntax for each tool
- More complex authoring

### Recommended Approach: Hybrid (Option A + C)

1. **Primary content** uses symbolic links (Option A)
2. **Format-specific additions** use includes (Option C)
3. **Wrapper files** in each destination add format-specific navigation

---

## Directory Structure (Revised)

```
publish/
├── content/                    # CANONICAL SOURCE - all content lives here
│   ├── introduction.md         # Book introduction / Website home content
│   ├── ch01-why-gorai/
│   │   ├── _index.md           # Chapter overview
│   │   ├── landscape.md
│   │   ├── philosophy.md
│   │   ├── audience.md
│   │   ├── whatyoullbuild.md
│   │   └── prerequisites.md
│   ├── ch02-architecture/
│   │   ├── _index.md
│   │   ├── bigpicture.md
│   │   ├── coreconcepts.md
│   │   ├── distributed.md
│   │   ├── config.md
│   │   └── nwsnwc.md
│   ├── ch03-nats/
│   │   └── ...
│   ├── ... (ch04-ch17)
│   ├── appendices/
│   │   └── ...
│   ├── examples/               # Shared example content
│   │   ├── hello-sensor/
│   │   ├── pan-tilt/
│   │   └── skimmer/
│   └── reference/              # Shared reference content
│       ├── cli.md
│       └── framework-spec.md
│
├── book/                       # mdBook configuration
│   ├── book.toml
│   ├── src/
│   │   ├── SUMMARY.md          # Book-specific table of contents
│   │   ├── README.md           # Book introduction (may include content/)
│   │   └── [symlinks to content/]
│   └── theme/                  # Custom theme overrides
│
├── website/                    # MkDocs configuration
│   ├── mkdocs.yml
│   ├── docs/
│   │   ├── index.md            # Website landing page (website-specific)
│   │   ├── getting-started/    # Website-specific quick start
│   │   │   ├── installation.md
│   │   │   └── quickstart.md
│   │   ├── book/               # Symlinks to content/ for full book content
│   │   │   └── [symlinks]
│   │   ├── guides/             # Task-oriented guides (may link to book)
│   │   ├── examples/           # Symlinks to content/examples/
│   │   ├── reference/          # Symlinks to content/reference/
│   │   └── community/          # Website-specific community pages
│   └── overrides/              # Theme customizations
│
├── container/                  # Publishing container
│   ├── Containerfile
│   └── entrypoint.sh
│
├── scripts/                    # Build scripts
│   ├── setup-links.sh          # Create symbolic links
│   ├── build-book.sh           # Build book with transformations
│   └── build-website.sh        # Build website with transformations
│
└── dist/                       # Build output (gitignored)
    ├── book/
    └── website/
```

---

## Content Authoring Guidelines

### Markdown Conventions

All content uses a common markdown format compatible with both mdBook and MkDocs:

```markdown
# Section Title

Brief introduction paragraph.

## Subsection

Content here.

### Code Examples

Always use fenced code blocks with language identifiers:

```go
// Code example
func Example() {
    // ...
}
```

### Diagrams

Use Mermaid for all diagrams (supported by both tools):

```mermaid
graph LR
    A --> B
```

### Admonitions

Use a format compatible with both:

> **Note:** This is important information.

> **Warning:** Be careful about this.

### Links

Use relative links that work in both contexts:

- Same chapter: `[Link](./other-section.md)`
- Other chapter: `[Link](../ch02-architecture/concepts.md)`
- External: `[Link](https://example.com)`
```

### Format-Specific Content

When content must differ between formats, use HTML comments as markers:

```markdown
<!-- book-only -->
This content only appears in the book.
<!-- /book-only -->

<!-- website-only -->
This content only appears on the website.
<!-- /website-only -->
```

Build scripts strip the appropriate sections for each format.

---

## Phase 1: Infrastructure Setup (COMPLETED)

- [x] Create `publish/` directory structure
- [x] Create `publish/container/Containerfile`
- [x] Create `publish/container/entrypoint.sh`
- [x] Create `publish/.gitignore`
- [x] Build and test container
- [x] Add Makefile targets

---

## Phase 2: Canonical Content Structure

**Goal:** Reorganize content into the single-source structure.

### 2.1 Create Content Directory

```bash
mkdir -p publish/content/{ch01-why-gorai,ch02-architecture,ch03-nats,ch04-sensors,ch05-actuators,ch06-vision,ch07-services,ch08-behaviors,ch09-coordinators,ch10-devenv,ch11-hello-sensor,ch12-custom,ch13-testing,ch14-ai-ml,ch15-organization,ch16-ai-dev,ch17-conclusion,appendices,examples,reference}
```

### 2.2 Migrate Existing Content

Move content from `book/tmp/` to `publish/content/`:

| Source | Destination |
|--------|-------------|
| `book/tmp/ch00_introduction.md` | `publish/content/introduction.md` |
| `book/tmp/ch01_s1_landscape.md` | `publish/content/ch01-why-gorai/landscape.md` |
| `book/tmp/ch01_s2_philosophy.md` | `publish/content/ch01-why-gorai/philosophy.md` |
| ... | ... |

### 2.3 Create Chapter Index Files

Each chapter directory needs an `_index.md` that:
- Introduces the chapter
- Lists sections with brief descriptions
- Works as both book chapter intro and website section landing

**Example:** `publish/content/ch01-why-gorai/_index.md`

```markdown
# Why Gorai?

This chapter establishes the motivation, philosophy, and positioning of Gorai in the robotics software landscape.

## In This Chapter

| Section | Description |
|---------|-------------|
| [The Robotics Landscape](landscape.md) | History of ROS, YARP, Viam and their limitations |
| [Design Philosophy](philosophy.md) | Core principles driving Gorai's design |
| [Who Should Use Gorai](audience.md) | Target users and use cases |
| [What You'll Build](whatyoullbuild.md) | Preview of book projects |
| [Prerequisites](prerequisites.md) | Knowledge requirements |

## Key Takeaways

After reading this chapter, you'll understand:
- Why existing robotics frameworks fall short for modern needs
- Gorai's design principles and trade-offs
- Whether Gorai is right for your project
```

### 2.4 Create Shared Examples

Move example content to `publish/content/examples/`:

```
publish/content/examples/
├── hello-sensor/
│   ├── overview.md
│   ├── reader.md
│   ├── sensor.md
│   ├── main.md
│   └── code/           # Actual code files
│       ├── reader.go
│       ├── sensor.go
│       └── main.go
├── pan-tilt/
│   └── ...
└── skimmer/
    └── ...
```

### 2.5 Create Shared Reference

```
publish/content/reference/
├── cli.md
├── configuration.md
├── topic-naming.md
└── proto-reference.md
```

---

## Phase 3: Book Configuration

**Goal:** Configure mdBook to use canonical content.

### 3.1 Update book.toml

```toml
[book]
title = "Gorai: Building Modern Robots with Go and NATS"
authors = ["The Gorai Authors"]
description = "A comprehensive guide to building robots with the Gorai framework"
language = "en"
src = "src"

[build]
build-dir = "../dist/book"
create-missing = false

[output.html]
default-theme = "light"
preferred-dark-theme = "ayu"
git-repository-url = "https://github.com/gorai/gorai"
edit-url-template = "https://github.com/gorai/gorai/edit/main/publish/content/{path}"
site-url = "/book/"
additional-js = ["mermaid.min.js", "mermaid-init.js"]

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

[preprocessor.mermaid]
command = "mdbook-mermaid"
```

### 3.2 Create SUMMARY.md with Full Structure

```markdown
# Summary

[Introduction](README.md)

---

# Getting Started

- [Why Gorai?](ch01-why-gorai/_index.md)
    - [The Robotics Landscape](ch01-why-gorai/landscape.md)
    - [Design Philosophy](ch01-why-gorai/philosophy.md)
    - [Who Should Use Gorai](ch01-why-gorai/audience.md)
    - [What You'll Build](ch01-why-gorai/whatyoullbuild.md)
    - [Prerequisites](ch01-why-gorai/prerequisites.md)

- [Mental Model & Architecture](ch02-architecture/_index.md)
    - [The Big Picture](ch02-architecture/bigpicture.md)
    - [Core Concepts](ch02-architecture/coreconcepts.md)
    - [Distributed Systems](ch02-architecture/distributed.md)
    - [Configuration](ch02-architecture/config.md)
    - [NWS/NWC Pattern](ch02-architecture/nwsnwc.md)

... (full chapter structure)

---

# Appendices

- [NATS Topic Reference](appendices/topics.md)
- [Protocol Buffer Reference](appendices/protobuf.md)
- [Hardware Compatibility](appendices/hardware.md)
- [Troubleshooting](appendices/troubleshooting.md)
- [Glossary](appendices/glossary.md)
```

### 3.3 Create Symbolic Links

**Script:** `publish/scripts/setup-book-links.sh`

```bash
#!/bin/bash
# Create symbolic links from book/src to content/

BOOK_SRC="publish/book/src"
CONTENT="publish/content"

# Link introduction
ln -sf "../../content/introduction.md" "$BOOK_SRC/README.md"

# Link each chapter directory
for chapter in ch01-why-gorai ch02-architecture ch03-nats ch04-sensors \
               ch05-actuators ch06-vision ch07-services ch08-behaviors \
               ch09-coordinators ch10-devenv ch11-hello-sensor ch12-custom \
               ch13-testing ch14-ai-ml ch15-organization ch16-ai-dev \
               ch17-conclusion appendices; do
    ln -sf "../../content/$chapter" "$BOOK_SRC/$chapter"
done

echo "Book symbolic links created"
```

### 3.4 Add PDF/ePub Output

Add to `book.toml`:

```toml
[output.pdf]
# Requires mdbook-pdf
enabled = true

[output.epub]
# Requires mdbook-epub
enabled = true
```

Update Containerfile to include:
```dockerfile
RUN cargo install mdbook-pdf mdbook-epub
```

---

## Phase 4: Website Configuration

**Goal:** Configure MkDocs to use canonical content with enhanced interactivity.

### 4.1 Website-Specific Landing Page

The website has a unique landing page optimized for discovery:

**File:** `publish/website/docs/index.md`

```markdown
# Gorai

**A lightweight, Go-based robotics framework built on NATS.io**

*Pronounced "Go-ray-I" (rhymes with "samurai")*

[Get Started](getting-started/installation.md){ .md-button .md-button--primary }
[Read the Book](book/ch01-why-gorai/){ .md-button }

---

## Quick Links

<div class="grid cards" markdown>

-   :material-rocket-launch:{ .lg .middle } **Quick Start**

    Build your first Gorai node in 5 minutes

    [:octicons-arrow-right-24: Installation](getting-started/installation.md)

-   :material-book-open-variant:{ .lg .middle } **The Book**

    Complete guide from basics to advanced

    [:octicons-arrow-right-24: Start Reading](book/ch01-why-gorai/)

-   :material-api:{ .lg .middle } **API Reference**

    Go package documentation

    [:octicons-arrow-right-24: API Docs](/api/)

-   :material-github:{ .lg .middle } **Examples**

    Working robot projects

    [:octicons-arrow-right-24: Examples](examples/)

</div>

## Why Gorai?

... (comparison table, features list)
```

### 4.2 Website-Specific Quick Start

The website has condensed getting-started content that links to the book for details:

**File:** `publish/website/docs/getting-started/installation.md`

```markdown
# Installation

Get Gorai running in under 5 minutes.

## Prerequisites

- Go 1.21+
- Podman or Docker
- Linux or macOS

## Quick Install

```bash
# Start NATS
podman run -d --name nats -p 4222:4222 nats:latest

# Get Gorai
go get github.com/gorai/gorai
```

## Verify

```bash
go run github.com/gorai/gorai/cmd/gorai version
```

## Next Steps

- [Quick Start →](quickstart.md) Build your first node
- [Full Installation Guide →](../book/ch10-devenv/) Complete setup details
```

### 4.3 Update mkdocs.yml

```yaml
site_name: Gorai
site_url: https://gorai.dev
site_description: A lightweight, Go-based robotics framework built on NATS.io

repo_name: gorai/gorai
repo_url: https://github.com/gorai/gorai
edit_uri: edit/main/publish/content/

theme:
  name: material
  features:
    - navigation.instant
    - navigation.tracking
    - navigation.tabs
    - navigation.tabs.sticky
    - navigation.sections
    - navigation.expand
    - navigation.indexes
    - navigation.top
    - search.suggest
    - search.highlight
    - search.share
    - content.code.copy
    - content.code.annotate
    - content.action.edit
    - content.tabs.link
    - toc.follow

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

plugins:
  - search
  - tags

markdown_extensions:
  - pymdownx.highlight:
      anchor_linenums: true
      line_spans: __span
      pygments_lang_class: true
  - pymdownx.inlinehilite
  - pymdownx.snippets:
      base_path: ["publish/content", "."]
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
      toc_depth: 3

nav:
  - Home: index.md
  - Getting Started:
    - Installation: getting-started/installation.md
    - Quick Start: getting-started/quickstart.md
    - Core Concepts: getting-started/concepts.md
  - Book:
    - book/index.md
    - "Part 1: Getting Started":
      - Why Gorai?: book/ch01-why-gorai/
      - Architecture: book/ch02-architecture/
    - "Part 2: Core Framework":
      - NATS Messaging: book/ch03-nats/
      - Sensors: book/ch04-sensors/
      - Actuators: book/ch05-actuators/
      - Vision & More: book/ch06-vision/
      - Services: book/ch07-services/
      - Behaviors: book/ch08-behaviors/
      - Coordinators: book/ch09-coordinators/
    - "Part 3: Development":
      - Dev Environment: book/ch10-devenv/
      - Hello Sensor: book/ch11-hello-sensor/
      - Custom Components: book/ch12-custom/
      - Testing: book/ch13-testing/
    - "Part 4: Advanced":
      - AI/ML: book/ch14-ai-ml/
      - Organization: book/ch15-organization/
      - AI-Assisted Dev: book/ch16-ai-dev/
      - Conclusion: book/ch17-conclusion/
    - Appendices: book/appendices/
  - Examples:
    - examples/index.md
    - Hello Sensor: examples/hello-sensor/
    - Pan-Tilt Platform: examples/pan-tilt/
    - Surface Vehicle: examples/skimmer/
  - Reference:
    - reference/index.md
    - CLI: reference/cli.md
    - Configuration: reference/configuration.md
    - Topic Naming: reference/topics.md
  - Community:
    - Contributing: community/contributing.md
    - Support: community/support.md
```

### 4.4 Create Symbolic Links for Website

**Script:** `publish/scripts/setup-website-links.sh`

```bash
#!/bin/bash
# Create symbolic links from website/docs to content/

WEBSITE_DOCS="publish/website/docs"
CONTENT="publish/content"

# Link book content
mkdir -p "$WEBSITE_DOCS/book"
for chapter in ch01-why-gorai ch02-architecture ch03-nats ch04-sensors \
               ch05-actuators ch06-vision ch07-services ch08-behaviors \
               ch09-coordinators ch10-devenv ch11-hello-sensor ch12-custom \
               ch13-testing ch14-ai-ml ch15-organization ch16-ai-dev \
               ch17-conclusion appendices; do
    ln -sf "../../../content/$chapter" "$WEBSITE_DOCS/book/$chapter"
done

# Link examples
ln -sf "../../content/examples" "$WEBSITE_DOCS/examples"

# Link reference
ln -sf "../../content/reference" "$WEBSITE_DOCS/reference"

echo "Website symbolic links created"
```

### 4.5 Website-Specific Enhancements

The website adds features not available in the book:

1. **Copy buttons on code blocks** (built into Material theme)
2. **Expandable sections** for lengthy content
3. **Tabbed interfaces** for multi-language examples
4. **Search with suggestions**
5. **Version selector** (for future versions)
6. **Edit on GitHub links**
7. **Tags and categories**

---

## Phase 5: Build Process

**Goal:** Create build scripts that handle both outputs correctly.

### 5.1 Setup Script

**File:** `publish/scripts/setup.sh`

```bash
#!/bin/bash
set -e

echo "Setting up Gorai documentation..."

# Create symbolic links for book
./publish/scripts/setup-book-links.sh

# Create symbolic links for website
./publish/scripts/setup-website-links.sh

echo "Setup complete!"
```

### 5.2 Update Entrypoint

**File:** `publish/container/entrypoint.sh`

```bash
#!/bin/bash
set -e

# Ensure links are set up
if [ ! -L "/workspace/publish/book/src/ch01-why-gorai" ]; then
    echo "Setting up symbolic links..."
    /workspace/publish/scripts/setup.sh
fi

case "$1" in
    book)
        cd /workspace/publish/book
        mdbook-mermaid install .
        mdbook build --dest-dir /workspace/publish/dist/book
        echo "Book built to publish/dist/book/"
        ;;

    book-pdf)
        cd /workspace/publish/book
        mdbook-mermaid install .
        mdbook build --dest-dir /workspace/publish/dist/book
        # PDF generation would go here
        echo "Book PDF built to publish/dist/book.pdf"
        ;;

    website)
        cd /workspace/publish/website
        mkdocs build --site-dir /workspace/publish/dist/website
        echo "Website built to publish/dist/website/"
        ;;

    # ... rest of commands
esac
```

### 5.3 Update Makefile

Add new targets:

```makefile
.PHONY: publish-setup
publish-setup: ## Set up symbolic links for publishing
	./publish/scripts/setup.sh

.PHONY: publish-book-pdf
publish-book-pdf: publish-container publish-setup ## Build book as PDF
	podman run --rm -v $${PWD}:/workspace:Z $(PUBLISH_IMAGE) book-pdf
```

---

## Phase 6: Content Migration

**Goal:** Move all existing content to canonical location.

### 6.1 Migration Mapping

| Source (book/tmp/) | Destination (publish/content/) |
|-------------------|-------------------------------|
| `ch00_introduction.md` | `introduction.md` |
| `ch01_s1_landscape.md` | `ch01-why-gorai/landscape.md` |
| `ch01_s2_philosophy.md` | `ch01-why-gorai/philosophy.md` |
| `ch01_s3_audience.md` | `ch01-why-gorai/audience.md` |
| `ch01_s4_whatyoullbuild.md` | `ch01-why-gorai/whatyoullbuild.md` |
| `ch01_s5_prerequisites.md` | `ch01-why-gorai/prerequisites.md` |
| `ch02_s1_bigpicture.md` | `ch02-architecture/bigpicture.md` |
| `ch02_s2_coreconcepts.md` | `ch02-architecture/coreconcepts.md` |
| `ch02_s3_distributed.md` | `ch02-architecture/distributed.md` |
| `ch02_s4_config.md` | `ch02-architecture/config.md` |
| `ch02_s5_nwsnwc.md` | `ch02-architecture/nwsnwc.md` |
| `ch03_s1_whynats.md` | `ch03-nats/whynats.md` |
| `ch03_s2_fundamentals.md` | `ch03-nats/fundamentals.md` |
| `ch03_s3_patterns.md` | `ch03-nats/patterns.md` |
| `ch03_s4_qos.md` | `ch03-nats/qos.md` |
| `ch03_s5_jetstream.md` | `ch03-nats/jetstream.md` |
| `ch03_s6_cli.md` | `ch03-nats/cli.md` |
| `ch04_s1_interface.md` | `ch04-sensors/interface.md` |
| `ch04_s2_builtin.md` | `ch04-sensors/builtin.md` |
| `ch04_s3_datatypes.md` | `ch04-sensors/datatypes.md` |
| `ch04_s4_fakes.md` | `ch04-sensors/fakes.md` |
| `ch05_s1_actuator.md` | `ch05-actuators/actuator.md` |
| `ch05_s2_motor.md` | `ch05-actuators/motor.md` |
| `ch05_s3_motortypes.md` | `ch05-actuators/motortypes.md` |
| `ch05_s4_control.md` | `ch05-actuators/control.md` |
| `ch05_s5_servo.md` | `ch05-actuators/servo.md` |
| `ch05_s6_base_arm.md` | `ch05-actuators/base_arm.md` |
| `ch06_s1_camera.md` | `ch06-vision/camera.md` |
| `ch06_s2_types.md` | `ch06-vision/types.md` |
| `ch06_s3_dataflow.md` | `ch06-vision/dataflow.md` |
| `ch06_s4_cv.md` | `ch06-vision/cv.md` |
| `ch07_services.md` | `ch07-services/_index.md` |
| `ch08_devenv.md` | `ch10-devenv/_index.md` |
| `ch09_s1_overview.md` | `ch11-hello-sensor/overview.md` |
| `ch09_s2_reader.md` | `ch11-hello-sensor/reader.md` |
| `ch09_s3_sensor.md` | `ch11-hello-sensor/sensor.md` |
| `ch09_s4_main.md` | `ch11-hello-sensor/main.md` |
| `ch10_custom.md` | `ch12-custom/_index.md` |
| `ch11_testing.md` | `ch13-testing/_index.md` |
| `ch12_ml.md` | `ch14-ai-ml/_index.md` |
| `ch13_organization.md` | `ch15-organization/_index.md` |
| `ch14_ai_dev.md` | `ch16-ai-dev/_index.md` |
| `ch15_conclusion.md` | `ch17-conclusion/_index.md` |
| `appendices.md` | `appendices/_index.md` |

**Note:** Chapter numbers shifted due to new chapters (Behaviors, Coordinators).

### 6.2 Create New Chapter Content

New chapters need content:

- `ch08-behaviors/` - Robot Decision Making (new)
- `ch09-coordinators/` - Mission Orchestration (new)

### 6.3 Migration Script

**File:** `publish/scripts/migrate-content.sh`

```bash
#!/bin/bash
# Migrate content from book/tmp to publish/content

SOURCE="book/tmp"
DEST="publish/content"

# Create directory structure
mkdir -p "$DEST"/{ch01-why-gorai,ch02-architecture,ch03-nats,ch04-sensors,ch05-actuators,ch06-vision,ch07-services,ch08-behaviors,ch09-coordinators,ch10-devenv,ch11-hello-sensor,ch12-custom,ch13-testing,ch14-ai-ml,ch15-organization,ch16-ai-dev,ch17-conclusion,appendices,examples,reference}

# Copy and rename files
cp "$SOURCE/ch00_introduction.md" "$DEST/introduction.md"
cp "$SOURCE/ch01_s1_landscape.md" "$DEST/ch01-why-gorai/landscape.md"
# ... (full mapping)

echo "Content migration complete"
```

---

## Phase 7: Verification and Testing

### 7.1 Build Tests

```bash
# Clean build
make publish-clean

# Set up links
make publish-setup

# Build both
make publish-all

# Verify outputs exist
ls -la publish/dist/book/
ls -la publish/dist/website/
```

### 7.2 Link Verification

```bash
# Check for broken links in book
mdbook test publish/book/

# Check for broken links in website
# (MkDocs plugin or external tool)
```

### 7.3 Content Parity Check

Create a script that verifies both outputs contain the same topics:

```bash
# Compare book chapters to website sections
diff <(ls publish/dist/book/) <(ls publish/dist/website/book/)
```

---

## Phase 8: CI/CD Updates

### 8.1 Updated GitHub Actions

```yaml
name: Publish Documentation

on:
  push:
    branches: [main]
    paths:
      - 'publish/**'

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Build container
        run: podman build -t gorai-publish publish/container/

      - name: Setup links
        run: podman run --rm -v ${PWD}:/workspace gorai-publish setup

      - name: Build book
        run: podman run --rm -v ${PWD}:/workspace gorai-publish book

      - name: Build website
        run: podman run --rm -v ${PWD}:/workspace gorai-publish website

      - name: Combine outputs
        run: |
          cp -r publish/dist/website/* publish/dist/combined/
          cp -r publish/dist/book publish/dist/combined/book

      - name: Deploy
        uses: peaceiris/actions-gh-pages@v3
        with:
          github_token: ${{ secrets.GITHUB_TOKEN }}
          publish_dir: ./publish/dist/combined
```

---

## Implementation Checklist

### Phase 2: Canonical Content Structure
- [ ] Create `publish/content/` directory tree
- [ ] Create chapter `_index.md` files
- [ ] Create examples directory structure
- [ ] Create reference directory structure

### Phase 3: Book Configuration
- [ ] Update `book.toml`
- [ ] Create complete `SUMMARY.md`
- [ ] Create `setup-book-links.sh`
- [ ] Test book build with links

### Phase 4: Website Configuration
- [ ] Create website landing page
- [ ] Create getting-started pages
- [ ] Update `mkdocs.yml` with full nav
- [ ] Create `setup-website-links.sh`
- [ ] Test website build with links

### Phase 5: Build Process
- [ ] Create `setup.sh` script
- [ ] Update `entrypoint.sh`
- [ ] Update Makefile targets
- [ ] Test full build process

### Phase 6: Content Migration
- [ ] Run migration script
- [ ] Create new chapter content (behaviors, coordinators)
- [ ] Verify all content migrated
- [ ] Update internal links

### Phase 7: Verification
- [ ] Test book build
- [ ] Test website build
- [ ] Verify link integrity
- [ ] Check content parity

### Phase 8: CI/CD
- [ ] Update GitHub Actions workflow
- [ ] Test deployment

---

## Success Criteria

1. **Single source**: All content lives in `publish/content/`
2. **Book builds**: `make publish-book` produces complete book
3. **Website builds**: `make publish-website` produces complete website
4. **Content parity**: Both outputs cover the same material
5. **No broken links**: All internal links resolve
6. **Format optimization**: Each format is optimized for its consumption mode
7. **Maintainability**: Changes to content automatically appear in both outputs
