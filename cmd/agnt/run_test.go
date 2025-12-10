package main

import (
	"testing"
)

func TestShellQuote(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple word",
			input:    "hello",
			expected: "hello",
		},
		{
			name:     "word with space",
			input:    "hello world",
			expected: "'hello world'",
		},
		{
			name:     "word with single quote",
			input:    "it's",
			expected: "'it'\\''s'",
		},
		{
			name:     "word with double quote",
			input:    `say "hello"`,
			expected: `'say "hello"'`,
		},
		{
			name:     "word with dollar sign",
			input:    "$HOME",
			expected: "'$HOME'",
		},
		{
			name:     "word with backtick",
			input:    "`whoami`",
			expected: "'`whoami`'",
		},
		{
			name:     "word with semicolon",
			input:    "cmd;rm -rf",
			expected: "'cmd;rm -rf'",
		},
		{
			name:     "word with pipe",
			input:    "cmd|cat",
			expected: "'cmd|cat'",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "flag with equals",
			input:    "--flag=value",
			expected: "--flag=value",
		},
		{
			name:     "flag with space in value",
			input:    "--message=hello world",
			expected: "'--message=hello world'",
		},
		{
			name:     "complex path",
			input:    "/path/to/file",
			expected: "/path/to/file",
		},
		{
			name:     "path with space",
			input:    "/path/to/my file",
			expected: "'/path/to/my file'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shellQuote(tt.input)
			if result != tt.expected {
				t.Errorf("shellQuote(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
