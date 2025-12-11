# Gorai Website (Hugo)

This directory contains the website source for [gorai.dev](https://gorai.dev).

## Building

### Requirements

- [Hugo Extended](https://gohugo.io/) (extended version **required** for SCSS)
- [Node.js](https://nodejs.org/) v18+ (for PostCSS)
- Git (with submodule support)

### Installation

**macOS:**
```bash
brew install hugo node
```

**Ubuntu/Debian:**
```bash
sudo snap install hugo
sudo apt install nodejs npm
```

**Verify Hugo Extended:**
```bash
hugo version
# Must show "extended" - e.g., "hugo v0.139.0+extended"
```

### Initial Setup

```bash
# Clone with submodules (if cloning fresh)
git clone --recurse-submodules https://github.com/gorai/gorai.git

# Or if already cloned, initialize submodules
git submodule update --init --recursive

# Install npm dependencies
cd publish/newsite
npm install
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
newsite/
├── themes/
│   └── docsy/              # Theme (git submodule)
├── config/
│   └── _default/
│       ├── hugo.toml       # Main configuration
│       ├── params.toml     # Theme parameters
│       └── menus.toml      # Navigation menus
├── content/
│   ├── _index.md           # Homepage
│   ├── docs/               # Documentation
│   │   ├── getting-started/
│   │   ├── guides/
│   │   └── reference/
│   ├── examples/           # Code examples
│   ├── community/          # Community pages
│   └── book/               # Book download page
├── layouts/
│   └── shortcodes/         # Custom shortcodes (overrides)
├── assets/
│   └── scss/
│       └── _variables_project.scss  # Custom branding
├── static/
│   ├── downloads/          # PDF/ePub files
│   └── images/             # Static images
├── package.json            # npm dependencies
├── Makefile                # Build automation
├── MIGRATION.md            # Migration guide
└── README.md               # This file
```

## Content Guidelines

- Use front matter for title, description, weight
- Add `linkTitle` for shorter navigation labels
- Keep pages focused and scannable
- Use code blocks with language hints
- Link to related pages
- Each directory needs an `_index.md` file

## Theme System

The site uses Git submodules for theme management, making it easy to switch themes while keeping content separate.

### Current Theme

**[Docsy](https://www.docsy.dev/)** - A Hugo theme for technical documentation, used by Kubernetes, gRPC, and other major projects.

---

## Changing Website Theme

This section covers how to change the website theme. The theme is managed as a Git submodule, making it straightforward to switch themes while preserving all content.

### Understanding the Theme Structure

```
themes/
└── docsy/          # Current theme (git submodule)

config/_default/
├── hugo.toml       # Contains: theme = "docsy"
└── params.toml     # Theme-specific parameters

layouts/
└── shortcodes/     # Custom shortcodes (survive theme changes)

assets/scss/
└── _variables_project.scss  # Custom CSS variables
```

### Option A: Switch to a Different Theme

#### Step 1: Remove Current Theme Submodule

```bash
cd publish/newsite

# Remove submodule from .gitmodules and .git/config
git submodule deinit -f themes/docsy

# Remove the submodule directory
rm -rf .git/modules/themes/docsy
rm -rf themes/docsy

# Remove from .gitmodules file
git rm -f themes/docsy
```

#### Step 2: Add New Theme as Submodule

```bash
# Example: Switch to Hextra
git submodule add --depth 1 https://github.com/imfing/hextra.git themes/hextra

# Pin to a specific version
cd themes/hextra
git fetch --tags
git checkout v0.8.0  # Check releases for latest stable
cd ../..

# Example: Switch to Hugo Book
git submodule add --depth 1 https://github.com/alex-shpak/hugo-book.git themes/hugo-book

cd themes/hugo-book
git fetch --tags
git checkout v10  # Check releases for latest stable
cd ../..
```

#### Step 3: Update Configuration

Edit `config/_default/hugo.toml`:

```toml
# Change this line to your new theme
theme = "hextra"  # or "hugo-book", etc.
```

#### Step 4: Update Theme Parameters

Create/update `config/_default/params.toml` with parameters specific to the new theme. Each theme has different configuration options - check the theme's documentation.

#### Step 5: Test and Adjust

```bash
# Build and check for errors
hugo server --buildDrafts

# You may need to:
# - Update shortcodes in layouts/shortcodes/
# - Adjust SCSS variables in assets/scss/
# - Update content front matter
```

#### Step 6: Commit Changes

```bash
git add -A
git commit -m "Switch theme from docsy to hextra"
```

### Option B: Update Current Theme Version

```bash
cd publish/newsite/themes/docsy

# Fetch latest changes
git fetch --tags

# List available versions
git tag -l

# Checkout a new version
git checkout v0.12.0  # Replace with desired version

# Return to main repo
cd ../..

# Commit the update
git add themes/docsy
git commit -m "Update Docsy theme to v0.12.0"
```

### Option C: Use a Theme Fork

If you need to customize a theme significantly:

```bash
# 1. Fork the theme on GitHub

# 2. Remove original submodule
git submodule deinit -f themes/docsy
rm -rf .git/modules/themes/docsy
rm -rf themes/docsy
git rm -f themes/docsy

# 3. Add your fork
git submodule add https://github.com/YOUR-ORG/docsy-fork.git themes/docsy
```

### Theme Compatibility Checklist

When switching themes, verify:

- [ ] **Search works** - Each theme has different search implementation
- [ ] **Navigation renders** - Sidebar, breadcrumbs, menus
- [ ] **Shortcodes work** - May need to update `layouts/shortcodes/`
- [ ] **Dark mode** - If needed, verify it's supported
- [ ] **Mobile responsive** - Test on different screen sizes
- [ ] **Mermaid diagrams** - May need theme-specific config
- [ ] **Build succeeds** - No SCSS/template errors

### Recommended Alternative Themes

| Theme | Best For | GitHub |
|-------|----------|--------|
| **Docsy** (current) | Large documentation sites | [google/docsy](https://github.com/google/docsy) |
| **Hextra** | Modern, lightweight docs | [imfing/hextra](https://github.com/imfing/hextra) |
| **Hugo Book** | Simple book-style docs | [alex-shpak/hugo-book](https://github.com/alex-shpak/hugo-book) |
| **Doks** | Performance-focused | [gethyas/doks](https://github.com/gethyas/doks) |
| **Geekdoc** | Clean, minimal | [thegeeklab/hugo-geekdoc](https://github.com/thegeeklab/hugo-geekdoc) |
| **Relearn** | Feature-rich, Learn fork | [McShelby/hugo-theme-relearn](https://github.com/McShelby/hugo-theme-relearn) |

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

2. **Avoid theme-specific shortcodes** when possible, or create wrapper shortcodes in `layouts/shortcodes/`

3. **Use standard Markdown** - Most themes support GitHub Flavored Markdown

4. **Keep custom CSS minimal** - Only override what's necessary

5. **Document customizations** - Note what's theme-specific in comments

---

## Troubleshooting

### Submodule Issues

```bash
# Submodule is empty
git submodule update --init --recursive

# Submodule is detached HEAD
cd themes/docsy
git checkout v0.11.0  # or desired version
cd ../..

# Reset submodule to clean state
git submodule update --init --force themes/docsy
```

### Build Errors

```bash
# SCSS/PostCSS errors
npm install  # Reinstall dependencies

# Hugo version issues
hugo version  # Must show "extended"

# Clear Hugo cache
rm -rf resources/
hugo --gc
```

### Theme Not Loading

1. Check `config/_default/hugo.toml` has correct `theme = "docsy"` line
2. Verify theme directory exists: `ls themes/docsy`
3. Run `git submodule update --init`

---

## Migration

See [MIGRATION.md](MIGRATION.md) for detailed instructions on migrating from the previous no-theme setup to Docsy.
