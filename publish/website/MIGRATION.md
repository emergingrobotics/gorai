# Hugo Theme Migration Plan: No Theme → Hugo Book

This document provides step-by-step instructions for migrating the Gorai website from the current custom (no theme) setup to Hugo Book, while preserving the ability to easily switch themes later.

## Table of Contents

1. [Overview](#overview)
2. [Prerequisites](#prerequisites)
3. [Migration Strategy](#migration-strategy)
4. [Step-by-Step Instructions](#step-by-step-instructions)
5. [Content Migration](#content-migration)
6. [Configuration Changes](#configuration-changes)
7. [Shortcode Compatibility](#shortcode-compatibility)
8. [Testing](#testing)
9. [Rollback Plan](#rollback-plan)
10. [Switching to Hextra](#switching-to-hextra)

---

## Overview

### Current State
- Custom layouts in `layouts/` directory
- Inline CSS in `baseof.html` (~65 lines)
- Custom shortcodes: `mermaid`, `callout`
- No sidebar navigation, search, or dark mode

### Target State
- Hugo Book theme via git submodule
- Theme-provided layouts (with override capability)
- Sidebar navigation, search, dark mode
- Mermaid diagrams support
- **No npm/Node.js dependencies**
- Easy path to switch themes if needed

### Why Hugo Book?

| Feature | Hugo Book |
|---------|-----------|
| npm/Node.js Required | **No** |
| Hugo Extended Required | Yes |
| Search | Built-in |
| Dark Mode | Yes |
| Sidebar Navigation | Auto-generated |
| Mermaid Diagrams | Supported |
| Multi-language | Yes |
| Maintenance | Active |

### Why Git Submodule (Not Hugo Modules)

We're using **git submodules** instead of Hugo Modules because:

1. **No Go toolchain required** - Simpler setup
2. **Explicit version control** - Theme version pinned in `.gitmodules`
3. **Easier theme switching** - Just change the submodule
4. **Offline development** - Theme files are local
5. **CI/CD simplicity** - `git submodule update --init` is universal

---

## Prerequisites

### Required Software

```bash
# Hugo Extended (required for SCSS compilation)
hugo version
# Must show "extended" - e.g., "hugo v0.146.0+extended"

# Minimum version: 0.146.0
```

### Installing Hugo Extended

**macOS:**
```bash
brew install hugo
# Homebrew installs extended version by default
```

**Ubuntu/Debian:**
```bash
# Snap installs extended version by default
sudo snap install hugo

# Or download from GitHub releases (choose "extended")
# https://github.com/gohugoio/hugo/releases
```

**Windows:**
```bash
# Using Chocolatey
choco install hugo-extended

# Or using Scoop
scoop install hugo-extended
```

**Verify Installation:**
```bash
hugo version
# Should show: hugo v0.146.0+extended linux/amd64 ...
#                              ^^^^^^^^ This is important!
```

### What You DON'T Need

- ❌ Node.js
- ❌ npm
- ❌ PostCSS
- ❌ Go (unless using Hugo Modules)

---

## Migration Strategy

### Directory Structure After Migration

```
publish/website/
├── themes/
│   └── hugo-book/                # Git submodule
├── layouts/
│   └── shortcodes/
│       ├── mermaid.html          # Override for mermaid support
│       └── callout.html          # Maps to Book's hint shortcode
├── assets/
│   └── _custom.scss              # Custom styling (optional)
├── static/
│   └── images/                   # Logos, favicons
├── content/
│   └── docs/                     # Documentation (Book expects this)
├── hugo.toml                     # Main configuration
├── .gitmodules                   # Theme submodule reference
└── MIGRATION.md                  # This file
```

### Key Design Decisions

1. **Simple flat config** - Single `hugo.toml` file (no config directory needed)
2. **Minimal overrides** - Only override what's necessary
3. **Preserve custom shortcodes** - Map to Hugo Book equivalents
4. **Archive old layouts** - Keep in `layouts.archive/` for reference

---

## Step-by-Step Instructions

### Step 1: Create a Migration Branch

```bash
cd /path/to/gorai/publish/website
git checkout -b migrate-to-hugo-book
```

### Step 2: Add Hugo Book as Git Submodule

```bash
# Create themes directory
mkdir -p themes

# Add Hugo Book as submodule
git submodule add https://github.com/alex-shpak/hugo-book.git themes/hugo-book

# Pin to a specific release tag for stability
cd themes/hugo-book
git fetch --tags
git checkout v10  # Or check https://github.com/alex-shpak/hugo-book/releases for latest
cd ../..

# Commit the submodule
git add .gitmodules themes/hugo-book
git commit -m "Add Hugo Book theme as git submodule"
```

### Step 3: Archive Current Custom Layouts

```bash
# Keep old layouts for reference (don't delete yet)
mkdir -p layouts.archive
mv layouts/_default layouts.archive/
mv layouts/index.html layouts.archive/

# Keep shortcodes directory - we'll update the files
# layouts/shortcodes/ stays in place
```

### Step 4: Update Configuration

Replace `hugo.toml` with:

```toml
# Gorai Website Configuration

baseURL = "https://gorai.dev/"
title = "Gorai"
languageCode = "en-us"
defaultContentLanguage = "en"

# Theme
theme = "hugo-book"

# Build settings
enableRobotsTXT = true
enableGitInfo = true
enableEmoji = true

# Disable unused taxonomies
disableKinds = ["taxonomy", "term"]

# Markup configuration
[markup]
  [markup.goldmark]
    [markup.goldmark.renderer]
      unsafe = true  # Allow raw HTML in markdown
  [markup.highlight]
    style = "dracula"
    lineNos = false
    noClasses = false
  [markup.tableOfContents]
    startLevel = 2
    endLevel = 4

# Output formats
[outputs]
  home = ["HTML", "RSS"]
  section = ["HTML", "RSS"]

# Menu configuration
[menu]
  [[menu.before]]
    identifier = "docs"
    name = "Documentation"
    url = "/docs/"
    weight = 10

  [[menu.after]]
    identifier = "github"
    name = "GitHub"
    url = "https://github.com/gorai/gorai"
    weight = 100

# Hugo Book theme parameters
[params]
  # Site description
  description = "A lightweight, Go-based robotics framework built on NATS.io"

  # (Optional) Set the path to a logo for the book
  # BookLogo = "/images/logo.png"

  # Set source repository location
  BookRepo = "https://github.com/gorai/gorai"

  # Enable "Edit this page" links
  BookEditPath = "edit/main/publish/website/content"

  # (Optional) Specify section for docs (defaults to "docs")
  # BookSection = "docs"

  # Table of Contents settings
  BookToC = true

  # (Optional) Set leaf bundle to render as a single page
  # BookSinglePage = false

  # Enable search
  BookSearch = true

  # (Optional) Set this to hide the table of contents
  # BookHiddenTocTree = true

  # Menu style: "flex" or "flat"
  BookMenuBundle = "/menu"

  # Theme color: auto, light, dark
  BookTheme = "auto"

  # (Optional) Configure how dates are displayed
  # BookDateFormat = "January 2, 2006"

  # (Optional) Additional CSS at the bottom
  # BookPortableLinks = true

  # Comments integration (optional)
  # BookComments = false

  # Service Worker for offline support (experimental)
  # BookServiceWorker = false
```

### Step 5: Update Shortcodes for Compatibility

Hugo Book has built-in shortcodes. Update yours to be compatible:

**`layouts/shortcodes/callout.html`** (map to Book's hint shortcode):

```html
{{- $type := .Get "type" | default "info" -}}
{{- $title := .Get "title" | default "" -}}
{{/*
  Map our callout types to Hugo Book hint types:
  - info -> info
  - warning -> warning
  - danger -> danger
  - note -> info
  - tip -> tip (Book specific)
*/}}
<blockquote class="book-hint {{ $type }}">
  {{- if $title }}<strong>{{ $title }}</strong><br>{{ end -}}
  {{ .Inner | markdownify }}
</blockquote>
```

**`layouts/shortcodes/mermaid.html`** (Hugo Book supports mermaid):

```html
{{- $id := .Get "id" | default (printf "mermaid-%d" .Ordinal) -}}
<pre class="mermaid" id="{{ $id }}">
{{ .Inner | safeHTML }}
</pre>
```

Note: Hugo Book has native mermaid support via fenced code blocks:

````markdown
```mermaid
graph LR
    A --> B
```
````

### Step 6: Reorganize Content for Hugo Book

Hugo Book expects documentation in `content/docs/`. Verify your structure:

```
content/
├── _index.md              # Homepage (optional custom landing)
├── docs/
│   ├── _index.md          # Docs section landing
│   ├── getting-started/
│   │   ├── _index.md
│   │   ├── installation.md
│   │   └── quickstart.md
│   ├── guides/
│   │   ├── _index.md
│   │   └── ...
│   └── reference/
│       ├── _index.md
│       └── ...
├── examples/              # Will appear in menu if configured
├── community/
└── book/
```

### Step 7: Update Content Front Matter

Hugo Book uses specific front matter. Update key pages:

**`content/docs/_index.md`**:

```yaml
---
title: "Documentation"
weight: 1
bookFlatSection: true
---
```

**Section `_index.md` files**:

```yaml
---
title: "Getting Started"
weight: 1
bookCollapseSection: true
---
```

**Regular pages**:

```yaml
---
title: "Installation"
weight: 1
---
```

### Step 8: Add Custom Styling (Optional)

Create `assets/_custom.scss` for any custom styles:

```scss
// Gorai custom styles
// This file is automatically included by Hugo Book

// Custom primary color
:root {
  --color-link: #663399;
}

// Any additional customizations
.book-brand {
  // Custom logo styling if needed
}
```

### Step 9: Add Static Assets

```bash
# Create directories
mkdir -p static/images

# Add your assets:
# static/images/logo.png       - Site logo (optional)
# static/images/favicon.png    - Favicon
```

### Step 10: Build and Test

```bash
# Clean old build artifacts
rm -rf public/ resources/

# Build the site
hugo

# Or start dev server
hugo server --buildDrafts

# Check for errors in output
```

### Step 11: Commit the Migration

```bash
git add -A
git commit -m "Migrate to Hugo Book theme

- Add Hugo Book as git submodule
- Update configuration for Hugo Book
- Update shortcodes for compatibility
- Archive old custom layouts
- No npm/Node.js dependencies required

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Content Migration

### Front Matter Reference

| Front Matter | Purpose |
|--------------|---------|
| `title` | Page title |
| `weight` | Sort order (lower = first) |
| `bookFlatSection` | Show section as flat list |
| `bookCollapseSection` | Make section collapsible |
| `bookHidden` | Hide from menu |
| `bookToC` | Show/hide table of contents |
| `bookComments` | Enable/disable comments |
| `bookSearchExclude` | Exclude from search |

### Section Organization

Hugo Book auto-generates sidebar navigation from the directory structure:

1. Each directory should have an `_index.md` file
2. Use `weight` in front matter for ordering
3. Files are sorted by weight, then alphabetically

---

## Shortcode Compatibility

### Callout/Hint

**Before (custom):**
```markdown
{{< callout type="warning" title="Important" >}}
This is a warning message.
{{< /callout >}}
```

**After (Hugo Book native):**
```markdown
{{< hint warning >}}
**Important**
This is a warning message.
{{< /hint >}}
```

Both work with the compatibility shortcode provided.

### Mermaid Diagrams

**Both approaches work:**

Shortcode:
```markdown
{{< mermaid >}}
graph LR
    A --> B
{{< /mermaid >}}
```

Fenced code block (preferred):
````markdown
```mermaid
graph LR
    A --> B
```
````

### Hugo Book Built-in Shortcodes

- `{{< hint [info|warning|danger] >}}` - Callout boxes
- `{{< expand "Title" >}}` - Expandable sections
- `{{< tabs "uniqueid" >}}` - Tabbed content
- `{{< columns >}}` - Multi-column layout
- `{{< button >}}` - Styled buttons
- `{{< katex >}}` - Math equations

---

## Testing

### Verification Checklist

- [ ] Site builds without errors: `hugo`
- [ ] Dev server runs: `hugo server`
- [ ] Homepage loads correctly
- [ ] Documentation sidebar appears
- [ ] Search works (click search icon or press `/`)
- [ ] Dark mode toggle works (click moon/sun icon)
- [ ] Mermaid diagrams render
- [ ] Callout/hint shortcodes work
- [ ] Mobile responsive (resize browser)
- [ ] "Edit this page" links work

### Common Issues

| Issue | Solution |
|-------|----------|
| SCSS errors | Ensure Hugo **Extended** is installed |
| Submodule empty | Run `git submodule update --init` |
| Sidebar not showing | Ensure content is in `content/docs/` |
| Search not working | Check `BookSearch = true` in params |
| Dark mode missing | Check `BookTheme = "auto"` in params |

---

## Rollback Plan

If migration fails, rollback is simple:

```bash
# Restore old layouts
mv layouts.archive/_default layouts/
mv layouts.archive/index.html layouts/

# Remove theme
rm -rf themes/hugo-book
git submodule deinit -f themes/hugo-book

# Restore old config (if you backed it up)
git checkout HEAD -- hugo.toml

# Clean up
rm -rf .gitmodules

# Commit rollback
git checkout main
git branch -D migrate-to-hugo-book
```

---

## Switching to Hextra

If you later decide you want a more modern design, Hextra is another excellent theme that also requires **no npm/Node.js**.

### Why Consider Hextra?

| Feature | Hugo Book | Hextra |
|---------|-----------|--------|
| Design | Classic book style | Modern (Nextra-inspired) |
| npm Required | No | No |
| Search | Built-in | FlexSearch |
| Dark Mode | Yes | Yes |
| Mermaid | Yes | Yes |
| Tailwind CSS | No | Yes (pre-built) |
| Blog Support | Minimal | Full |

### Steps to Switch from Hugo Book to Hextra

#### 1. Remove Hugo Book Submodule

```bash
cd publish/website

# Remove submodule
git submodule deinit -f themes/hugo-book
rm -rf .git/modules/themes/hugo-book
rm -rf themes/hugo-book
git rm -f themes/hugo-book
```

#### 2. Add Hextra Submodule

```bash
# Add Hextra
git submodule add https://github.com/imfing/hextra.git themes/hextra

# Pin to stable version
cd themes/hextra
git fetch --tags
git checkout v0.9.0  # Check releases for latest
cd ../..
```

#### 3. Update Configuration

Replace `hugo.toml`:

```toml
# Gorai Website Configuration (Hextra)

baseURL = "https://gorai.dev/"
title = "Gorai"
languageCode = "en-us"
defaultContentLanguage = "en"

# Theme
theme = "hextra"

# Build settings
enableRobotsTXT = true
enableGitInfo = true
enableEmoji = true

[markup]
  [markup.goldmark]
    [markup.goldmark.renderer]
      unsafe = true
  [markup.highlight]
    noClasses = false

# Hextra theme parameters
[params]
  description = "A lightweight, Go-based robotics framework built on NATS.io"

  [params.navbar]
    displayTitle = true
    displayLogo = false

  [params.footer]
    displayCopyright = true
    displayPoweredBy = false

  [params.editURL]
    enable = true
    base = "https://github.com/gorai/gorai/edit/main/publish/website/content"

[menu]
  [[menu.main]]
    identifier = "docs"
    name = "Docs"
    url = "/docs/"
    weight = 1

  [[menu.main]]
    identifier = "examples"
    name = "Examples"
    url = "/examples/"
    weight = 2

  [[menu.main]]
    identifier = "github"
    name = "GitHub"
    url = "https://github.com/gorai/gorai"
    weight = 100
```

#### 4. Update Shortcodes

Hextra uses different shortcode syntax. Update `layouts/shortcodes/callout.html`:

```html
{{- $type := .Get "type" | default "info" -}}
{{- $title := .Get "title" | default "" -}}
<div class="hextra-callout {{ $type }}">
  {{- if $title }}<strong>{{ $title }}</strong><br>{{ end -}}
  {{ .Inner | markdownify }}
</div>
```

Or use Hextra's native callout:
```markdown
{{< callout type="info" >}}
Your message here
{{< /callout >}}
```

#### 5. Test and Commit

```bash
hugo server --buildDrafts
# Verify everything works

git add -A
git commit -m "Switch theme from Hugo Book to Hextra"
```

---

## Next Steps After Migration

1. **Customize branding** - Add logo, update colors in `_custom.scss`
2. **Review all pages** - Ensure content renders correctly
3. **Set up CI/CD** - GitHub Actions for automated deployment
4. **Add analytics** - If needed (Google Analytics, Plausible, etc.)
5. **Review Hugo Book docs** - https://github.com/alex-shpak/hugo-book

---

## Summary

| Aspect | Before | After |
|--------|--------|-------|
| Theme | None (custom) | Hugo Book |
| Dependencies | Hugo | Hugo Extended only |
| npm/Node.js | Not used | **Not required** |
| Search | None | Built-in |
| Dark Mode | Partial | Full |
| Sidebar | None | Auto-generated |
| Maintenance | High | Low |
