# Gorai Book (Pandoc)

This directory contains the book source for *Gorai: Building Modern Robots with Go and NATS*.

## Building

### Requirements

- [Pandoc](https://pandoc.org/) 2.x or later
- For PDF: TeX Live with XeLaTeX (`texlive-xetex`, `texlive-fonts-recommended`)

### Installation

**macOS:**
```bash
brew install pandoc
brew install --cask mactex  # or basictex for smaller install
```

**Ubuntu/Debian:**
```bash
sudo apt install pandoc texlive-xetex texlive-fonts-recommended
```

### Build Commands

```bash
# Build PDF and ePub
make all

# Build PDF only
make pdf

# Build ePub only
make epub

# Build HTML preview
make html

# Clean build artifacts
make clean
```

### Output

Built files appear in `dist/`:
- `gorai-book.pdf` — PDF for printing or reading
- `gorai-book.epub` — ePub for e-readers
- `gorai-book.html` — Single-page HTML preview

## Structure

```
book/
├── chapters/
│   ├── 00-frontmatter/     # Preface, acknowledgments
│   ├── 01-introduction.md  # Chapter 1
│   ├── 02-why-gorai.md     # Chapter 2
│   ├── ...                 # More chapters
│   └── 99-appendices/      # Glossary, reference
├── templates/
│   ├── pdf-template.tex    # LaTeX template for PDF
│   ├── epub.css            # Stylesheet for ePub
│   └── html.css            # Stylesheet for HTML preview
├── filters/                # Pandoc Lua filters
├── metadata.yaml           # Book metadata
├── Makefile               # Build automation
└── README.md              # This file
```

## Writing Conventions

- Chapters are numbered (`01-`, `02-`, etc.) for explicit ordering
- Frontmatter uses `00-frontmatter/` prefix
- Appendices use `99-appendices/` prefix
- Use standard Markdown; Pandoc handles the rest
- Images reference `../images/` directory
