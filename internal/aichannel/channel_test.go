package aichannel

import (
	"context"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	ch := New()
	if ch == nil {
		t.Fatal("New() returned nil")
	}
	if ch.configured {
		t.Error("New channel should not be configured")
	}
}

func TestNewWithConfig(t *testing.T) {
	ch := NewWithConfig(Config{
		Agent:   AgentClaude,
		Timeout: 30 * time.Second,
	})
	if ch == nil {
		t.Fatal("NewWithConfig() returned nil")
	}
	if !ch.configured {
		t.Error("Channel should be configured")
	}
}

func TestConfigure_AppliesDefaults(t *testing.T) {
	tests := []struct {
		name                string
		agent               AgentType
		expectedCommand     string
		expectedNonIntFlag  string
		expectedQuietFlag   string
		expectedUseStdin    bool
	}{
		{
			name:               "Claude",
			agent:              AgentClaude,
			expectedCommand:    "claude",
			expectedNonIntFlag: "-p",
			expectedQuietFlag:  "",
			expectedUseStdin:   true,
		},
		{
			name:               "Copilot",
			agent:              AgentCopilot,
			expectedCommand:    "copilot",
			expectedNonIntFlag: "-p",
			expectedQuietFlag:  "-s",
			expectedUseStdin:   true,
		},
		{
			name:               "Gemini",
			agent:              AgentGemini,
			expectedCommand:    "gemini",
			expectedNonIntFlag: "-e",
			expectedQuietFlag:  "-q",
			expectedUseStdin:   true,
		},
		{
			name:               "Kimi",
			agent:              AgentKimi,
			expectedCommand:    "kimi-cli",
			expectedNonIntFlag: "",
			expectedQuietFlag:  "",
			expectedUseStdin:   true,
		},
		{
			name:               "Cursor",
			agent:              AgentCursor,
			expectedCommand:    "cursor-agent",
			expectedNonIntFlag: "-p",
			expectedQuietFlag:  "",
			expectedUseStdin:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ch := NewWithConfig(Config{Agent: tt.agent})
			cfg := ch.Config()

			if cfg.Command != tt.expectedCommand {
				t.Errorf("Command = %q, want %q", cfg.Command, tt.expectedCommand)
			}
			if cfg.NonInteractiveFlag != tt.expectedNonIntFlag {
				t.Errorf("NonInteractiveFlag = %q, want %q", cfg.NonInteractiveFlag, tt.expectedNonIntFlag)
			}
			if cfg.QuietFlag != tt.expectedQuietFlag {
				t.Errorf("QuietFlag = %q, want %q", cfg.QuietFlag, tt.expectedQuietFlag)
			}
			if cfg.UseStdin != tt.expectedUseStdin {
				t.Errorf("UseStdin = %v, want %v", cfg.UseStdin, tt.expectedUseStdin)
			}
		})
	}
}

func TestConfigure_CustomCommand(t *testing.T) {
	ch := NewWithConfig(Config{
		Agent:   AgentClaude,
		Command: "/custom/path/claude",
	})
	cfg := ch.Config()

	if cfg.Command != "/custom/path/claude" {
		t.Errorf("Custom command not preserved: got %q", cfg.Command)
	}
}

func TestConfigure_DefaultTimeout(t *testing.T) {
	ch := NewWithConfig(Config{Agent: AgentClaude})
	cfg := ch.Config()

	if cfg.Timeout != 2*time.Minute {
		t.Errorf("Timeout = %v, want 2 minutes", cfg.Timeout)
	}
}

func TestConfigure_CustomTimeout(t *testing.T) {
	ch := NewWithConfig(Config{
		Agent:   AgentClaude,
		Timeout: 5 * time.Minute,
	})
	cfg := ch.Config()

	if cfg.Timeout != 5*time.Minute {
		t.Errorf("Custom timeout not preserved: got %v", cfg.Timeout)
	}
}

func TestIsAvailable_NotConfigured(t *testing.T) {
	ch := New()
	if ch.IsAvailable() {
		t.Error("Unconfigured channel should not be available")
	}
}

func TestIsAvailable_MissingCommand(t *testing.T) {
	ch := NewWithConfig(Config{
		Agent:   AgentCustom,
		Command: "nonexistent-ai-tool-12345",
	})
	if ch.IsAvailable() {
		t.Error("Channel with missing command should not be available")
	}
}

func TestIsAvailable_Echo(t *testing.T) {
	// Echo should be available on all systems
	ch := NewWithConfig(Config{
		Agent:   AgentCustom,
		Command: "echo",
	})
	if !ch.IsAvailable() {
		t.Error("Channel with 'echo' command should be available")
	}
}

func TestSend_NotConfigured(t *testing.T) {
	ch := New()
	_, err := ch.Send(context.Background(), "test", "")
	if err != ErrNotConfigured {
		t.Errorf("Expected ErrNotConfigured, got %v", err)
	}
}

func TestSend_NotAvailable(t *testing.T) {
	ch := NewWithConfig(Config{
		Agent:   AgentCustom,
		Command: "nonexistent-ai-tool-12345",
	})
	_, err := ch.Send(context.Background(), "test", "")
	if err == nil {
		t.Error("Expected error for unavailable command")
	}
}

func TestSend_WithEcho(t *testing.T) {
	// Use echo as a mock AI agent
	ch := NewWithConfig(Config{
		Agent:              AgentCustom,
		Command:            "echo",
		NonInteractiveFlag: "", // echo doesn't need flags
	})

	result, err := ch.Send(context.Background(), "hello world", "")
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}

	if result != "hello world" {
		t.Errorf("Send() = %q, want %q", result, "hello world")
	}
}

func TestSend_WithStdin(t *testing.T) {
	// Use cat as a mock that reads stdin
	ch := NewWithConfig(Config{
		Agent:              AgentCustom,
		Command:            "cat",
		NonInteractiveFlag: "",
		UseStdin:           true,
	})

	result, err := ch.Send(context.Background(), "", "stdin content")
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}

	if result != "stdin content" {
		t.Errorf("Send() = %q, want %q", result, "stdin content")
	}
}

func TestSend_Timeout(t *testing.T) {
	// Use sleep to test timeout
	ch := NewWithConfig(Config{
		Agent:              AgentCustom,
		Command:            "sleep",
		NonInteractiveFlag: "",
		Timeout:            100 * time.Millisecond,
	})

	_, err := ch.Send(context.Background(), "10", "")
	if err != ErrTimeout {
		t.Errorf("Expected ErrTimeout, got %v", err)
	}
}

func TestBuildArgs_Claude(t *testing.T) {
	ch := NewWithConfig(Config{Agent: AgentClaude})
	args := ch.buildArgs("test prompt")

	// Should have: -p "test prompt"
	found := false
	for i, arg := range args {
		if arg == "-p" && i+1 < len(args) && args[i+1] == "test prompt" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected -p flag with prompt, got %v", args)
	}
}

func TestBuildArgs_Copilot(t *testing.T) {
	ch := NewWithConfig(Config{Agent: AgentCopilot})
	args := ch.buildArgs("test prompt")

	// Should have: -p "test prompt" -s
	hasP := false
	hasS := false
	for i, arg := range args {
		if arg == "-p" && i+1 < len(args) && args[i+1] == "test prompt" {
			hasP = true
		}
		if arg == "-s" {
			hasS = true
		}
	}
	if !hasP {
		t.Errorf("Expected -p flag with prompt, got %v", args)
	}
	if !hasS {
		t.Errorf("Expected -s (quiet) flag, got %v", args)
	}
}

func TestBuildArgs_WithExtraArgs(t *testing.T) {
	ch := NewWithConfig(Config{
		Agent: AgentClaude,
		Args:  []string{"--model", "opus"},
	})
	args := ch.buildArgs("test")

	// Extra args should come first
	if len(args) < 2 || args[0] != "--model" || args[1] != "opus" {
		t.Errorf("Extra args not at beginning: %v", args)
	}
}

func TestBuildArgs_OutputFormat(t *testing.T) {
	ch := NewWithConfig(Config{
		Agent:        AgentClaude,
		OutputFormat: "json",
	})
	args := ch.buildArgs("test")

	found := false
	for i, arg := range args {
		if arg == "--output-format" && i+1 < len(args) && args[i+1] == "json" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected --output-format json, got %v", args)
	}
}

func TestGetKnownAgents(t *testing.T) {
	agents := GetKnownAgents()
	if len(agents) == 0 {
		t.Error("GetKnownAgents() returned empty list")
	}

	// Check that known agents are in the list
	expected := map[AgentType]bool{
		AgentClaude:  true,
		AgentCopilot: true,
		AgentGemini:  true,
	}

	for agent := range expected {
		found := false
		for _, a := range agents {
			if a == agent {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected agent %s not in list", agent)
		}
	}
}

func TestDetectAvailableAgents(t *testing.T) {
	// This just tests that the function runs without error
	// The result depends on what's installed
	agents := DetectAvailableAgents()
	t.Logf("Detected available agents: %v", agents)
}
