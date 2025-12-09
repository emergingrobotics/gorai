# Gorai Website (Hugo)

This directory contains the website source for [gorai.dev](https://gorai.dev).

## Building

### Requirements

- [Hugo](https://gohugo.io/) (extended version recommended)

### Installation

**macOS:**
```bash
brew install hugo
```

**Ubuntu/Debian:**
```bash
sudo snap install hugo
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
│   ├── _default/           # Default templates
│   └── shortcodes/         # Custom shortcodes
├── static/
│   ├── downloads/          # PDF/ePub files
│   └── images/             # Static images
├── assets/                 # SCSS, JS assets
├── hugo.toml               # Hugo configuration
├── Makefile               # Build automation
└── README.md              # This file
```

## Content Guidelines

- Use front matter for title, description, weight
- Keep pages focused and scannable
- Use code blocks with language hints
- Link to related pages
