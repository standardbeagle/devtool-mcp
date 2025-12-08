# MCP Registry Submission Guide

The MCP servers directory has moved to `registry.modelcontextprotocol.io`. Community servers are now published using the `mcp-publisher` CLI tool.

## Prerequisites

1. **Publish your package first** - The package must be live on npm/PyPI before registry submission
2. **Install mcp-publisher CLI**:
   ```bash
   npm install -g mcp-publisher
   ```

## Submission Steps

### 1. Verify package.json has mcpName

The `mcpName` field should match your registry namespace:
```json
{
  "name": "@standardbeagle/devtool-mcp",
  "mcpName": "io.github.standardbeagle/devtool-mcp"
}
```

### 2. Verify server.json exists

The `server.json` file in the repo root defines your MCP server metadata for the registry.

### 3. Authenticate with GitHub

```bash
mcp-publisher login github
```

This authenticates you for the `io.github.standardbeagle/*` namespace.

### 4. Publish to Registry

```bash
mcp-publisher publish
```

This submits your `server.json` to the MCP Registry.

## Server Metadata

Our `server.json` includes:
- **Name**: `io.github.standardbeagle/devtool-mcp`
- **Packages**: npm (`@standardbeagle/devtool-mcp`) and PyPI (`devtool-mcp`)
- **Transport**: stdio

## MCP Tools Provided

| Tool | Description |
|------|-------------|
| `detect` | Detect project type and available scripts |
| `run` | Execute scripts or raw commands |
| `proc` | Monitor, query output, stop processes |
| `proxy` | Manage reverse proxies with traffic logging |
| `proxylog` | Query HTTP traffic, errors, and performance logs |
| `currentpage` | View grouped page sessions |
| `daemon` | Manage background daemon service |

## Links

- **MCP Registry**: https://registry.modelcontextprotocol.io/
- **Documentation**: https://standardbeagle.github.io/devtool-mcp/
- **Repository**: https://github.com/standardbeagle/devtool-mcp
- **npm**: https://www.npmjs.com/package/@standardbeagle/devtool-mcp
- **PyPI**: https://pypi.org/project/devtool-mcp/
