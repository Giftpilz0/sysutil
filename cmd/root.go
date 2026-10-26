package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/giftpilz0/sysutil/internal/tui"
)

var interactive bool

var rootCmd = &cobra.Command{
	Use:   "sysutil",
	Short: "A collection of small system utilities",
	Long: `sysutil is a collection of small system utilities.

Run it without arguments in a terminal to pick a tool from an interactive menu,
or pass -i to any command to be prompted for its options.`,
	Args:          cobra.NoArgs,
	SilenceUsage:  true,
	SilenceErrors: true,
	PersistentPreRunE: func(*cobra.Command, []string) error {
		if interactive && !tui.IsInteractive() {
			return errors.New("--interactive requires a terminal")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, _ []string) error {
		if !tui.IsInteractive() {
			return cmd.Help()
		}

		return tui.Launcher(cmd.Context(), tools())
	},
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&interactive, "interactive", "i", false, "prompt for options interactively")
	rootCmd.CompletionOptions.HiddenDefaultCmd = true
}

func tools() []tui.Tool {
	return []tui.Tool{
		{Name: "ip", Description: "Show your public IP address", Run: tui.IP},
		{Name: "mac", Description: "Look up the vendor of a MAC address", Run: tui.MAC},
		{Name: "snap", Description: "Manage snapper snapshots", Run: tui.Snap},
		{Name: "ldif", Description: "Convert between CSV and LDIF", Run: tui.LDIF},
		{Name: "bench", Description: "Run a matrix-vector multiplication benchmark", Run: tui.Bench},
	}
}

func Execute(version string) int {
	rootCmd.Version = version

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := rootCmd.ExecuteContext(ctx); err != nil {
		if errors.Is(err, context.Canceled) {
			return 130
		}
		tui.PrintError(os.Stderr, err)
		return 1
	}

	return 0
}

func printJSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")

	return enc.Encode(v)
}
