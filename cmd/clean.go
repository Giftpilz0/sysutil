package cmd

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
)

// Define global variables for command-line arguments.
var (
	noPrompt bool
	journal  bool
)

// Initialize the 'clean' subcommand and add its flags.
func init() {
	rootCmd.AddCommand(cleanCmd)
	cleanCmd.Flags().BoolVarP(&noPrompt, "yes", "y", false, "Skip confirmation prompts before deleting files")
	cleanCmd.Flags().BoolVarP(&journal, "journalctl", "j", false, "Clean journalctl logs")
}

// Function to format and normalize user input.
func formatInput(input string) string {
	// Trim leading and trailing spaces
	input = strings.TrimSpace(input)

	// Convert input to lowercase
	input = strings.ToLower(input)

	return input
}

// Define the 'clean' subcommand.
var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Remove unnecessary files",
	Args:  cobra.MaximumNArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		// List of directories to clean
		cleanPaths := []string{
			getHomeDirectory() + "/.cache",
			getHomeDirectory() + "/.java",
			getHomeDirectory() + "/.npm",
			getHomeDirectory() + "/.ansible",
			getHomeDirectory() + "/.bash_history",
			getHomeDirectory() + "/.local/share/Trash",
		}

		// Notify the user that file deletion is in progress
		fmt.Println("\nCleaning up files...\n------------------------------")

		// Variables to store user input
		var confirm string
		var deleteAll bool

		// Check if the user has opted to skip confirmation prompts
		if noPrompt {
			deleteAll = true
		} else {
			// Prompt the user to confirm deleting all files
			fmt.Print("Delete all files (y/n): ")
			fmt.Scan(&confirm)

			// Format and normalize user input
			confirm = formatInput(confirm)

			if confirm == "y" || confirm == "yes" {
				deleteAll = true
			}
		}

		// Iterate through the list of directories to clean
		for _, dir := range cleanPaths {
			// If the user has opted to delete all files or skipped confirmation prompts, proceed with deleting the directory
			if deleteAll {
				err := os.RemoveAll(dir)
				if err != nil {
					log.Fatal("Error deleting directory:", dir, err)
				}
				// Display the directory being cleaned
				fmt.Println(dir)
			} else {
				// Prompt the user for confirmation before deleting each directory
				fmt.Print("Delete this directory? (y/n) | ", dir, " | ")
				fmt.Scan(&confirm)

				// Format and normalize user input
				confirm = formatInput(confirm)

				// If the user confirms deletion, proceed with removing the directory
				if confirm == "y" || confirm == "yes" {
					err := os.RemoveAll(dir)
					if err != nil {
						log.Fatal("Error deleting directory:", dir, err)
					}
				}
			}
		}

		// If the user has enabled journalctl cleaning, clean up journalctl logs
		if journal {
			fmt.Println("\nCleaning up journalctl logs...")

			// Create a command to execute journalctl vacuum
			journalCmd := exec.Command("sudo", "journalctl", "--vacuum-time", "1d")

			// Execute the journalctl vacuum command
			err := journalCmd.Run()
			if err != nil {
				log.Fatal("Error cleaning the system journal:", err)
			}
		}
	},
}
