package cmd

import (
	"fmt"
	"log"
	"os/exec"

	"github.com/spf13/cobra"
)

// Initialize the 'update' subcommand.
func init() {
	rootCmd.AddCommand(updateCmd)
}

// Define the 'update' subcommand.
var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update packages using dnf",
	Args:  cobra.MaximumNArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		// Check if the script is running as root
		if !isRoot() {
			log.Fatal("Error: This script must be executed with root privileges.")
		}

		// Notify the user that the package update process is starting
		fmt.Println("Updating packages, please wait...")

		// Create a command to execute
		updateCommand := exec.Command("sudo", "dnf", "update", "-y")

		// Execute the 'dnf update' command
		err := updateCommand.Run()
		if err != nil {
			log.Fatal("Error updating packages:", err)
		}

		// Inform the user that the package update has completed successfully
		fmt.Println("Package update completed successfully.")
	},
}
