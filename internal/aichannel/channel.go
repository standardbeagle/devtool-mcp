// Package aichannel provides an interface for communicating with AI coding agents via CLI.
package aichannel

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/creack/pty"
)

// Common errors
var (
	ErrNotConfigured = errors.New("channel not configured")
	ErrNotAvailable  = errors.New("AI agent not available")
	ErrTimeout       = errors.New("request timed out")
)

// AgentType represents a known AI coding agent.
type AgentType string

const (
	AgentClaude   AgentType = "claude"
	AgentCopilot  AgentType = "copilot"
	AgentGemini   AgentType = "gemini"
	AgentOpenCode AgentType = "opencode"
	AgentKimi     AgentType = "kimi-cli"
	AgentAuggie   AgentType = "auggie"
	AgentAider    AgentType = "aider"
	AgentCursor   AgentType = "cursor-agent"
	AgentCustom   AgentType = "custom"
)

// Config holds the configuration for an AI channel.
type Config struct {
	// Agent is the type of AI agent to use
	Agent AgentType `json:"agent"`

	// Command is the executable command (defaults based on agent type)
	Command string `json:"command,omitempty"`

	// Args are additional arguments to pass to the command
	Args []string `json:"args,omitempty"`

	// NonInteractiveFlag is the flag to use for non-interactive mode (e.g., "-p")
	NonInteractiveFlag string `json:"non_interactive_flag,omitempty"`

	// QuietFlag suppresses progress output (e.g., "-s" for copilot, "-q" for gemini)
	QuietFlag string `json:"quiet_flag,omitempty"`

	// OutputFormat specifies output format flag (e.g., "--output-format json")
	OutputFormat string `json:"output_format,omitempty"`

	// UseStdin determines if context should be piped via stdin
	UseStdin bool `json:"use_stdin"`

	// UsePTY runs the command in a pseudo-terminal (required for some CLI tools)
	UsePTY bool `json:"use_pty"`

	// Timeout for the request (default 2 minutes)
	Timeout time.Duration `json:"timeout,omitempty"`

	// Environment variables to set
	Env map[string]string `json:"env,omitempty"`
}

// Channel represents a communication channel to an AI coding agent.
type Channel struct {
	config     Config
	configured bool
}

// New creates a new AI channel.
func New() *Channel {
	return &Channel{}
}

// NewWithConfig creates a new AI channel with the given configuration.
func NewWithConfig(config Config) *Channel {
	ch := &Channel{}
	ch.Configure(config)
	return ch
}

// Configure sets up the channel with the given configuration.
func (c *Channel) Configure(config Config) {
	// Apply defaults based on agent type
	config = applyDefaults(config)
	c.config = config
	c.configured = true
}

// applyDefaults fills in default values based on agent type.
func applyDefaults(config Config) Config {
	if config.Timeout == 0 {
		config.Timeout = 2 * time.Minute
	}

	// Set command and flags based on agent type if not specified
	switch config.Agent {
	case AgentClaude:
		if config.Command == "" {
			config.Command = "claude"
		}
		if config.NonInteractiveFlag == "" {
			config.NonInteractiveFlag = "-p"
		}
		if config.OutputFormat == "" {
			config.OutputFormat = "text" // or "json", "stream-json"
		}
		config.UseStdin = true
		config.UsePTY = true // Claude Code requires a PTY for -p mode

	case AgentCopilot:
		if config.Command == "" {
			config.Command = "copilot"
		}
		if config.NonInteractiveFlag == "" {
			config.NonInteractiveFlag = "-p"
		}
		if config.QuietFlag == "" {
			config.QuietFlag = "-s"
		}
		config.UseStdin = true

	case AgentGemini:
		if config.Command == "" {
			config.Command = "gemini"
		}
		if config.NonInteractiveFlag == "" {
			config.NonInteractiveFlag = "-e"
		}
		if config.QuietFlag == "" {
			config.QuietFlag = "-q"
		}
		config.UseStdin = true

	case AgentOpenCode:
		if config.Command == "" {
			config.Command = "opencode"
		}

	case AgentKimi:
		if config.Command == "" {
			config.Command = "kimi-cli"
		}
		config.UseStdin = true

	case AgentAuggie:
		if config.Command == "" {
			config.Command = "auggie"
		}

	case AgentAider:
		if config.Command == "" {
			config.Command = "aider"
		}
		// Aider uses --message for non-interactive
		if config.NonInteractiveFlag == "" {
			config.NonInteractiveFlag = "--message"
		}

	case AgentCursor:
		if config.Command == "" {
			config.Command = "cursor-agent"
		}
		if config.NonInteractiveFlag == "" {
			config.NonInteractiveFlag = "-p"
		}
	}

	return config
}

// IsAvailable checks if the configured AI agent is available.
func (c *Channel) IsAvailable() bool {
	if !c.configured {
		return false
	}

	_, err := exec.LookPath(c.config.Command)
	return err == nil
}

// Send sends a prompt to the AI agent and returns the response.
// If context is non-empty and UseStdin is true, context is piped via stdin.
func (c *Channel) Send(ctx context.Context, prompt string, inputContext string) (string, error) {
	if !c.configured {
		return "", ErrNotConfigured
	}

	if !c.IsAvailable() {
		return "", fmt.Errorf("%w: %s not found in PATH", ErrNotAvailable, c.config.Command)
	}

	// Use PTY mode if configured (required for some CLI tools like Claude Code)
	if c.config.UsePTY {
		return c.sendWithPTY(ctx, prompt, inputContext)
	}

	return c.sendWithPipe(ctx, prompt, inputContext)
}

// sendWithPipe runs the command with standard pipes (no PTY).
func (c *Channel) sendWithPipe(ctx context.Context, prompt string, inputContext string) (string, error) {
	// Build command arguments
	args := c.buildArgs(prompt)

	// Create context with timeout
	execCtx, cancel := context.WithTimeout(ctx, c.config.Timeout)
	defer cancel()

	// Create command
	cmd := exec.CommandContext(execCtx, c.config.Command, args...)

	// Set environment - inherit current environment and add custom vars
	if len(c.config.Env) > 0 {
		cmd.Env = os.Environ()
		for k, v := range c.config.Env {
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
		}
	}

	// Setup stdin if context is provided
	var stdin bytes.Buffer
	if inputContext != "" && c.config.UseStdin {
		stdin.WriteString(inputContext)
		cmd.Stdin = &stdin
	}

	// Capture output
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Run command
	err := cmd.Run()

	// Check for context timeout
	if execCtx.Err() == context.DeadlineExceeded {
		return "", ErrTimeout
	}

	if err != nil {
		// Include stderr in error message if available
		errMsg := stderr.String()
		if errMsg != "" {
			return "", fmt.Errorf("command failed: %w: %s", err, strings.TrimSpace(errMsg))
		}
		return "", fmt.Errorf("command failed: %w", err)
	}

	return strings.TrimSpace(stdout.String()), nil
}

// sendWithPTY runs the command in a pseudo-terminal.
// This is required for CLI tools that need a TTY (like Claude Code).
func (c *Channel) sendWithPTY(ctx context.Context, prompt string, inputContext string) (string, error) {
	// Build command arguments
	args := c.buildArgs(prompt)

	// Create context with timeout
	execCtx, cancel := context.WithTimeout(ctx, c.config.Timeout)
	defer cancel()

	// Create command
	cmd := exec.CommandContext(execCtx, c.config.Command, args...)

	// Set environment - inherit current environment and add custom vars
	cmd.Env = os.Environ()
	for k, v := range c.config.Env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	// Start the command with a PTY
	ptmx, err := pty.Start(cmd)
	if err != nil {
		return "", fmt.Errorf("failed to start PTY: %w", err)
	}
	defer ptmx.Close()

	// Write input context to PTY if configured
	if inputContext != "" && c.config.UseStdin {
		if _, err := ptmx.WriteString(inputContext); err != nil {
			return "", fmt.Errorf("failed to write to PTY: %w", err)
		}
	}

	// Read all output from PTY
	var output bytes.Buffer
	done := make(chan error, 1)

	go func() {
		_, err := io.Copy(&output, ptmx)
		done <- err
	}()

	// Wait for command to complete or context to be cancelled
	waitErr := cmd.Wait()

	// Wait for output reading to complete (with timeout for cleanup)
	select {
	case <-done:
		// Output reading completed
	case <-time.After(100 * time.Millisecond):
		// Give up waiting for output
	}

	// Check for context timeout
	if execCtx.Err() == context.DeadlineExceeded {
		return "", ErrTimeout
	}

	if waitErr != nil {
		// Check if it's just a signal error from context cancellation
		if execCtx.Err() != nil {
			return "", ErrTimeout
		}
		return "", fmt.Errorf("command failed: %w", waitErr)
	}

	// Clean ANSI escape sequences from output
	result := stripANSI(output.String())
	return strings.TrimSpace(result), nil
}

// stripANSI removes ANSI escape sequences from a string.
func stripANSI(s string) string {
	var result strings.Builder
	inEscape := false
	for i := 0; i < len(s); i++ {
		if s[i] == '\x1b' {
			inEscape = true
			continue
		}
		if inEscape {
			// Check for CSI sequences (ESC [)
			if s[i] == '[' {
				// Skip until we hit a letter (the command byte)
				for i++; i < len(s); i++ {
					if (s[i] >= 'A' && s[i] <= 'Z') || (s[i] >= 'a' && s[i] <= 'z') {
						break
					}
				}
				inEscape = false
				continue
			}
			// Check for OSC sequences (ESC ])
			if s[i] == ']' {
				// Skip until BEL or ST
				for i++; i < len(s); i++ {
					if s[i] == '\x07' { // BEL
						break
					}
					if s[i] == '\x1b' && i+1 < len(s) && s[i+1] == '\\' { // ST
						i++
						break
					}
				}
				inEscape = false
				continue
			}
			// Single character escape
			inEscape = false
			continue
		}
		// Skip carriage returns (terminal line endings)
		if s[i] == '\r' {
			continue
		}
		result.WriteByte(s[i])
	}
	return result.String()
}

// buildArgs constructs the command arguments based on configuration.
func (c *Channel) buildArgs(prompt string) []string {
	var args []string

	// Add configured args first
	args = append(args, c.config.Args...)

	// Add non-interactive flag with prompt (only if prompt is non-empty)
	if prompt != "" {
		if c.config.NonInteractiveFlag != "" {
			args = append(args, c.config.NonInteractiveFlag, prompt)
		} else {
			args = append(args, prompt)
		}
	}

	// Add quiet flag
	if c.config.QuietFlag != "" {
		args = append(args, c.config.QuietFlag)
	}

	// Add output format if specified and not "text"
	if c.config.OutputFormat != "" && c.config.OutputFormat != "text" {
		args = append(args, "--output-format", c.config.OutputFormat)
	}

	return args
}

// Config returns a copy of the current configuration.
func (c *Channel) Config() Config {
	return c.config
}

// GetKnownAgents returns a list of known AI agent types.
func GetKnownAgents() []AgentType {
	return []AgentType{
		AgentClaude,
		AgentCopilot,
		AgentGemini,
		AgentOpenCode,
		AgentKimi,
		AgentAuggie,
		AgentAider,
		AgentCursor,
	}
}

// DetectAvailableAgents returns a list of AI agents that are available in PATH.
func DetectAvailableAgents() []AgentType {
	var available []AgentType
	for _, agent := range GetKnownAgents() {
		ch := NewWithConfig(Config{Agent: agent})
		if ch.IsAvailable() {
			available = append(available, agent)
		}
	}
	return available
}
