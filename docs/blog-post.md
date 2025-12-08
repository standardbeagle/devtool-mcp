# Introducing devtool-mcp: Give Your AI Assistant Superpowers for Web Development

**TL;DR**: devtool-mcp is an MCP server that lets AI assistants run your dev server, inspect your frontend, debug layout issues, and audit accessibility - all through natural conversation.

---

## The Problem

You're debugging a CSS layout issue. You describe it to your AI assistant:

> "The sidebar is overlapping the main content on mobile"

The AI suggests fixes, but it's guessing. It can't *see* your app. It can't inspect the actual CSS. It can't check if elements are overflowing or measure the overlap.

Until now.

## Enter devtool-mcp

devtool-mcp is an MCP (Model Context Protocol) server that gives AI assistants direct access to your development environment:

```
You: Start my dev server and set up a debugging proxy

AI: [Runs `npm run dev`, creates proxy on port 8080]
    Dev server running. Proxy ready at http://localhost:8080

You: The sidebar looks broken. What's wrong?

AI: [Executes diagnostic functions in your browser]
    Found 3 issues:
    - .sidebar has `position: fixed` but no `width` constraint
    - Overlap detected: .sidebar covers 45% of .main-content
    - On viewports < 768px, .sidebar extends 120px beyond viewport

    Suggested fix: Add `max-width: 280px` to .sidebar
```

## What It Does

### 1. Project Detection & Process Management

Automatically detects your project type and runs scripts:

```
detect {path: "."}
→ {type: "node", package_manager: "pnpm", scripts: ["dev", "build", "test"]}

run {script_name: "dev"}
→ {process_id: "dev", state: "running", pid: 12345}

proc {action: "output", process_id: "dev", tail: 20}
→ [Last 20 lines of dev server output]
```

### 2. Reverse Proxy with Traffic Logging

Transparent proxy that captures all HTTP traffic and frontend errors:

```
proxy {action: "start", id: "app", target_url: "http://localhost:3000", port: 8080}

proxylog {proxy_id: "app", types: ["error"]}
→ [{message: "TypeError: Cannot read property 'map' of undefined",
    source: "components/List.jsx", line: 42}]
```

### 3. 50+ Frontend Diagnostic Primitives

The proxy injects `window.__devtool` into every page with powerful inspection tools:

```javascript
// Comprehensive element analysis
window.__devtool.inspect('#my-button')
→ {position, box model, layout, stacking context, visibility, accessibility info}

// Find all layout problems
window.__devtool.diagnoseLayout()
→ {overflows: [...], stackingContexts: [...], offscreenElements: [...]}

// Accessibility audit with WCAG scoring
window.__devtool.auditAccessibility()
→ {score: 78, errors: [...], warnings: [...]}

// Interactive element picker
window.__devtool.selectElement()
→ [User clicks element] → {selector: ".card:nth-child(3)", element: ...}

// Check color contrast
window.__devtool.getContrast('.light-text')
→ {ratio: 3.2, passes: {AA: false, AAA: false}}

// Capture screenshot
window.__devtool.screenshot('bug-report')
→ {path: "/tmp/bug-report-1234.png"}
```

## Real-World Use Cases

### Debugging Layout Issues

```
You: Why is there horizontal scroll on mobile?

AI: [Runs checkResponsiveRisk()]
    Found 2 elements causing horizontal overflow:
    - .hero-image: 1200px fixed width, no max-width
    - .data-table: 800px min-width, doesn't fit 375px viewport
```

### Accessibility Auditing

```
You: Is this page accessible?

AI: [Runs auditAccessibility() and runAxeAudit()]
    Accessibility Score: 72/100

    Critical Issues:
    - 3 images missing alt text
    - 2 buttons have no accessible name
    - Color contrast fails on .muted-text (ratio: 2.8, needs 4.5)

    [Highlights problematic elements]
```

### Performance Analysis

```
You: The page feels slow

AI: [Runs auditPageQuality()]
    Performance Grade: C (68/100)

    Issues Found:
    - DOM nodes: 3,200 (recommended: <1,500)
    - Total Blocking Time: 450ms (poor)
    - Largest Contentful Paint: 3.2s (needs improvement)
    - 12 render-blocking resources

    Top recommendation: Virtualize the product list (currently 800 items)
```

### CSS Architecture Review

```
You: Is our CSS well-structured?

AI: [Runs auditCSS()]
    CSS Architecture Score: B (82/100)

    Findings:
    - Specificity: Good distribution, max 0,3,2
    - 45 unique colors (consider design tokens)
    - Tailwind detected: Good utility coverage
    - 3 !important declarations (acceptable)
```

## Installation

**npm:**
```bash
npm install -g @standardbeagle/devtool-mcp
```

**pip/uv:**
```bash
pip install devtool-mcp
```

**One-liner:**
```bash
curl -fsSL https://raw.githubusercontent.com/standardbeagle/devtool-mcp/main/install.sh | bash
```

## Configuration

Add to your MCP client (Claude Code, etc.):

```json
{
  "mcpServers": {
    "devtool": {
      "command": "devtool-mcp"
    }
  }
}
```

## Architecture Highlights

- **Daemon-based**: Processes survive client disconnections
- **Lock-free concurrency**: Uses `sync.Map` and atomics for performance
- **Bounded memory**: Ring buffers prevent unbounded output growth
- **Zero frontend dependencies**: Injected JavaScript works everywhere

## Get Started

- **Documentation**: https://standardbeagle.github.io/devtool-mcp/
- **GitHub**: https://github.com/standardbeagle/devtool-mcp
- **npm**: https://www.npmjs.com/package/@standardbeagle/devtool-mcp
- **PyPI**: https://pypi.org/project/devtool-mcp/

---

devtool-mcp is open source (MIT). Contributions welcome!

*What would you build if your AI could actually see and interact with your running application?*
