# Shared Images

This directory contains images used by both the book and website.

## Structure

```
images/
├── architecture/     # System architecture diagrams
├── diagrams/         # General diagrams (Mermaid exports, etc.)
├── screenshots/      # Application screenshots
└── logos/            # Gorai logos and branding
    └── cover.png     # Book cover image
```

## Usage

### From Book (Pandoc)

Reference images relative to the chapter file:

```markdown
![Architecture](../images/architecture/overview.svg)
```

### From Website (Hugo)

Reference images from static directory:

```markdown
![Architecture](/images/architecture/overview.svg)
```

Note: The website's `static/images` directory should symlink or copy from here.

## Image Guidelines

- Prefer SVG for diagrams (scalable, small file size)
- Use PNG for screenshots
- Keep images under 500KB when possible
- Use descriptive filenames: `component-lifecycle.svg` not `diagram1.svg`
