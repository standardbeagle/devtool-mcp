//go:build windows

package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// runCmd is the "run" command stub for Windows.
// PTY-based overlay requires Unix-specific features and is not supported on Windows.
var runCmd = &cobra.Command{
	Use:   "run <command> [args...]",
	Short: "Run an AI coding tool with overlay features (Unix only)",
	Long: `Run any AI coding tool (Claude, Gemini, Copilot, etc.) with overlay features.

NOTE: This command is not supported on Windows. The PTY-based overlay requires
Unix-specific features (pseudo-terminals, SIGWINCH signals, etc.).

On Windows, run your AI coding tool directly without the overlay wrapper.`,
	DisableFlagParsing: true,
	Args:               cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Fprintln(os.Stderr, "Error: 'run' command is not supported on Windows")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "The PTY-based overlay requires Unix-specific features.")
		fmt.Fprintln(os.Stderr, "Run your AI coding tool directly instead:")
		fmt.Fprintf(os.Stderr, "  %s", args[0])
		if len(args) > 1 {
			for _, a := range args[1:] {
				fmt.Fprintf(os.Stderr, " %s", a)
			}
		}
		fmt.Fprintln(os.Stderr, "")
		os.Exit(1)
	},
}
