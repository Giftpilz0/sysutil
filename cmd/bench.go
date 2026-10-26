package cmd

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/giftpilz0/sysutil/internal/bench"
	"github.com/giftpilz0/sysutil/internal/tui"
)

var benchCfg = bench.DefaultConfig()

func init() {
	benchCmd.Flags().IntVarP(&benchCfg.Runs, "runs", "r", benchCfg.Runs, "number of matrix-vector multiplications")
	benchCmd.Flags().IntVarP(&benchCfg.Size, "size", "s", benchCfg.Size, "dimension of the square matrix")
	benchCmd.Flags().IntVarP(&benchCfg.Workers, "workers", "w", benchCfg.Workers, "number of parallel workers")
	benchCmd.Flags().BoolVar(&jsonOutput, "json", false, "print the result as JSON")
	rootCmd.AddCommand(benchCmd)
}

var benchCmd = &cobra.Command{
	Use:   "bench",
	Short: "Run a matrix-vector multiplication benchmark",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		if interactive {
			return tui.Bench(cmd.Context())
		}

		res, err := bench.Run(cmd.Context(), benchCfg, nil)
		if err != nil {
			return err
		}

		if jsonOutput {
			return printJSON(res)
		}

		tui.PrintBenchResult(os.Stdout, res)

		return nil
	},
}
