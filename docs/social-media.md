# Social Media Announcements

## Twitter/X (Thread)

### Tweet 1 (Main announcement)
```
🚀 Introducing devtool-mcp - an MCP server that gives AI assistants superpowers for web development

Your AI can now:
• Run your dev server
• Inspect DOM elements
• Debug layout issues
• Audit accessibility
• Capture screenshots

All through natural conversation.

🔗 github.com/standardbeagle/devtool-mcp

#MCP #AI #WebDev
```

### Tweet 2 (Demo)
```
"The sidebar is overlapping on mobile"

Before: AI guesses at fixes
After: AI actually inspects your page

→ Detects .sidebar has no width constraint
→ Measures 45% overlap with .main-content
→ Finds 120px viewport overflow
→ Suggests specific fix

No more guessing games.
```

### Tweet 3 (Features)
```
50+ frontend diagnostic primitives:

🔍 inspect() - Full element analysis
📐 diagnoseLayout() - Find overflows & issues
♿ auditAccessibility() - WCAG compliance check
🎨 getContrast() - Color contrast ratio
👆 selectElement() - Interactive picker
📸 screenshot() - Capture evidence

All via MCP.
```

### Tweet 4 (Install)
```
Install in seconds:

npm install -g @standardbeagle/devtool-mcp
# or
pip install devtool-mcp

Add to Claude Code config:
{
  "mcpServers": {
    "devtool": {"command": "devtool-mcp"}
  }
}

Docs: standardbeagle.github.io/devtool-mcp
```

---

## Reddit Post (r/ClaudeAI)

### Title
I built an MCP server that lets Claude actually see and debug your web application

### Body
```
I was tired of describing UI bugs to Claude only to get generic suggestions. So I built devtool-mcp.

**What it does:**

Instead of describing your layout issue, Claude can now:
- Start your dev server (`run {script_name: "dev"}`)
- Set up a debugging proxy that captures all traffic
- Execute 50+ diagnostic functions directly in your browser
- Inspect elements, find overflows, check accessibility, measure contrast ratios

**Example conversation:**

> Me: "Something's wrong with the mobile layout"
>
> Claude: *runs diagnoseLayout()*
> "Found 3 issues: .hero-image has fixed 1200px width causing horizontal scroll, .sidebar overlaps .main-content by 45%, and .nav-menu extends beyond viewport on screens < 768px"

**Features:**
- Project detection (Go/Node/Python)
- Process management with output capture
- Reverse proxy with traffic logging
- Frontend diagnostics (DOM inspection, a11y auditing, CSS analysis)
- Interactive element picker
- Screenshot capture

**Install:**
```
npm install -g @standardbeagle/devtool-mcp
```

GitHub: https://github.com/standardbeagle/devtool-mcp
Docs: https://standardbeagle.github.io/devtool-mcp/

It's open source (MIT). Would love feedback!
```

---

## Reddit Post (r/webdev)

### Title
Open source tool that lets AI assistants debug your frontend by actually inspecting the DOM

### Body
```
Built an MCP server called devtool-mcp that bridges AI assistants with your running web app.

**The problem:** AI can suggest CSS fixes, but it can't see your actual page. It's guessing.

**The solution:** A reverse proxy that injects diagnostic JavaScript into your pages. Your AI can now:

- `inspect('#element')` - Get position, box model, computed styles, stacking context
- `diagnoseLayout()` - Find all overflows, offscreen elements, stacking issues
- `auditAccessibility()` - WCAG compliance check with scoring
- `getContrast('.text')` - Check color contrast ratios
- `checkResponsiveRisk()` - Find elements that'll break on mobile
- `selectElement()` - Interactive picker (click to select)

**50+ primitives total** covering DOM inspection, layout debugging, accessibility, CSS architecture, and security.

Works with Claude Code and other MCP clients.

```bash
npm install -g @standardbeagle/devtool-mcp
# or
pip install devtool-mcp
```

GitHub: https://github.com/standardbeagle/devtool-mcp

Open source, MIT licensed. What other diagnostics would be useful?
```

---

## Hacker News (Show HN)

### Title
Show HN: devtool-mcp – MCP server for AI-assisted web debugging with 50+ DOM primitives

### Body
```
I built devtool-mcp to solve a frustrating problem: AI assistants can suggest code fixes, but they can't actually see your running application.

devtool-mcp is an MCP (Model Context Protocol) server that gives AI assistants direct access to your development environment:

1. **Process Management** - Start dev servers, capture output, manage processes
2. **Reverse Proxy** - Transparent proxy that logs all HTTP traffic and frontend errors
3. **Frontend Diagnostics** - 50+ JavaScript primitives injected into web pages

The diagnostics include:
- Element inspection (position, box model, computed styles, stacking context)
- Layout debugging (find overflows, offscreen elements, responsive breakage)
- Accessibility auditing (WCAG compliance, contrast ratios, screen reader text)
- CSS analysis (specificity distribution, architecture scoring, Tailwind detection)
- Interactive tools (element picker, screenshot capture, user prompts)

Example flow:
```
User: "The sidebar looks broken on mobile"
AI: [runs checkResponsiveRisk()]
    "Found .sidebar has fixed 300px width in a flex container.
     It overflows viewport at 375px.
     Fix: Add max-width: 100% or use clamp()"
```

Technical highlights:
- Written in Go, single binary
- Lock-free concurrency with sync.Map
- Daemon architecture for persistent state
- Zero frontend dependencies (vanilla ES5 JavaScript)

Install: `npm install -g @standardbeagle/devtool-mcp` or `pip install devtool-mcp`

GitHub: https://github.com/standardbeagle/devtool-mcp
Docs: https://standardbeagle.github.io/devtool-mcp/

Would love feedback on what other diagnostics would be useful!
```

---

## LinkedIn Post

```
Excited to announce devtool-mcp - an open source MCP server that transforms how AI assistants help with web development.

The Problem:
When you describe a UI bug to an AI, it's essentially guessing. It can't see your actual application.

The Solution:
devtool-mcp creates a bridge between AI assistants and your running web app. It provides:

✅ Process management - AI can start your dev server
✅ Reverse proxy - Captures all HTTP traffic and errors
✅ 50+ diagnostic primitives - DOM inspection, layout debugging, accessibility auditing

Now instead of:
"Try adding overflow: hidden" (generic guess)

You get:
"Found .sidebar overlaps .main-content by 45% due to missing width constraint. The element extends 120px beyond the viewport on mobile. Specific fix: Add max-width: 280px."

Built with:
• Go (single binary, lock-free concurrency)
• MCP (Model Context Protocol)
• Vanilla JavaScript for browser diagnostics

Available on npm and PyPI. Open source under MIT license.

GitHub: github.com/standardbeagle/devtool-mcp

#OpenSource #WebDevelopment #AI #MCP #DeveloperTools
```

---

## Discord (Anthropic/AI servers)

```
**🛠️ New MCP Server: devtool-mcp**

Just released an MCP server that gives Claude direct access to your dev environment!

**What it does:**
- Runs your dev server and captures output
- Sets up a debugging proxy with traffic logging
- Injects 50+ diagnostic functions into web pages

**Cool stuff you can do:**
```
"Debug the layout" → AI inspects actual DOM, finds overflows
"Is this accessible?" → AI runs WCAG audit, reports score
"Why is this slow?" → AI analyzes DOM complexity, blocking time
```

**Install:**
`npm i -g @standardbeagle/devtool-mcp`

**Links:**
• GitHub: <https://github.com/standardbeagle/devtool-mcp>
• Docs: <https://standardbeagle.github.io/devtool-mcp/>

MIT licensed, feedback welcome! 🙏
```
