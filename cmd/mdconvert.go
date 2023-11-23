package cmd

import (
	"fmt"
	"log"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
)

// Define global variables for command-line arguments.
var (
	inputFilePath  string
	outputFilePath string
)

// Initialize the 'mdconvert' subcommand and add its flags.
func init() {
	rootCmd.AddCommand(mdconvertCmd)
	mdconvertCmd.Flags().StringVarP(&inputFilePath, "input", "i", "input.md", "Input Markdown file to convert to PDF")
	mdconvertCmd.Flags().StringVarP(&outputFilePath, "output", "o", "output.pdf", "Output PDF file name")
}

// Define the 'mdconvert' subcommand.
var mdconvertCmd = &cobra.Command{
	Use:   "mdconvert",
	Short: "Convert a Markdown file to PDF using grip and Chromium browser",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		// Notify the user that the conversion process is starting
		fmt.Println("Converting Markdown file to PDF. Please wait...")

		// Validate input file name for valid extension
		if !strings.HasSuffix(inputFilePath, ".md") {
			log.Fatal("Error: Input file name must have a .md extension.")
		}

		// Validate output file name for valid extension
		if !strings.HasSuffix(outputFilePath, ".pdf") {
			log.Fatal("Error: Output file name must have a .pdf extension.")
		}

		// Launch the grip command to serve the Markdown file as a web page
		gripCmd := exec.Command("grip", inputFilePath, "10000")
		if err := gripCmd.Start(); err != nil {
			log.Fatal("Error starting grip server:", err)
		}

		// Use Chromium to print the web page to a PDF file
		chromiumCmd := exec.Command("chromium-browser", "--headless", "--no-pdf-header-footer", "--print-to-pdf="+outputFilePath, "http://127.0.0.1:10000")
		if err := chromiumCmd.Run(); err != nil {
			log.Fatal("Error starting Chromium browser:", err)
		}

		// Terminate the grip server process after completion
		if err := gripCmd.Process.Kill(); err != nil {
			log.Fatal("Error killing grip:", err)
		}

		// Inform the user that the conversion is complete
		fmt.Println("Conversion completed successfully. The PDF file is available as", outputFilePath)
	},
}
