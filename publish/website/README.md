# Gorai Website (Hugo)

This directory contains the website source for [gorai.dev](https://gorai.dev).

## Building

### Requirements

- [Hugo Extended](https://gohugo.io/) v0.146.0 or higher (**extended** version required)
- Git (with submodule support)

**No npm, Node.js, or other dependencies required.**

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
```

**Windows:**
```bash
choco install hugo-extended
# Or: scoop install hugo-extended
```

**Verify Installation:**
```bash
hugo version
# Must show "extended" - e.g., "hugo v0.146.0+extended"
```

### Initial Setup

```bash
# Clone with submodules (if cloning fresh)
git clone --recurse-submodules https://github.com/gorai/gorai.git

# Or if already cloned, initialize submodules
git submodule update --init --recursive

# That's it! No npm install needed.
```

### Build Commands

```bash
# Build the website
make build

# Start development server with live reload
make serve

# Copy book downloads from book build
make downloads

# Clean build artifacts
make clean
```

### Output

Built files appear in `public/` (gitignored).

## Structure

```
website/
├── themes/
│   └── hugo-book/          # Theme (git submodule)
├── content/
│   ├── _index.md           # Homepage
│   ├── docs/               # Documentation (main content)
│   │   ├── getting-started/
│   │   ├── guides/
│   │   └── reference/
│   ├── examples/           # Code examples
│   ├── community/          # Community pages
│   └── book/               # Book download page
├── layouts/
│   └── shortcodes/         # Custom shortcodes (overrides)
├── assets/
│   └── _custom.scss        # Custom styling (optional)
├── static/
│   ├── downloads/          # PDF/ePub files
│   └── images/             # Static images
├── hugo.toml               # Configuration
├── Makefile                # Build automation
├── MIGRATION.md            # Migration guide
└── README.md               # This file
```

## Content Guidelines

- Use front matter for title, description, weight
- Add `weight` for ordering pages in sidebar
- Use `bookCollapseSection: true` for collapsible sections
- Keep pages focused and scannable
- Use code blocks with language hints
- Each directory needs an `_index.md` file

## Theme System

The site uses Git submodules for theme management, making it easy to switch themes while keeping content separate.

### Current Theme

**[Hugo Book](https://github.com/alex-shpak/hugo-book)** - A clean, simple documentation theme.

**Key Features:**
- No npm/Node.js required (Hugo Extended only)
- Auto-generated sidebar navigation
- Built-in search
- Dark mode toggle
- Mermaid diagram support
- Multi-language support

---

## Changing Website Theme

This section covers how to change the website theme. The theme is managed as a Git submodule, making it straightforward to switch themes while preserving all content.

### Understanding the Theme Structure

```
themes/
└── hugo-book/       # Current theme (git submodule)

hugo.toml            # Contains: theme = "hugo-book"

layouts/
└── shortcodes/      # Custom shortcodes (survive theme changes)

assets/
└── _custom.scss     # Custom CSS (optional)
```

### Option A: Switch to Hextra

Hextra is a modern, Tailwind-based theme that also requires **no npm/Node.js**.

#### Step 1: Remove Hugo Book Submodule

```bash
cd publish/website

# Remove submodule from .gitmodules and .git/config
git submodule deinit -f themes/hugo-book

# Remove the submodule directory
rm -rf .git/modules/themes/hugo-book
rm -rf themes/hugo-book

# Remove from .gitmodules file
git rm -f themes/hugo-book
```

#### Step 2: Add Hextra Submodule

```bash
# Add Hextra
git submodule add https://github.com/imfing/hextra.git themes/hextra

# Pin to a specific version
cd themes/hextra
git fetch --tags
git checkout v0.9.0  # Check https://github.com/imfing/hextra/releases for latest
cd ../..
```

#### Step 3: Update Configuration

Replace `hugo.toml` with Hextra configuration:

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

#### Step 4: Test and Commit

```bash
# Test the build
hugo server --buildDrafts

# If everything works, commit
git add -A
git commit -m "Switch theme from Hugo Book to Hextra"
```

### Option B: Switch to Another Theme

The same process works for any Hugo theme:

```bash
# 1. Remove current theme
git submodule deinit -f themes/hugo-book
rm -rf .git/modules/themes/hugo-book themes/hugo-book
git rm -f themes/hugo-book

# 2. Add new theme (examples)
# Relearn (feature-rich, no npm):
git submodule add https://github.com/McShelby/hugo-theme-relearn.git themes/relearn

# Geekdoc (use pre-built release, no npm):
# Download from https://github.com/thegeeklab/hugo-geekdoc/releases
# Extract to themes/hugo-geekdoc/

# 3. Update hugo.toml
# Change: theme = "new-theme-name"

# 4. Update params section for new theme's requirements
```

### Option C: Update Current Theme Version

```bash
cd publish/website/themes/hugo-book

# Fetch latest changes
git fetch --tags

# List available versions
git tag -l

# Checkout a new version
git checkout v11  # Replace with desired version

# Return to main repo
cd ../..

# Commit the update
git add themes/hugo-book
git commit -m "Update Hugo Book theme to v11"
```

### Option D: Use a Theme Fork

If you need significant customizations:

```bash
# 1. Fork the theme on GitHub

# 2. Remove original submodule
git submodule deinit -f themes/hugo-book
rm -rf .git/modules/themes/hugo-book themes/hugo-book
git rm -f themes/hugo-book

# 3. Add your fork
git submodule add https://github.com/YOUR-ORG/hugo-book-fork.git themes/hugo-book
```

### Theme Compatibility Checklist

When switching themes, verify:

- [ ] **Build succeeds** - `hugo` completes without errors
- [ ] **Navigation renders** - Sidebar, menus appear correctly
- [ ] **Search works** - If theme supports search
- [ ] **Dark mode** - If needed, verify it's supported
- [ ] **Shortcodes work** - May need to update `layouts/shortcodes/`
- [ ] **Mermaid diagrams** - May need theme-specific config
- [ ] **Mobile responsive** - Test on different screen sizes

### Recommended Themes (No npm Required)

| Theme | Style | GitHub |
|-------|-------|--------|
| **Hugo Book** (current) | Classic book style | [alex-shpak/hugo-book](https://github.com/alex-shpak/hugo-book) |
| **Hextra** | Modern, Tailwind-based | [imfing/hextra](https://github.com/imfing/hextra) |
| **Relearn** | Feature-rich | [McShelby/hugo-theme-relearn](https://github.com/McShelby/hugo-theme-relearn) |

### Themes That Require npm (Avoid if possible)

| Theme | Note |
|-------|------|
| Docsy | Requires PostCSS via npm |
| Doks | Heavy npm dependencies |
| Geekdoc | Requires npm unless using pre-built release |

### Keeping Content Theme-Agnostic

To make future theme switches easier:

1. **Use standard front matter:**
   ```yaml
   ---
   title: "Page Title"
   description: "Page description"
   weight: 10
   ---
   ```

2. **Avoid theme-specific shortcodes** - Or create wrapper shortcodes in `layouts/shortcodes/`

3. **Use standard Markdown** - Most themes support GitHub Flavored Markdown

4. **Keep custom CSS minimal** - Only override what's necessary

5. **Document customizations** - Note what's theme-specific in comments

---

## Troubleshooting

### Submodule Issues

```bash
# Submodule is empty
git submodule update --init --recursive

# Submodule is in detached HEAD state (normal)
cd themes/hugo-book
git checkout v10  # or desired version
cd ../..

# Reset submodule to clean state
git submodule update --init --force themes/hugo-book
```

### Build Errors

```bash
# SCSS errors - ensure Hugo Extended
hugo version  # Must show "extended"

# Clear Hugo cache
rm -rf resources/ public/
hugo --gc
```

### Theme Not Loading

1. Check `hugo.toml` has correct `theme = "hugo-book"` line
2. Verify theme directory exists: `ls themes/hugo-book`
3. Run `git submodule update --init`

### Sidebar Not Appearing

1. Ensure content is in `content/docs/` directory
2. Each directory needs an `_index.md` file
3. Check `weight` values in front matter for ordering

---

## Dependencies for Publishing

### Required Software

| Dependency | Minimum Version | Purpose |
|------------|-----------------|---------|
| **Hugo Extended** | v0.146.0+ | Static site generator (extended version required for SCSS) |
| **Git** | 2.x | Submodule support for theme management |

**No npm, Node.js, PostCSS, or other JavaScript tooling required.**

### Installing Hugo Extended

#### Linux (Debian/Ubuntu)

```bash
# Download the .deb package (check https://github.com/gohugoio/hugo/releases for latest)
wget https://github.com/gohugoio/hugo/releases/download/v0.152.2/hugo_extended_0.152.2_linux-amd64.deb

# Install
sudo dpkg -i hugo_extended_0.152.2_linux-amd64.deb

# Verify (must show "extended")
hugo version
```

#### Linux (Snap - easiest)

```bash
sudo snap install hugo
hugo version
```

#### macOS

```bash
brew install hugo
hugo version
```

#### Windows

```bash
choco install hugo-extended
# Or: scoop install hugo-extended
```

### Initializing Git Submodules

After cloning the repository, initialize the theme submodule:

```bash
# If cloning fresh
git clone --recurse-submodules https://github.com/gorai/gorai.git

# If already cloned
git submodule update --init --recursive

# If submodule has issues, try
git submodule sync
git submodule update --init --recursive --force
```

### Verifying Your Setup

```bash
# Check Hugo version (must be 0.146.0+ extended)
hugo version

# Check submodule is populated
ls publish/website/themes/hugo-book/

# Test build
cd publish/website
hugo

# Test local server
hugo server
```

---

## Migration

See [MIGRATION.md](MIGRATION.md) for detailed instructions on migrating from the previous no-theme setup to Hugo Book.
