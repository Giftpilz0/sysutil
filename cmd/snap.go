package cmd

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"

	"charm.land/huh/v2"
	"github.com/spf13/cobra"

	"github.com/giftpilz0/sysutil/internal/snapper"
	"github.com/giftpilz0/sysutil/internal/tui"
)

var (
	snapConfig string
	assumeYes  bool
)

func init() {
	snapCmd.PersistentFlags().StringVarP(&snapConfig, "config", "c", "root", "snapper configuration to use")
	_ = snapCmd.RegisterFlagCompletionFunc("config", completeConfigs)
	snapListCmd.Flags().BoolVar(&jsonOutput, "json", false, "print the snapshots as JSON")
	snapUndoCmd.Flags().BoolVarP(&assumeYes, "yes", "y", false, "do not ask for confirmation")

	snapCmd.AddCommand(snapConfigsCmd, snapListCmd, snapCreateCmd, snapDeleteCmd, snapUndoCmd)
	rootCmd.AddCommand(snapCmd)
}

var snapCmd = &cobra.Command{
	Use:   "snap",
	Short: "Manage snapper snapshots",
	Long: `Manage snapper snapshots.

Without a subcommand an interactive snapshot manager is started, where you can
browse snapshots of every configuration, create, delete (also several at once)
and undo changes since a snapshot.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		if !tui.IsInteractive() {
			return cmd.Help()
		}

		return tui.Snap(cmd.Context())
	},
}

var snapConfigsCmd = &cobra.Command{
	Use:   "configs",
	Short: "List snapper configurations",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		configs, err := snapper.Configs(cmd.Context())
		if err != nil {
			return err
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		_, _ = fmt.Fprintln(w, "CONFIG\tSUBVOLUME")
		for _, c := range configs {
			_, _ = fmt.Fprintf(w, "%s\t%s\n", c.Name, c.Subvolume)
		}

		return w.Flush()
	},
}

var snapListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List snapshots",
	Args:    cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		snaps, err := snapper.List(cmd.Context(), snapConfig)
		if err != nil {
			return err
		}

		if jsonOutput {
			return printJSON(snaps)
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		_, _ = fmt.Fprintln(w, "#\tTYPE\tPRE\tDATE\tUSER\tCLEANUP\tDESCRIPTION")
		for _, s := range snaps {
			_, _ = fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\t%s\t%s\n", s.Number, s.Type, s.PreNumber, s.Date, s.User, s.Cleanup, s.Description)
		}

		return w.Flush()
	},
}

var snapCreateCmd = &cobra.Command{
	Use:     "create <description>",
	Short:   "Create a snapshot",
	Example: `  sysutil snap create "before kernel update"`,
	Args:    cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		n, err := snapper.Create(cmd.Context(), snapConfig, strings.Join(args, " "))
		if err != nil {
			return err
		}

		tui.PrintSuccess(os.Stdout, "Created snapshot %d in %s", n, snapConfig)

		return nil
	},
}

var snapDeleteCmd = &cobra.Command{
	Use:     "delete <number>...",
	Aliases: []string{"rm"},
	Short:   "Delete one or more snapshots",
	Args:    cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		nums, err := parseSnapshotNumbers(args)
		if err != nil {
			return err
		}

		if err := snapper.Delete(cmd.Context(), snapConfig, nums...); err != nil {
			return err
		}

		tui.PrintSuccess(os.Stdout, "Deleted %s from %s", strings.Join(args, ", "), snapConfig)

		return nil
	},
}

var snapUndoCmd = &cobra.Command{
	Use:   "undo <number>",
	Short: "Revert all file changes made since a snapshot",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		nums, err := parseSnapshotNumbers(args)
		if err != nil {
			return err
		}

		if !assumeYes {
			if !tui.IsInteractive() {
				return errors.New("refusing to undo changes without confirmation, pass --yes")
			}

			ok := false
			err := huh.NewConfirm().
				Title(fmt.Sprintf("Revert all changes in %q made since snapshot %d?", snapConfig, nums[0])).
				Value(&ok).
				Run()
			if err != nil {
				return err
			}
			if !ok {
				return nil
			}
		}

		if err := snapper.UndoChange(cmd.Context(), snapConfig, nums[0]); err != nil {
			return err
		}

		tui.PrintSuccess(os.Stdout, "Reverted changes since snapshot %d", nums[0])

		return nil
	},
}

func parseSnapshotNumbers(args []string) ([]int, error) {
	nums := make([]int, len(args))
	for i, a := range args {
		n, err := strconv.Atoi(a)
		if err != nil || n <= 0 {
			return nil, fmt.Errorf("invalid snapshot number %q", a)
		}
		nums[i] = n
	}

	return nums, nil
}

func completeConfigs(cmd *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
	configs, err := snapper.Configs(cmd.Context())
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	names := make([]string, len(configs))
	for i, c := range configs {
		names[i] = c.Name
	}

	return names, cobra.ShellCompDirectiveNoFileComp
}
