# Content Migration Plan: Pandoc Book + Hugo Website

**Date:** December 2024
**Status:** Draft
**Purpose:** Migrate from mdBook + MkDocs with symlinks to Pandoc + Hugo with separate, optimized content

---

## 1. Executive Summary

### Current State
- **Book:** mdBook (Rust) with symlinks to shared content
- **Website:** MkDocs Material (Python) with symlinks to shared content
- **Content:** Single source in `publish/content/` (~70 files, ~20,000 lines)
- **Pain points:** Complex symlink management, two different tool ecosystems, content not optimized for either medium

### Target State
- **Book:** Pandoc generating PDF and ePub from dedicated book content
- **Website:** Hugo static site with dedicated web content
- **Content:** Two separate folders, intentionally duplicated where beneficial, each optimized for its reading style
- **Benefits:** Simpler architecture, format-optimized content, single-tool builds, better output quality

### Key Decisions
1. **No symlinks** — Two separate content directories
2. **Intentional duplication** — Optimize content for each format
3. **Shared images** — Single image directory referenced by both
4. **Pandoc for book** — Industry standard, excellent PDF/ePub output
5. **Hugo for website** — Fast, Go-based, excellent documentation themes

---

## 2. Target Directory Structure

```
/gorai/publish/
├── book/                           # Pandoc book project
│   ├── metadata.yaml               # Book metadata (title, author, etc.)
│   ├── chapters/                   # Chapter markdown files
│   │   ├── 00-frontmatter/
│   │   │   ├── title.md
│   │   │   ├── copyright.md
│   │   │   └── preface.md
│   │   ├── 01-introduction.md
│   │   ├── 02-why-gorai.md
│   │   ├── 03-architecture.md
│   │   ├── 04-nats.md
│   │   ├── 05-sensors.md
│   │   ├── 06-actuators.md
│   │   ├── 07-vision.md
│   │   ├── 08-services.md
│   │   ├── 09-behaviors.md
│   │   ├── 10-coordinators.md
│   │   ├── 11-dev-environment.md
│   │   ├── 12-hello-sensor.md
│   │   ├── 13-custom-components.md
│   │   ├── 14-testing.md
│   │   ├── 15-ai-ml.md
│   │   ├── 16-organization.md
│   │   ├── 17-ai-dev.md
│   │   ├── 18-conclusion.md
│   │   └── 99-appendices/
│   │       ├── a-nats-topics.md
│   │       ├── b-protobuf.md
│   │       ├── c-hardware.md
│   │       ├── d-troubleshooting.md
│   │       └── e-glossary.md
│   ├── templates/
│   │   ├── pdf-template.tex        # LaTeX template for PDF
│   │   ├── epub.css                # CSS for ePub
│   │   └── html.css                # CSS for HTML preview
│   ├── filters/
│   │   └── mermaid-filter.lua      # Pandoc filter for diagrams
│   ├── dist/                       # Build output
│   │   ├── gorai-book.pdf
│   │   ├── gorai-book.epub
│   │   └── gorai-book.html
│   ├── Makefile
│   └── README.md
│
├── website/                        # Hugo website project
│   ├── hugo.toml                   # Hugo configuration
│   ├── content/
│   │   ├── _index.md               # Homepage
│   │   ├── docs/
│   │   │   ├── _index.md           # Docs landing
│   │   │   ├── getting-started/
│   │   │   │   ├── _index.md
│   │   │   │   ├── installation.md
│   │   │   │   ├── quickstart.md
│   │   │   │   └── concepts.md
│   │   │   ├── guides/
│   │   │   │   ├── _index.md
│   │   │   │   ├── components.md
│   │   │   │   ├── services.md
│   │   │   │   ├── nats.md
│   │   │   │   ├── configuration.md
│   │   │   │   └── testing.md
│   │   │   └── reference/
│   │   │       ├── _index.md
│   │   │       ├── cli.md
│   │   │       ├── configuration.md
│   │   │       ├── topics.md
│   │   │       └── framework-spec.md
│   │   ├── examples/
│   │   │   ├── _index.md
│   │   │   ├── hello-sensor.md
│   │   │   ├── pan-tilt.md
│   │   │   └── skimmer.md
│   │   ├── community/
│   │   │   ├── _index.md
│   │   │   ├── contributing.md
│   │   │   └── support.md
│   │   └── book/                   # Link to downloadable book
│   │       └── _index.md
│   ├── static/
│   │   ├── downloads/              # PDF/ePub downloads
│   │   └── images/ -> ../../images # Symlink to shared images
│   ├── layouts/
│   │   └── shortcodes/
│   │       ├── mermaid.html
│   │       ├── callout.html
│   │       └── code-file.html
│   ├── themes/
│   │   └── gorai-docs/             # Custom theme or git submodule
│   ├── assets/
│   │   └── scss/
│   ├── public/                     # Build output (gitignored)
│   ├── Makefile
│   └── README.md
│
├── images/                         # Shared images (both use)
│   ├── architecture/
│   ├── diagrams/
│   ├── screenshots/
│   └── logos/
│
├── Makefile                        # Master build orchestration
├── scripts/
│   ├── build-book.sh
│   ├── build-website.sh
│   ├── render-mermaid.sh           # Pre-render Mermaid to SVG
│   └── validate-content.sh
│
└── README.md
```

---

## 3. Book Setup (Pandoc)

### 3.1 Why Pandoc?

| Aspect | Pandoc | mdBook |
|--------|--------|--------|
| PDF output | Excellent (LaTeX) | Requires print.js workaround |
| ePub output | Native, high quality | Not supported |
| Customization | Full control via templates | Limited theme options |
| Ecosystem | Universal, mature | Rust-specific |
| Diagrams | Filters available | Built-in mermaid |
| Dependencies | Pandoc + LaTeX | Rust toolchain |

### 3.2 Book Metadata

**File:** `book/metadata.yaml`

```yaml
---
title: "Gorai"
subtitle: "Building Modern Robots with Go and NATS"
author:
  - Greg Herlein
  - Luca Herlein
date: 2025
lang: en-US
description: |
  A practical guide to building robotics software with Go,
  NATS messaging, and modern AI integration.

# PDF settings
documentclass: book
papersize: letter
fontsize: 11pt
geometry:
  - margin=1in
  - bindingoffset=0.5in
toc: true
toc-depth: 3
numbersections: true
colorlinks: true
linkcolor: purple
urlcolor: blue

# ePub settings
epub-cover-image: ../images/logos/cover.png
epub-metadata: epub-metadata.xml
css: templates/epub.css

# Code highlighting
highlight-style: tango
listings: true
---
```

### 3.3 Chapter Organization

Chapters are numbered for explicit ordering:

| File | Chapter | Content |
|------|---------|---------|
| `00-frontmatter/` | - | Title, copyright, preface |
| `01-introduction.md` | 1 | What is Gorai, who this is for |
| `02-why-gorai.md` | 2 | Landscape, philosophy, audience |
| `03-architecture.md` | 3 | Mental model, core concepts |
| `04-nats.md` | 4 | NATS messaging deep dive |
| `05-sensors.md` | 5 | Sensor interface and types |
| `06-actuators.md` | 6 | Motors, servos, control |
| `07-vision.md` | 7 | Cameras and computer vision |
| `08-services.md` | 8 | Service architecture |
| `09-behaviors.md` | 9 | Behavior-based robotics |
| `10-coordinators.md` | 10 | Orchestrating behaviors |
| `11-dev-environment.md` | 11 | Setting up development |
| `12-hello-sensor.md` | 12 | Tutorial: first sensor |
| `13-custom-components.md` | 13 | Building custom components |
| `14-testing.md` | 14 | Testing strategies |
| `15-ai-ml.md` | 15 | AI/ML integration |
| `16-organization.md` | 16 | Project structure |
| `17-ai-dev.md` | 17 | AI-assisted development |
| `18-conclusion.md` | 18 | What's next |
| `99-appendices/` | A-E | Reference material |

### 3.4 Pandoc Build Commands

**PDF Build:**
```bash
pandoc \
  --defaults=book/defaults.yaml \
  --metadata-file=book/metadata.yaml \
  --template=book/templates/pdf-template.tex \
  --pdf-engine=xelatex \
  --toc \
  --number-sections \
  --highlight-style=tango \
  --lua-filter=book/filters/mermaid-filter.lua \
  -o book/dist/gorai-book.pdf \
  book/chapters/00-frontmatter/*.md \
  book/chapters/[0-1][0-9]-*.md \
  book/chapters/99-appendices/*.md
```

**ePub Build:**
```bash
pandoc \
  --defaults=book/defaults.yaml \
  --metadata-file=book/metadata.yaml \
  --css=book/templates/epub.css \
  --epub-cover-image=images/logos/cover.png \
  --toc \
  --number-sections \
  --lua-filter=book/filters/mermaid-filter.lua \
  -o book/dist/gorai-book.epub \
  book/chapters/00-frontmatter/*.md \
  book/chapters/[0-1][0-9]-*.md \
  book/chapters/99-appendices/*.md
```

### 3.5 Handling Mermaid Diagrams

Pandoc doesn't natively support Mermaid. Options:

**Option A: Pre-render to SVG (Recommended)**
```bash
# scripts/render-mermaid.sh
# Find all .mmd files, render to SVG using mermaid-cli
mmdc -i diagrams/architecture.mmd -o images/diagrams/architecture.svg
```

In markdown, reference as images:
```markdown
![Architecture Overview](../images/diagrams/architecture.svg)
```

**Option B: Lua Filter**
```lua
-- book/filters/mermaid-filter.lua
-- Converts mermaid code blocks to images during build
```

**Option C: Use mermaid-filter package**
```bash
pip install mermaid-filter
pandoc --filter mermaid-filter ...
```

### 3.6 Book Makefile

**File:** `book/Makefile`

```makefile
.PHONY: all pdf epub html clean

CHAPTERS := $(wildcard chapters/00-frontmatter/*.md) \
            $(wildcard chapters/[0-1][0-9]-*.md) \
            $(wildcard chapters/99-appendices/*.md)

all: pdf epub

pdf: dist/gorai-book.pdf

epub: dist/gorai-book.epub

html: dist/gorai-book.html

dist/gorai-book.pdf: $(CHAPTERS) metadata.yaml templates/pdf-template.tex
	@mkdir -p dist
	pandoc \
		--metadata-file=metadata.yaml \
		--template=templates/pdf-template.tex \
		--pdf-engine=xelatex \
		--toc --number-sections \
		--highlight-style=tango \
		-o $@ $(CHAPTERS)

dist/gorai-book.epub: $(CHAPTERS) metadata.yaml templates/epub.css
	@mkdir -p dist
	pandoc \
		--metadata-file=metadata.yaml \
		--css=templates/epub.css \
		--epub-cover-image=../images/logos/cover.png \
		--toc --number-sections \
		-o $@ $(CHAPTERS)

dist/gorai-book.html: $(CHAPTERS) metadata.yaml templates/html.css
	@mkdir -p dist
	pandoc \
		--metadata-file=metadata.yaml \
		--css=templates/html.css \
		--standalone --toc --number-sections \
		-o $@ $(CHAPTERS)

clean:
	rm -rf dist/
```

---

## 4. Website Setup (Hugo)

### 4.1 Why Hugo?

| Aspect | Hugo | MkDocs Material |
|--------|------|-----------------|
| Build speed | Extremely fast (<1s) | Fast (few seconds) |
| Language | Go (single binary) | Python (pip deps) |
| Templating | Go templates | Jinja2 |
| Themes | Extensive ecosystem | Material only |
| Shortcodes | Native, powerful | Limited |
| Taxonomies | Built-in | Plugin |
| Deployment | Single binary | Python environment |

### 4.2 Theme Selection

**Recommended: Docsy or Doks**

| Theme | Pros | Cons |
|-------|------|------|
| **Docsy** | Google-backed, feature-rich, versioning | Heavy, complex setup |
| **Doks** | Modern, fast, SEO-focused | Newer, smaller community |
| **Book** | Simple, clean, book-like | Less features |
| **Geekdoc** | Technical-focused, clean | Simpler styling |

**Recommendation:** Start with **Doks** for simplicity and speed, or **Docsy** for comprehensive documentation features.

### 4.3 Hugo Configuration

**File:** `website/hugo.toml`

```toml
baseURL = "https://gorai.dev/"
title = "Gorai"
languageCode = "en-us"
defaultContentLanguage = "en"

# Theme
theme = "doks"

# Build settings
enableRobotsTXT = true
enableGitInfo = true
enableEmoji = true

# Markup
[markup]
  [markup.goldmark]
    [markup.goldmark.renderer]
      unsafe = true  # Allow raw HTML
  [markup.highlight]
    style = "dracula"
    lineNos = true
    lineNumbersInTable = true

# Menus
[menu]
  [[menu.main]]
    name = "Docs"
    url = "/docs/"
    weight = 10
  [[menu.main]]
    name = "Examples"
    url = "/examples/"
    weight = 20
  [[menu.main]]
    name = "Community"
    url = "/community/"
    weight = 30
  [[menu.main]]
    name = "Book"
    url = "/book/"
    weight = 40

# Parameters
[params]
  description = "A lightweight, Go-based robotics framework built on NATS.io"
  author = "Greg Herlein & Luca Herlein"
  github = "https://github.com/gorai/gorai"

  # Features
  [params.features]
    darkMode = true
    search = true

  # Social
  [params.social]
    github = "gorai/gorai"

# Outputs
[outputs]
  home = ["HTML", "RSS", "JSON"]
  section = ["HTML", "RSS"]
```

### 4.4 Content Front Matter

**Standard front matter for website pages:**

```yaml
---
title: "Getting Started with Gorai"
description: "Install Gorai and build your first robot component in 5 minutes"
weight: 10
draft: false
toc: true
---
```

**Section index pages:**

```yaml
---
title: "Documentation"
description: "Learn how to use Gorai for robotics development"
weight: 10
cascade:
  type: docs
---
```

### 4.5 Hugo Shortcodes

**File:** `website/layouts/shortcodes/mermaid.html`
```html
<div class="mermaid">
{{ .Inner }}
</div>
```

**File:** `website/layouts/shortcodes/callout.html`
```html
<div class="callout callout-{{ .Get "type" | default "info" }}">
  <div class="callout-title">{{ .Get "title" }}</div>
  <div class="callout-content">{{ .Inner | markdownify }}</div>
</div>
```

**Usage in content:**
```markdown
{{</* mermaid */>}}
graph LR
    A[Sensor] --> B[NATS]
    B --> C[Actuator]
{{</* /mermaid */>}}

{{</* callout type="warning" title="Important" */>}}
This requires NATS to be running.
{{</* /callout */>}}
```

### 4.6 Website Makefile

**File:** `website/Makefile`

```makefile
.PHONY: all build serve clean

all: build

build:
	hugo --minify

serve:
	hugo server --buildDrafts --buildFuture

clean:
	rm -rf public/

# Copy book downloads from book build
downloads:
	mkdir -p static/downloads
	cp ../book/dist/gorai-book.pdf static/downloads/
	cp ../book/dist/gorai-book.epub static/downloads/
```

---

## 5. Content Migration Strategy

### 5.1 Content Classification

| Content Type | Book Treatment | Website Treatment |
|--------------|----------------|-------------------|
| **Core concepts** | Narrative chapters | Concept guides |
| **Tutorials** | Step-by-step chapters | Quick-start guides |
| **Reference** | Appendices | Reference section |
| **Examples** | Inline in chapters | Standalone pages |
| **API docs** | Not included | Auto-generated |
| **Community** | Not included | Dedicated section |

### 5.2 Reading Style Optimization

**Book (Lean-back):**
- Sequential flow — each chapter builds on previous
- Complete explanations — assume reader is learning
- Cross-references: "As discussed in Chapter 3..."
- Code examples with full context
- Figures with numbered captions
- No navigation aids (reader proceeds linearly)
- Formal tone, comprehensive coverage

**Website (Lean-forward):**
- Modular pages — each page self-contained
- Quick answers — assume reader is searching
- Cross-references: hyperlinks to related pages
- Copy-paste code snippets
- Inline images without formal captions
- Rich navigation (breadcrumbs, sidebar, search)
- Conversational tone, task-focused

### 5.3 Content Duplication Guidelines

**Always duplicate (different treatment needed):**
- Introduction/overview content
- Tutorials (book: narrative; web: step-by-step)
- Getting started content
- Examples (book: integrated; web: standalone)

**Potentially share (similar treatment):**
- Technical reference tables
- API documentation
- Configuration reference
- Troubleshooting guides

**Website-only:**
- Installation instructions (version-specific)
- Community guidelines
- Contributing guide
- Support resources
- Blog/news (if added)
- API reference (auto-generated)

**Book-only:**
- Preface and acknowledgments
- Extended narratives
- Historical context
- "Why we made this choice" discussions

### 5.4 Migration Mapping

| Current File | Book Destination | Website Destination |
|--------------|------------------|---------------------|
| `introduction.md` | `01-introduction.md` | `docs/_index.md` |
| `part1/.../landscape.md` | `02-why-gorai.md` | `docs/getting-started/concepts.md` |
| `part1/.../philosophy.md` | `02-why-gorai.md` | (merged into concepts) |
| `part1/.../audience.md` | `02-why-gorai.md` | Homepage section |
| `part2/.../ch03-nats/*` | `04-nats.md` | `docs/guides/nats.md` |
| `part2/.../ch04-sensors/*` | `05-sensors.md` | `docs/guides/components.md` |
| `part3/.../ch11-hello-sensor/*` | `12-hello-sensor.md` | `examples/hello-sensor.md` |
| `appendices/glossary.md` | `99-appendices/e-glossary.md` | `docs/reference/glossary.md` |
| `reference/cli.md` | `99-appendices/` | `docs/reference/cli.md` |

### 5.5 Content Transformation Examples

**Current (shared):**
```markdown
# NATS Messaging

NATS is the messaging backbone of Gorai...

## Why NATS?

We chose NATS because...
```

**Book version (narrative):**
```markdown
# NATS: The Messaging Backbone

In the previous chapter, we explored Gorai's architecture and saw how
components communicate through message passing. Now we'll dive deep into
NATS, the messaging system that makes this possible.

By the end of this chapter, you'll understand not just how to use NATS,
but why we chose it over alternatives like ROS 2's DDS or gRPC. This
understanding will help you make better design decisions as you build
your own robot systems.

## Why We Chose NATS

When we began designing Gorai, we evaluated several messaging systems...
[Extended discussion with comparisons, history, decision process]
```

**Website version (task-focused):**
```markdown
---
title: "NATS Messaging"
description: "Learn NATS messaging patterns for Gorai components"
weight: 30
---

# NATS Messaging Guide

Gorai uses NATS for all component communication. This guide covers
the patterns you'll use most often.

## Quick Reference

| Pattern | Use Case | Example |
|---------|----------|---------|
| Pub/Sub | Sensor data | Temperature readings |
| Request/Reply | Commands | Motor control |
| Queue Groups | Load balancing | Multiple processors |

## Common Tasks

### Publishing Sensor Data

```go
// Copy-paste ready example
nc.Publish("sensors.temp.reading", data)
```

[More focused, scannable content]
```

---

## 6. Shared Resources

### 6.1 Images

Images are stored in a shared location and referenced by both:

```
publish/images/
├── architecture/
│   ├── overview.svg
│   ├── message-flow.svg
│   └── component-model.svg
├── diagrams/
│   ├── nats-pubsub.svg
│   ├── sensor-pipeline.svg
│   └── behavior-tree.svg
├── screenshots/
│   ├── cli-output.png
│   └── nats-dashboard.png
└── logos/
    ├── gorai-logo.svg
    ├── gorai-logo.png
    └── cover.png           # Book cover image
```

**Book references:**
```markdown
![Architecture Overview](../images/architecture/overview.svg)
```

**Website references (via symlink or copy):**
```markdown
![Architecture Overview](/images/architecture/overview.svg)
```

### 6.2 Code Examples

Code examples that appear in both can be stored once and included:

```
publish/examples/
├── hello-sensor/
│   ├── main.go
│   ├── sensor.go
│   └── reader.go
├── pan-tilt/
│   └── ...
└── skimmer/
    └── ...
```

**Book (Pandoc):** Use code includes or inline
**Website (Hugo):** Use `readFile` shortcode

---

## 7. Build Infrastructure

### 7.1 Master Makefile

**File:** `publish/Makefile`

```makefile
.PHONY: all book website clean serve-book serve-website

all: book website

# Book targets
book:
	$(MAKE) -C book all

book-pdf:
	$(MAKE) -C book pdf

book-epub:
	$(MAKE) -C book epub

serve-book:
	$(MAKE) -C book html
	python -m http.server 8000 --directory book/dist

# Website targets
website:
	$(MAKE) -C website build

serve-website:
	$(MAKE) -C website serve

# Copy book to website downloads
website-with-book: book website
	cp book/dist/gorai-book.pdf website/static/downloads/
	cp book/dist/gorai-book.epub website/static/downloads/

# Clean
clean:
	$(MAKE) -C book clean
	$(MAKE) -C website clean

# Validation
validate:
	./scripts/validate-content.sh
```

### 7.2 GitHub Actions

**File:** `.github/workflows/publish.yml`

```yaml
name: Build and Deploy Documentation

on:
  push:
    branches: [main]
    paths:
      - 'publish/**'
      - '.github/workflows/publish.yml'
  pull_request:
    paths:
      - 'publish/**'

jobs:
  build-book:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Install Pandoc
        run: |
          sudo apt-get update
          sudo apt-get install -y pandoc texlive-xetex texlive-fonts-recommended

      - name: Build PDF
        run: make -C publish/book pdf

      - name: Build ePub
        run: make -C publish/book epub

      - name: Upload book artifacts
        uses: actions/upload-artifact@v4
        with:
          name: book
          path: publish/book/dist/

  build-website:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Setup Hugo
        uses: peaceiris/actions-hugo@v3
        with:
          hugo-version: 'latest'
          extended: true

      - name: Build website
        run: |
          cd publish/website
          hugo --minify

      - name: Upload website artifact
        uses: actions/upload-artifact@v4
        with:
          name: website
          path: publish/website/public/

  deploy:
    needs: [build-book, build-website]
    runs-on: ubuntu-latest
    if: github.ref == 'refs/heads/main'

    permissions:
      pages: write
      id-token: write

    environment:
      name: github-pages
      url: ${{ steps.deployment.outputs.page_url }}

    steps:
      - name: Download website
        uses: actions/download-artifact@v4
        with:
          name: website
          path: site/

      - name: Download book
        uses: actions/download-artifact@v4
        with:
          name: book
          path: site/downloads/

      - name: Setup Pages
        uses: actions/configure-pages@v4

      - name: Upload to Pages
        uses: actions/upload-pages-artifact@v3
        with:
          path: site/

      - name: Deploy to GitHub Pages
        id: deployment
        uses: actions/deploy-pages@v4
```

### 7.3 Local Development Requirements

**For book development:**
```bash
# macOS
brew install pandoc
brew install --cask mactex  # or basictex for smaller install

# Ubuntu/Debian
sudo apt install pandoc texlive-xetex texlive-fonts-recommended

# Verify
pandoc --version
```

**For website development:**
```bash
# macOS
brew install hugo

# Ubuntu/Debian (snap)
sudo snap install hugo

# Verify
hugo version
```

---

## 8. Migration Phases

### Phase 1: Infrastructure Setup

**Tasks:**
1. Create new directory structure under `publish/`
2. Set up Pandoc book project
   - Create `metadata.yaml`
   - Create basic LaTeX template
   - Create ePub CSS
   - Create Makefile
3. Set up Hugo website project
   - Initialize Hugo site
   - Install/configure theme
   - Create `hugo.toml`
   - Create basic shortcodes
   - Create Makefile
4. Set up shared images directory
5. Create master Makefile
6. Verify both build successfully (with placeholder content)

**Deliverables:**
- Working Pandoc build producing PDF/ePub
- Working Hugo build producing website
- Both builds pass in CI

### Phase 2: Book Content Migration

**Tasks:**
1. Create chapter structure files (empty)
2. Migrate Part I content (Ch 1-3)
   - Merge related files into single chapters
   - Add narrative transitions
   - Adjust cross-references
3. Migrate Part II content (Ch 4-10)
   - Consolidate subsections
   - Optimize for sequential reading
4. Migrate Part III content (Ch 11-14)
   - Adapt tutorials for print format
5. Migrate Part IV content (Ch 15-18)
6. Migrate appendices
7. Create frontmatter (title page, preface, copyright)
8. Review and edit for consistency

**Deliverables:**
- Complete book manuscript in Pandoc format
- PDF and ePub that render correctly
- All figures and code examples working

### Phase 3: Website Content Migration

**Tasks:**
1. Create section structure
2. Write homepage content
3. Migrate Getting Started section
   - Installation (fresh write)
   - Quick Start (adapted from book)
   - Concepts (condensed from book)
4. Migrate Guides section
   - Adapt from book chapters
   - Make task-focused and scannable
5. Migrate Reference section
   - CLI reference
   - Configuration reference
   - Framework spec
6. Migrate Examples section
   - Standalone, copy-paste ready
7. Create Community section
   - Contributing guide
   - Support resources
8. Create Book download page

**Deliverables:**
- Complete Hugo website
- All pages render correctly
- Navigation and search working
- Book PDFs available for download

### Phase 4: Polish and Deploy

**Tasks:**
1. Review book formatting
   - Check PDF pagination
   - Check ePub rendering on devices
   - Fix any LaTeX/CSS issues
2. Review website
   - Check all links
   - Test search functionality
   - Test on mobile devices
3. Update CI/CD pipeline
4. Remove old mdBook/MkDocs infrastructure
5. Update README and documentation
6. Deploy to production

**Deliverables:**
- Production-ready book and website
- CI/CD deploying automatically
- Old infrastructure removed
- Documentation updated

---

## 9. Rollback Plan

If issues arise during migration:

1. **Old system preserved:** Don't delete `publish/content/`, `publish/book/` (mdBook), or `publish/website/` (MkDocs) until new system is fully validated

2. **Git branches:** Work on migration in a feature branch, only merge when complete

3. **Parallel deployment:** Can run old and new systems simultaneously during transition

4. **Quick revert:** If new system fails in production, revert to old CI/CD workflow

---

## 10. Success Criteria

### Book
- [ ] PDF renders correctly with proper formatting
- [ ] ePub validates and renders on major readers (Kindle, Apple Books, etc.)
- [ ] All figures and diagrams display correctly
- [ ] Code examples have proper syntax highlighting
- [ ] Table of contents is accurate and clickable
- [ ] Page numbers and cross-references work

### Website
- [ ] All pages load quickly (<1s)
- [ ] Search returns relevant results
- [ ] Mobile responsive
- [ ] All code blocks have copy buttons
- [ ] Dark mode works correctly
- [ ] Book download links work

### Infrastructure
- [ ] Local builds complete in <30 seconds
- [ ] CI builds complete in <5 minutes
- [ ] Deployment is automatic on merge to main
- [ ] No symlinks required
- [ ] Single-command build for each artifact

---

## 11. Files to Delete After Migration

Once the new system is validated:

```
# Old mdBook setup
publish/book/book.toml
publish/book/src/SUMMARY.md
publish/book/src/README.md
publish/book/theme/

# Old MkDocs setup
publish/website/mkdocs.yml
publish/website/docs/
publish/website/overrides/

# Old shared content
publish/content/

# Old scripts
publish/scripts/setup-book-links.sh
publish/scripts/setup-website-links.sh

# Old container
publish/container/Containerfile
publish/container/entrypoint.sh
```

---

## 12. Open Questions

1. **Hugo theme:** Which theme best fits Gorai's needs?
   - Docsy (feature-rich, complex)
   - Doks (modern, simpler)
   - Custom (most work, most control)

2. **Mermaid handling:** Pre-render to SVG or use runtime rendering?
   - Pre-render: Better for book, requires build step
   - Runtime: Easier for website, doesn't work in PDF

3. **Versioning:** Do we need versioned documentation?
   - If yes, both Docsy and Hugo have solutions
   - Book naturally handles this via edition numbers

4. **Search:** Built-in Hugo search or external (Algolia)?
   - Built-in: Simpler, free
   - Algolia: Better results, requires account

5. **API documentation:** Auto-generate from Go source?
   - pkgsite integration
   - godoc pages
   - Embedded in website or separate

---

## Appendix A: Pandoc Template Basics

**Minimal LaTeX template** (`book/templates/pdf-template.tex`):

```latex
\documentclass[$if(fontsize)$$fontsize$,$endif$$if(papersize)$$papersize$paper,$endif$]{book}

\usepackage{fontspec}
\usepackage{xcolor}
\usepackage{hyperref}
\usepackage{listings}
\usepackage{graphicx}

% Define colors
\definecolor{linkcolor}{RGB}{102, 51, 153}
\definecolor{codebackground}{RGB}{245, 245, 245}

% Hyperlink setup
\hypersetup{
  colorlinks=true,
  linkcolor=linkcolor,
  urlcolor=linkcolor,
}

% Code listing setup
\lstset{
  backgroundcolor=\color{codebackground},
  basicstyle=\ttfamily\small,
  breaklines=true,
  frame=single,
}

\title{$title$}
\author{$for(author)$$author$$sep$ \and $endfor$}
\date{$date$}

\begin{document}

\maketitle

$if(toc)$
\tableofcontents
$endif$

$body$

\end{document}
```

---

## Appendix B: Hugo Shortcode Examples

**Mermaid diagram shortcode:**
```html
<!-- layouts/shortcodes/mermaid.html -->
{{ $id := .Get "id" | default (printf "mermaid-%d" .Ordinal) }}
<div class="mermaid" id="{{ $id }}">
{{ .Inner }}
</div>
```

**Code from file shortcode:**
```html
<!-- layouts/shortcodes/code-file.html -->
{{ $file := .Get "file" }}
{{ $lang := .Get "lang" | default "go" }}
{{ $code := readFile $file }}
```go{{ $lang }}
{{ $code }}
```
```
