# MCP Servers Directory Submission

## PR Title
Add devtool-mcp: Development tooling server with process management, proxy, and frontend diagnostics

## Entry to Add (under "Developer Tools" or appropriate section)

```markdown
**[devtool-mcp](https://github.com/standardbeagle/devtool-mcp)** - Development tooling server providing project detection (Go/Node/Python), process management with output capture, reverse proxy with HTTP traffic logging, and 50+ frontend diagnostic primitives for DOM inspection, layout debugging, and accessibility auditing. Enables AI assistants to run dev servers, debug web applications, and analyze frontend issues interactively.
```

## PR Description

### Summary

This PR adds devtool-mcp, a comprehensive development tooling MCP server that bridges AI assistants with development workflows.

### Features

- **Project Detection**: Auto-detect Go, Node.js, and Python projects with available scripts
- **Process Management**: Start, monitor, and control long-running processes with bounded output capture
- **Reverse Proxy**: Transparent HTTP proxy with automatic traffic logging and frontend instrumentation
- **Frontend Diagnostics**: 50+ JavaScript primitives injected into web pages for:
  - DOM inspection and layout analysis
  - Accessibility auditing (WCAG compliance)
  - Visual debugging with overlays
  - Interactive element selection
  - Screenshot capture
  - CSS architecture analysis
  - Security validation

### Installation

```bash
# npm
npm install -g @standardbeagle/devtool-mcp

# pip/uv
pip install devtool-mcp

# From source
git clone https://github.com/standardbeagle/devtool-mcp.git
cd devtool-mcp && make build
```

### Configuration

```json
{
  "mcpServers": {
    "devtool": {
      "command": "devtool-mcp"
    }
  }
}
```

### MCP Tools Provided

| Tool | Description |
|------|-------------|
| `detect` | Detect project type and available scripts |
| `run` | Execute scripts or raw commands |
| `proc` | Monitor, query output, stop processes |
| `proxy` | Manage reverse proxies with traffic logging |
| `proxylog` | Query HTTP traffic, errors, and performance logs |
| `currentpage` | View grouped page sessions |
| `daemon` | Manage background daemon service |

### Links

- **Documentation**: https://standardbeagle.github.io/devtool-mcp/
- **Repository**: https://github.com/standardbeagle/devtool-mcp
- **npm**: https://www.npmjs.com/package/@standardbeagle/devtool-mcp
- **PyPI**: https://pypi.org/project/devtool-mcp/

### License

MIT
