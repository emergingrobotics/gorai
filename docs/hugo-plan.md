# Hugo Website Theme Analysis and Recommendations

## Current State Assessment

### Current Setup

The Gorai website at `publish/website/` uses **Hugo without a theme** — it has a custom, minimal layout built from scratch with inline CSS.

**Configuration (`hugo.toml`):**
```toml
# Theme - using built-in styling for now, can add theme later
# theme = "doks"
```

**Current Implementation:**
- Custom `baseof.html` with ~65 lines of inline CSS
- Simple purple/white color scheme
- Basic responsive layout
- Manual navigation via Hugo menus
- Mermaid and callout shortcodes

### Current Limitations

| Area | Current State | Impact |
|------|---------------|--------|
| **Search** | None | Users cannot find content easily |
| **Navigation** | Flat menu only | No sidebar, no nested docs structure |
| **Dark Mode** | CSS variable defined but not switchable | No user preference |
| **Mobile** | Basic responsive | No hamburger menu, limited mobile UX |
| **Versioning** | None | Cannot document multiple releases |
| **API Reference** | Manual | No auto-generation from Go code |
| **i18n** | None | Single language only |
| **Syntax Highlighting** | Basic Chroma | No copy button, limited languages |
| **Diagrams** | Mermaid shortcode (manual) | Not integrated with theme |
| **Table of Contents** | None visible | Long pages hard to navigate |
| **Edit on GitHub** | None | No easy contribution path |

---

## Theme Recommendations

For a mixed hardware/software robotics project like Gorai, the ideal theme needs:

1. **Excellent documentation structure** - nested sidebar navigation, search, versioning
2. **Code-first features** - syntax highlighting, copy buttons, Go code support
3. **Diagram support** - Mermaid, PlantUML for architecture and hardware diagrams
4. **API reference capability** - ability to integrate or generate from Go code
5. **Active maintenance** - regular updates, good community
6. **Performance** - fast load times, good Lighthouse scores

### Recommendation: **Docsy** (Primary) or **Hextra** (Alternative)

---

## Detailed Theme Comparison

### 1. Docsy (Recommended)

**Repository:** https://github.com/google/docsy
**Demo:** https://www.docsy.dev/
**Stars:** 2,600+
**License:** Apache 2.0

**Why Docsy for Gorai:**
- Used by **Kubernetes**, **gRPC**, **Knative**, **Kubeflow** — proven at scale
- Designed for **medium to large documentation sets** (20+ pages)
- Native **Mermaid and PlantUML** support for diagrams
- **Swagger/OpenAPI** support via shortcode (useful for future REST APIs)
- Built-in **multi-language** support
- **Versioned documentation** capability
- **GitHub integration** - edit links, contribution workflows
- **Search** - Algolia DocSearch, Lunr, or Google Custom Search

**Docsy Features Matrix:**

| Feature | Supported | Notes |
|---------|-----------|-------|
| Sidebar Navigation | ✅ | Auto-generated from content structure |
| Full-text Search | ✅ | Lunr (offline), Algolia, Google CSE |
| Dark Mode | ✅ | Theme toggle |
| Mermaid Diagrams | ✅ | Native integration |
| PlantUML | ✅ | For more complex diagrams |
| LaTeX Math | ✅ | KaTeX or MathJax |
| Syntax Highlighting | ✅ | Prism with copy button |
| API Reference | ✅ | Swagger UI shortcode |
| Versioning | ✅ | Multi-version docs |
| i18n | ✅ | Full multilingual support |
| Blog Section | ✅ | Built-in |
| Community Pages | ✅ | Templates included |
| RSS Feeds | ✅ | Built-in |
| Edit on GitHub | ✅ | Configurable |
| Feedback Widget | ✅ | Was this page helpful? |
| Analytics | ✅ | Google Analytics integration |
| SEO | ✅ | Open Graph, Twitter Cards |

**Requirements:**
- Hugo Extended (SCSS support)
- Go (for Hugo Modules)
- PostCSS + Autoprefixer
- Node.js (for npm packages)

**Potential Downsides:**
- More complex setup than simpler themes
- Requires npm for some features
- Heavier than minimal themes (~500KB CSS)

---

### 2. Hextra (Strong Alternative)

**Repository:** https://github.com/imfing/hextra
**Demo:** https://imfing.github.io/hextra/
**Stars:** 1,700+
**License:** MIT

**Why Hextra might work:**
- **Modern design** inspired by Vercel's Nextra (Next.js docs theme)
- **Tailwind CSS** based — very customizable
- **Batteries-included** but lightweight
- **No Node.js required** for basic use
- **FlexSearch** for offline search
- Built-in **dark mode** toggle
- **Mermaid** support included
- **LLM-friendly** - generates llms.txt for AI tools

**Hextra Features Matrix:**

| Feature | Supported | Notes |
|---------|-----------|-------|
| Sidebar Navigation | ✅ | Auto-generated |
| Full-text Search | ✅ | FlexSearch (offline) |
| Dark Mode | ✅ | Toggle + system preference |
| Mermaid Diagrams | ✅ | Built-in |
| LaTeX Math | ✅ | KaTeX |
| Syntax Highlighting | ✅ | With copy button |
| API Reference | ⚠️ | Manual (no OpenAPI shortcode) |
| Versioning | ⚠️ | Possible but not built-in |
| i18n | ✅ | Hugo multilingual |
| Blog Section | ✅ | Supported |
| Edit on GitHub | ✅ | Configurable |
| SEO | ✅ | Full support |
| Performance | ✅ | 100 Lighthouse scores |

**Potential Downsides:**
- Newer, smaller community than Docsy
- Less battle-tested at enterprise scale
- Fewer ready-made page templates

---

### 3. Doks

**Repository:** https://github.com/gethyas/doks
**Demo:** https://getdoks.org/
**Stars:** 2,300+

**Pros:**
- Excellent Lighthouse scores (100/100)
- FlexSearch built-in
- Dark mode
- Clean, modern design

**Cons:**
- Content and theme historically mixed (though improved)
- Requires specific project structure
- Less flexible than Docsy for large projects

---

### 4. Hugo Book

**Repository:** https://github.com/alex-shpak/hugo-book
**Stars:** 3,000+

**Pros:**
- Very simple and clean
- Zero configuration to start
- Lightweight

**Cons:**
- Too minimal for a complex project like Gorai
- Limited customization
- No built-in API reference support
- Better suited for single-book documentation

---

### 5. Geekdoc

**Repository:** https://github.com/thegeeklab/hugo-geekdoc
**Stars:** 700+

**Pros:**
- Clean design
- Mermaid support
- Easy to customize with CSS variables

**Cons:**
- Smaller community
- Fewer integrations than Docsy
- Less suited for very large documentation sets

---

## Recommendation Summary

### Primary Recommendation: **Docsy**

For a mixed hardware/software robotics project like Gorai, Docsy is the best choice because:

1. **Proven at Scale** - If it works for Kubernetes documentation, it will work for Gorai
2. **Complete Feature Set** - Everything needed out of the box
3. **Diagram Support** - Critical for documenting robot architectures, message flows, hardware connections
4. **API Reference** - Swagger UI shortcode for future REST APIs
5. **Community & Maintenance** - Active development, Google backing
6. **Versioning** - Essential as Gorai matures and has multiple releases
7. **Contribution-Friendly** - Built-in GitHub integration encourages community docs

### Secondary Recommendation: **Hextra**

If Docsy feels too heavyweight or the team prefers:
- Simpler setup (no npm required)
- More modern aesthetic
- Tailwind CSS for easier customization
- Lighter weight deployment

---

## Migration Plan

### Phase 1: Setup (1-2 hours)

1. **Install Prerequisites**
   ```bash
   # Ensure Hugo Extended
   hugo version  # Should show "extended"

   # For Docsy: Install Go, Node.js, PostCSS
   npm install -D autoprefixer postcss postcss-cli
   ```

2. **Initialize Theme as Hugo Module**
   ```bash
   cd publish/website
   hugo mod init github.com/gorai/gorai-website
   ```

3. **Add Theme to `hugo.toml`**
   ```toml
   [module]
     [[module.imports]]
       path = "github.com/google/docsy"
       disable = false
   ```

### Phase 2: Content Migration (2-4 hours)

1. **Restructure Content Directories**
   ```
   content/
   ├── _index.md              # Landing page
   ├── docs/
   │   ├── _index.md          # Docs landing
   │   ├── getting-started/   # Existing
   │   ├── guides/            # Existing
   │   ├── reference/         # Existing
   │   ├── concepts/          # NEW: Architecture, design
   │   └── hardware/          # NEW: Hardware integration guides
   ├── blog/                   # NEW: Project updates
   ├── community/             # Existing
   └── about/                 # NEW: Project info, team
   ```

2. **Update Front Matter**
   - Add `weight` for ordering
   - Add `description` for SEO
   - Add `linkTitle` for sidebar

3. **Convert Shortcodes**
   - `{{< mermaid >}}` → Docsy native Mermaid
   - `{{< callout >}}` → Docsy `alert` shortcode

### Phase 3: Configuration (1-2 hours)

1. **Configure Docsy Parameters**
   ```toml
   [params]
     github_repo = "https://github.com/gorai/gorai"
     github_project_repo = "https://github.com/gorai/gorai"
     github_branch = "main"

     # Enable features
     offlineSearch = true
     prism_syntax_highlighting = true

     # UI configuration
     sidebar_menu_compact = true
     sidebar_menu_foldable = true

     # Links
     [params.links]
       [[params.links.developer]]
         name = "GitHub"
         url = "https://github.com/gorai/gorai"
         icon = "fab fa-github"
   ```

2. **Configure Search**
   - Start with Lunr (offline, no external dependencies)
   - Consider Algolia DocSearch later (free for open source)

3. **Configure Analytics** (optional)
   ```toml
   [services.googleAnalytics]
     id = "G-XXXXXXXXXX"
   ```

### Phase 4: Customization (2-4 hours)

1. **Branding**
   - Logo and favicon
   - Color scheme (purple theme → custom colors)
   - Custom CSS overrides in `assets/scss/_variables_project.scss`

2. **Landing Page**
   - Use Docsy's landing page blocks
   - Feature highlights
   - Quick start section

3. **Navigation**
   - Configure top menu
   - Configure footer links
   - Set up section navigation

### Phase 5: New Features (Ongoing)

1. **API Documentation**
   - Integrate `godoc` or `pkgsite` output
   - Or use Swagger UI for REST APIs

2. **Hardware Guides**
   - Create hardware section with wiring diagrams
   - Add schematic images
   - Document sensor/actuator integration

3. **Version Documentation**
   - Set up versioned docs as releases occur

---

## Content Structure Recommendation

```
content/
├── _index.md                    # Landing page with hero, features
├── docs/
│   ├── _index.md               # Docs overview
│   ├── getting-started/
│   │   ├── _index.md           # Section intro
│   │   ├── installation.md     # Setup instructions
│   │   ├── quickstart.md       # First robot
│   │   └── concepts.md         # Core concepts
│   ├── architecture/
│   │   ├── _index.md
│   │   ├── messaging.md        # NATS messaging
│   │   ├── components.md       # Component model
│   │   └── configuration.md    # Config system
│   ├── hardware/
│   │   ├── _index.md           # Hardware overview
│   │   ├── sensors.md          # Sensor integration
│   │   ├── actuators.md        # Motor/servo control
│   │   ├── cameras.md          # Vision systems
│   │   └── microcontrollers.md # TinyGo targets
│   ├── guides/
│   │   ├── _index.md
│   │   ├── custom-sensors.md
│   │   ├── custom-actuators.md
│   │   ├── services.md
│   │   └── testing.md
│   ├── reference/
│   │   ├── _index.md
│   │   ├── cli.md              # CLI reference
│   │   ├── config.md           # Config file reference
│   │   └── api/                # Generated API docs
│   └── tutorials/
│       ├── _index.md
│       ├── line-follower.md
│       ├── obstacle-avoider.md
│       └── remote-control.md
├── examples/
│   ├── _index.md
│   └── hello-sensor.md
├── blog/
│   ├── _index.md
│   └── posts/                  # Blog posts with dates
├── community/
│   ├── _index.md
│   └── contributing.md
└── about/
    └── _index.md
```

---

## Comparison: Before vs After

| Aspect | Current (No Theme) | After (Docsy) |
|--------|-------------------|---------------|
| Setup Time | Fast | Medium (1-2 hours) |
| Maintenance | High (custom CSS) | Low (theme updates) |
| Search | None | Full-text (Lunr/Algolia) |
| Navigation | Manual menu | Auto-generated sidebar |
| Mobile UX | Basic | Full responsive + hamburger |
| Dark Mode | Partial | Full with toggle |
| Diagrams | Manual shortcode | Native Mermaid + PlantUML |
| API Docs | Manual | Swagger UI shortcode |
| Edit Links | None | Auto GitHub links |
| Versioning | None | Built-in support |
| i18n | None | Full multilingual |
| SEO | Basic | Complete (OG, Twitter, etc.) |
| Analytics | None | Google Analytics ready |
| Community | DIY | Built-in templates |

---

## Decision Matrix

| Criteria | Weight | Docsy | Hextra | Doks | Book |
|----------|--------|-------|--------|------|------|
| Feature Completeness | 25% | 10 | 8 | 7 | 5 |
| Ease of Use | 20% | 7 | 9 | 7 | 10 |
| Customizability | 15% | 9 | 9 | 7 | 6 |
| Performance | 10% | 7 | 10 | 10 | 9 |
| Community/Support | 15% | 10 | 7 | 7 | 8 |
| Scalability | 15% | 10 | 8 | 7 | 5 |
| **Weighted Score** | 100% | **8.65** | **8.35** | **7.25** | **6.65** |

---

## Final Recommendation

**Adopt Docsy** for the Gorai website because:

1. It matches the project's ambition — a comprehensive robotics framework deserves comprehensive documentation
2. Battle-tested by similar projects (Kubernetes, gRPC, Kubeflow)
3. All required features available out of the box
4. Excellent diagram support for hardware/software documentation
5. Built-in versioning for future releases
6. Strong community and long-term maintenance

**Migration effort:** ~8-12 hours for a complete migration with customization.

---

## Sources

- [Docsy Theme](https://www.docsy.dev/)
- [Docsy GitHub](https://github.com/google/docsy)
- [Hextra Theme](https://imfing.github.io/hextra/)
- [Hextra GitHub](https://github.com/imfing/hextra)
- [Doks Theme](https://getdoks.org/)
- [Hugo Documentation Themes Comparison](https://cloudcannon.com/blog/twelve-amazing-free-hugo-documentation-themes/)
- [Kubernetes Documentation (Docsy example)](https://kubernetes.io/docs/)
- [Docsy Examples](https://www.docsy.dev/docs/examples/)
