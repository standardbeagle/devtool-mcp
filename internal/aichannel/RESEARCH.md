# AI Coding Agents CLI Research

## Summary of Stdin/Stdout Support

| Agent | Command | Stdin Input | Non-Interactive Flag | Output Format |
|-------|---------|-------------|---------------------|---------------|
| Claude Code | `claude` | `cat file \| claude -p "prompt"` | `-p` / `--print` | `--output-format json/stream-json` |
| GitHub Copilot | `copilot` | `cat error.log \| copilot -p "prompt" -s` | `-p` / `--prompt` | `-s` (silent/clean) |
| Gemini CLI | `gemini` | `echo "prompt" \| gemini` or `cat file \| gemini-cli -q "prompt"` | `-e` (execute) | `-q` (quiet), `--output-format json` |
| OpenCode | `opencode` | TBD - uses TUI | TBD | JSON support |
| Kimi CLI | `kimi-cli` | `kimi-cli cmd --stdin` | Supports chaining | Quiet mode |
| Auggie | `auggie` | Unix-style utility | Scripting support | TBD |
| Aider | `aider` | Scriptable via CLI | `--yes` auto-confirm | TBD |
| Cursor CLI | `cursor-agent` | Pipe to scripts | `-p` print mode | JSON output |

## Detailed Agent Information

### Claude Code
- **Command**: `claude -p "prompt"` for non-interactive
- **Stdin**: `cat file.txt | claude -p "summarize this"`
- **Output formats**: `--output-format json`, `--output-format stream-json`
- **Resume**: `--resume <session-id>`
- **Known issues**: Windows raw mode not supported in interactive mode

### GitHub Copilot CLI
- **Command**: `copilot -p "prompt"`
- **Silent mode**: `-s` flag strips everything but the answer (composable)
- **Examples**:
  - `cat error.log | copilot -p "What went wrong?" -s`
  - `npm test 2>&1 | copilot -p "Why is this failing?" -s`
- **Default model**: Claude Sonnet 4.5

### Gemini CLI
- **Command**: `gemini` or `gemini-cli`
- **Stdin**: `echo "prompt" | gemini` or `cat file | gemini-cli -q "prompt"`
- **Flags**: `-q` (quiet), `-e` (execute/non-interactive)
- **Hyphen attachment**: `git diff | gemini-cli "commit message for: -"`
- **Free tier**: 60 req/min, 1000 req/day

### Kimi CLI
- **Command**: `kimi-cli`
- **Stdin chaining**: `kimi-cli summarize … | kimi-cli keywords --stdin --count 10`
- **Shell integration**: Zsh plugin with Ctrl-X agent mode toggle
- **Context**: 128K token context window

### Auggie (Augment Code)
- **Command**: `auggie`
- **Style**: Unix-style utility for scripting/automation
- **CI integration**: Code review in pipelines
- **Custom commands**: `.augment/commands/` markdown files

### Cursor CLI
- **Command**: `cursor-agent chat "prompt"` or `cursor-agent -p "prompt"`
- **Modes**: Interactive and non-interactive
- **Output**: Can pipe responses to scripts in JSON
- **CI**: Pre-commit hooks, automated reviews

### Aider
- **Command**: `aider`
- **Scripting**: Via command line or Python API
- **Auto-confirm**: `--yes` flag
- **Git integration**: Automatic commits

## Common Patterns

1. **Non-interactive flag**: Most use `-p` or `--print` for single-shot execution
2. **Stdin piping**: `cat input | agent -p "prompt"` is universal pattern
3. **Output formats**: JSON and stream-json for parsing
4. **Silent/quiet mode**: Strip progress indicators for clean output

## Recommended Interface Design

```go
type AIChannel interface {
    // Send sends a prompt with optional context and returns the response
    Send(ctx context.Context, prompt string, context string) (string, error)

    // Configure sets up the channel (command, flags, etc.)
    Configure(config AIChannelConfig) error

    // IsAvailable checks if the agent is installed/accessible
    IsAvailable() bool
}

type AIChannelConfig struct {
    Command      string   // e.g., "claude", "copilot", "gemini"
    Args         []string // e.g., ["-p", "--output-format", "json"]
    QuietFlag    string   // e.g., "-s" for copilot, "-q" for gemini
    StdinSupport bool     // Whether to pipe context via stdin
    Timeout      time.Duration
}
```

## Sources

- [Claude Code Headless Mode](https://code.claude.com/docs/en/headless)
- [GitHub Copilot CLI Docs](https://docs.github.com/en/copilot/how-tos/use-copilot-agents/use-copilot-cli)
- [Gemini CLI](https://github.com/google-gemini/gemini-cli)
- [OpenCode](https://github.com/sst/opencode)
- [Kimi CLI](https://github.com/MoonshotAI/kimi-cli)
- [Auggie CLI](https://github.com/augmentcode/auggie)
- [Aider](https://github.com/Aider-AI/aider)
- [Cursor CLI](https://cursor.com/cli)
