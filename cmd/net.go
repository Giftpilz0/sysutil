package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/giftpilz0/sysutil/internal/netinfo"
	"github.com/giftpilz0/sysutil/internal/tui"
)

var (
	jsonOutput bool
	listIfaces bool
)

func init() {
	ipCmd.Flags().BoolVar(&jsonOutput, "json", false, "print the result as JSON")
	rootCmd.AddCommand(ipCmd)

	macCmd.Flags().BoolVar(&jsonOutput, "json", false, "print the result as JSON")
	macCmd.Flags().BoolVarP(&listIfaces, "list", "l", false, "list local network interfaces and their MAC addresses")
	rootCmd.AddCommand(macCmd)
}

var ipCmd = &cobra.Command{
	Use:   "ip",
	Short: "Show information about your public IP address",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		if interactive {
			return tui.IP(cmd.Context())
		}

		info, err := netinfo.LookupIP(cmd.Context())
		if err != nil {
			return err
		}

		if jsonOutput {
			return printJSON(info)
		}

		tui.PrintIPInfo(os.Stdout, info)

		return nil
	},
}

var macCmd = &cobra.Command{
	Use:   "mac [address]",
	Short: "Look up the vendor of a MAC address",
	Long: `Look up the vendor of a MAC address.

Without an address the MAC of the first active local interface is used.
Any common notation works (00:1a:2b:3c:4d:5e, 00-1A-2B-3C-4D-5E, 001a.2b3c.4d5e)
and the 3 byte vendor prefix alone is enough.`,
	Example: "  sysutil mac\n  sysutil mac 00:1a:2b\n  sysutil mac --list",
	Args:    cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if interactive {
			return tui.MAC(cmd.Context())
		}

		if listIfaces {
			return printInterfaces()
		}

		var mac string
		if len(args) == 1 {
			mac = args[0]
		} else {
			iface, err := netinfo.DefaultMAC()
			if err != nil {
				return err
			}
			mac = iface.MAC
		}

		info, err := netinfo.LookupMAC(cmd.Context(), mac)
		if err != nil {
			return err
		}

		if jsonOutput {
			return printJSON(info)
		}

		tui.PrintMACInfo(os.Stdout, mac, info)

		return nil
	},
}

func printInterfaces() error {
	ifaces, err := netinfo.Interfaces()
	if err != nil {
		return err
	}

	if jsonOutput {
		return printJSON(ifaces)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "INTERFACE\tMAC\tSTATE")
	for _, iface := range ifaces {
		state := "down"
		if iface.Up {
			state = "up"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\n", iface.Name, iface.MAC, state)
	}

	return w.Flush()
}
