package cmd

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/giftpilz0/sysutil/internal/ldif"
	"github.com/giftpilz0/sysutil/internal/tui"
)

var (
	ldifOutput    string
	ldifSeparator string
	ldifModify    ldif.Modify
)

func init() {
	for _, c := range []*cobra.Command{toLDIFCmd, toCSVCmd} {
		c.Flags().StringVarP(&ldifOutput, "output", "o", "", `output file, "-" for stdout (default: input name with new extension)`)
		c.Flags().StringVarP(&ldifSeparator, "separator", "s", ";", `CSV field separator, "tab" for tabs`)
		ldifCmd.AddCommand(c)
	}

	toLDIFCmd.Flags().StringVarP(&ldifModify.Attribute, "attribute", "a", "", "CSV column / LDAP attribute to modify (required)")
	toLDIFCmd.Flags().StringVarP(&ldifModify.ChangeType, "type", "t", "replace", "modify operation: replace, add or delete")
	_ = toLDIFCmd.MarkFlagRequired("attribute")
	_ = toLDIFCmd.RegisterFlagCompletionFunc("type", cobra.FixedCompletions(ldif.ChangeTypes, cobra.ShellCompDirectiveNoFileComp))

	rootCmd.AddCommand(ldifCmd)
}

var ldifCmd = &cobra.Command{
	Use:   "ldif",
	Short: "Convert between CSV and LDIF",
	Long: `Convert between CSV and LDIF.

Run without a subcommand in a terminal for an interactive wizard.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		if !interactive && !tui.IsInteractive() {
			return cmd.Help()
		}

		return tui.LDIF(cmd.Context())
	},
}

var toLDIFCmd = &cobra.Command{
	Use:   "to-ldif <input.csv>",
	Short: "Generate LDIF modify records from a CSV file",
	Long: `Generate an LDIF modify record for every CSV row with a value in the given
attribute column. The CSV needs a "dn" column. Multiple values can be written as
"[value1][value2]".`,
	Example: "  sysutil ldif to-ldif users.csv -a mail\n  sysutil ldif to-ldif users.csv -a memberOf -t add -o - | ldapmodify ...",
	Args:    cobra.ExactArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		return convertLDIF(ldif.ToLDIF, args[0])
	},
}

var toCSVCmd = &cobra.Command{
	Use:   "to-csv <input.ldif>",
	Short: "Export LDIF entries to a CSV file",
	Long: `Export LDIF entries to CSV. The first column is "dn", followed by all
attributes. Multi-valued attributes are written as "[value1][value2]".`,
	Example: "  sysutil ldif to-csv export.ldif\n  ldapsearch -LLL ... | sysutil ldif to-csv - -o users.csv",
	Args:    cobra.ExactArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		return convertLDIF(ldif.ToCSV, args[0])
	},
}

func convertLDIF(dir ldif.Direction, input string) error {
	sep, err := ldif.ParseSeparator(ldifSeparator)
	if err != nil {
		return err
	}

	output := ldifOutput
	if output == "" {
		output = ldif.DefaultOutput(input, dir)
	}

	n, err := ldif.ConvertFile(dir, input, output, sep, ldifModify)
	if err != nil {
		return err
	}

	if output != "-" {
		tui.PrintSuccess(os.Stderr, "Wrote %d records to %s", n, output)
	}

	return nil
}
