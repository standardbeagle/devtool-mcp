package overlay

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"devtool-mcp/internal/aichannel"
	"devtool-mcp/internal/daemon"
	"devtool-mcp/internal/protocol"
)

// Summarizer aggregates system status and uses an AI channel to generate summaries.
type Summarizer struct {
	socketPath  string
	channel     *aichannel.Channel
	debugOutput io.Writer
}

// SummarizerConfig configures the Summarizer.
type SummarizerConfig struct {
	SocketPath string
	Agent      aichannel.AgentType
	// Optional custom command (overrides agent default)
	Command string
	// Optional additional args
	Args []string
	// Timeout for AI response (default 2 minutes)
	Timeout time.Duration
	// DebugOutput is where debug messages are written (defaults to os.Stderr)
	DebugOutput io.Writer
}

// NewSummarizer creates a new Summarizer.
func NewSummarizer(config SummarizerConfig) *Summarizer {
	channelConfig := aichannel.Config{
		Agent:   config.Agent,
		Command: config.Command,
		Args:    config.Args,
		Timeout: config.Timeout,
	}

	return &Summarizer{
		socketPath:  config.SocketPath,
		channel:     aichannel.NewWithConfig(channelConfig),
		debugOutput: config.DebugOutput,
	}
}

// debugWriter returns the debug output writer, defaulting to os.Stderr.
func (s *Summarizer) debugWriter() io.Writer {
	if s.debugOutput != nil {
		return s.debugOutput
	}
	return os.Stderr
}

// IsAvailable returns true if the AI channel is available.
func (s *Summarizer) IsAvailable() bool {
	return s.channel.IsAvailable()
}

// AgentType returns the configured agent type.
func (s *Summarizer) AgentType() aichannel.AgentType {
	return s.channel.Config().Agent
}

// SummaryResult contains the result of a summarization.
type SummaryResult struct {
	Summary     string
	ProcessData []ProcessSummary
	ProxyData   []ProxySummary
	ErrorCount  int
	Duration    time.Duration
}

// ProcessSummary contains summarized info about a process.
type ProcessSummary struct {
	ID        string
	Command   string
	State     string
	HasErrors bool
	Output    string // Last N lines
}

// ProxySummary contains summarized info about a proxy.
type ProxySummary struct {
	ID         string
	TargetURL  string
	ListenAddr string
	ErrorCount int
	PageCount  int
}

// Summarize aggregates all system data and generates a summary.
func (s *Summarizer) Summarize(ctx context.Context) (*SummaryResult, error) {
	start := time.Now()

	// Connect to daemon
	client, err := s.createClient()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to daemon: %w", err)
	}
	defer client.Close()

	// Gather all data
	processes, err := s.gatherProcesses(client)
	if err != nil {
		return nil, fmt.Errorf("failed to gather process data: %w", err)
	}

	proxies, err := s.gatherProxies(client)
	if err != nil {
		return nil, fmt.Errorf("failed to gather proxy data: %w", err)
	}

	// Build context for AI
	contextData := s.buildContext(processes, proxies)

	// Generate summary via AI
	prompt := s.buildPrompt()

	// Log what we're about to do for debugging
	fmt.Fprintf(s.debugWriter(), "[agnt] Calling %s with %d bytes of context...\r\n",
		s.channel.Config().Command, len(contextData))

	summary, err := s.channel.Send(ctx, prompt, contextData)
	if err != nil {
		return nil, fmt.Errorf("AI summarization failed: %w", err)
	}

	// Count errors
	errorCount := 0
	for _, p := range processes {
		if p.HasErrors {
			errorCount++
		}
	}
	for _, p := range proxies {
		errorCount += p.ErrorCount
	}

	return &SummaryResult{
		Summary:     summary,
		ProcessData: processes,
		ProxyData:   proxies,
		ErrorCount:  errorCount,
		Duration:    time.Since(start),
	}, nil
}

func (s *Summarizer) createClient() (*daemon.Client, error) {
	opts := []daemon.ClientOption{}
	if s.socketPath != "" {
		opts = append(opts, daemon.WithSocketPath(s.socketPath))
	}

	client := daemon.NewClient(opts...)
	if err := client.Connect(); err != nil {
		return nil, err
	}
	return client, nil
}

func (s *Summarizer) gatherProcesses(client *daemon.Client) ([]ProcessSummary, error) {
	// Get process list
	result, err := client.ProcList(protocol.DirectoryFilter{Global: true})
	if err != nil {
		return nil, err
	}

	processesRaw, ok := result["processes"].([]interface{})
	if !ok {
		return nil, nil
	}

	processes := make([]ProcessSummary, 0, len(processesRaw))
	for _, p := range processesRaw {
		pm, ok := p.(map[string]interface{})
		if !ok {
			continue
		}

		summary := ProcessSummary{}
		if id, ok := pm["id"].(string); ok {
			summary.ID = id
		}
		if cmd, ok := pm["command"].(string); ok {
			summary.Command = cmd
		}
		if state, ok := pm["state"].(string); ok {
			summary.State = state
		}

		// Fetch last 50 lines of output
		if summary.ID != "" {
			filter := protocol.OutputFilter{
				Stream: "combined",
				Tail:   50,
			}
			output, err := client.ProcOutput(summary.ID, filter)
			if err == nil {
				summary.Output = output
				// Check for error patterns in output
				summary.HasErrors = containsErrorPatterns(output)
			}
		}

		processes = append(processes, summary)
	}

	return processes, nil
}

func (s *Summarizer) gatherProxies(client *daemon.Client) ([]ProxySummary, error) {
	// Get proxy list
	result, err := client.ProxyList(protocol.DirectoryFilter{Global: true})
	if err != nil {
		return nil, err
	}

	proxiesRaw, ok := result["proxies"].([]interface{})
	if !ok {
		return nil, nil
	}

	proxies := make([]ProxySummary, 0, len(proxiesRaw))
	for _, p := range proxiesRaw {
		pm, ok := p.(map[string]interface{})
		if !ok {
			continue
		}

		summary := ProxySummary{}
		if id, ok := pm["id"].(string); ok {
			summary.ID = id
		}
		if target, ok := pm["target_url"].(string); ok {
			summary.TargetURL = target
		}
		if listen, ok := pm["listen_addr"].(string); ok {
			summary.ListenAddr = listen
		}

		// Get stats
		if stats, ok := pm["stats"].(map[string]interface{}); ok {
			if errCount, ok := stats["error_count"].(float64); ok {
				summary.ErrorCount = int(errCount)
			}
		}

		// Get page count
		pagesResult, err := client.CurrentPageList(summary.ID)
		if err == nil {
			if sessions, ok := pagesResult["sessions"].([]interface{}); ok {
				summary.PageCount = len(sessions)
			}
		}

		proxies = append(proxies, summary)
	}

	return proxies, nil
}

func (s *Summarizer) buildContext(processes []ProcessSummary, proxies []ProxySummary) string {
	var sb strings.Builder

	sb.WriteString("=== SYSTEM STATUS ===\n\n")

	// Processes section
	sb.WriteString("== PROCESSES ==\n")
	if len(processes) == 0 {
		sb.WriteString("No running processes.\n")
	} else {
		for _, p := range processes {
			sb.WriteString(fmt.Sprintf("\n--- Process: %s ---\n", p.ID))
			sb.WriteString(fmt.Sprintf("Command: %s\n", p.Command))
			sb.WriteString(fmt.Sprintf("State: %s\n", p.State))
			if p.Output != "" {
				sb.WriteString("Output (last 50 lines):\n")
				sb.WriteString(p.Output)
				sb.WriteString("\n")
			}
		}
	}

	// Proxies section
	sb.WriteString("\n== PROXIES ==\n")
	if len(proxies) == 0 {
		sb.WriteString("No running proxies.\n")
	} else {
		for _, p := range proxies {
			sb.WriteString(fmt.Sprintf("\n--- Proxy: %s ---\n", p.ID))
			sb.WriteString(fmt.Sprintf("Target: %s\n", p.TargetURL))
			sb.WriteString(fmt.Sprintf("Listen: %s\n", p.ListenAddr))
			sb.WriteString(fmt.Sprintf("Error Count: %d\n", p.ErrorCount))
			sb.WriteString(fmt.Sprintf("Active Pages: %d\n", p.PageCount))
		}
	}

	return sb.String()
}

func (s *Summarizer) buildPrompt() string {
	return `Analyze the system status below and provide a concise summary. Focus on:

1. If everything is healthy, say so briefly (1-2 sentences)
2. If there are errors:
   - Identify the root cause from stack traces or error messages
   - Filter out noise (routine logs, SQL queries, etc.)
   - Highlight the key information needed to troubleshoot
   - Suggest potential fixes if obvious

Keep the summary short and actionable. Use bullet points for multiple issues.`
}

// containsErrorPatterns checks if output contains common error patterns.
func containsErrorPatterns(output string) bool {
	lower := strings.ToLower(output)
	patterns := []string{
		"error",
		"exception",
		"failed",
		"panic",
		"fatal",
		"segfault",
		"timeout",
		"refused",
		"denied",
		"not found",
		"undefined",
		"null pointer",
		"stack trace",
		"traceback",
	}

	for _, p := range patterns {
		if strings.Contains(lower, p) {
			return true
		}
	}
	return false
}
