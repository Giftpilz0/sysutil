package cmd

import (
	"fmt"
	"log"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
)

var (
	inputFilePath  string
	outputFilePath string
)

func init() {
	rootCmd.AddCommand(mdconvertCmd)
	mdconvertCmd.Flags().StringVarP(&inputFilePath, "input", "i", "input.md", "Input Markdown file to convert to PDF")
	mdconvertCmd.Flags().StringVarP(&outputFilePath, "output", "o", "output.pdf", "Output PDF file name")
}

var mdconvertCmd = &cobra.Command{
	Use:   "mdconvert",
	Short: "Convert a Markdown file to PDF using grip and Chromium browser",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Converting Markdown file to PDF. Please wait...")

		if !strings.HasSuffix(inputFilePath, ".md") {
			log.Fatal("Error: Input file name must have a .md extension.")
		}

		if !strings.HasSuffix(outputFilePath, ".pdf") {
			log.Fatal("Error: Output file name must have a .pdf extension.")
		}

		gripCmd := exec.Command("grip", inputFilePath, "10000")
		if err := gripCmd.Start(); err != nil {
			log.Fatal("Error starting grip server:", err)
		}

		chromiumCmd := exec.Command("chromium-browser", "--headless", "--no-pdf-header-footer", "--print-to-pdf="+outputFilePath, "http://127.0.0.1:10000")
		if err := chromiumCmd.Run(); err != nil {
			log.Fatal("Error starting Chromium browser:", err)
		}

		if err := gripCmd.Process.Kill(); err != nil {
			log.Fatal("Error killing grip:", err)
		}

		fmt.Println("Conversion completed successfully. The PDF file is available as", outputFilePath)
	},
}
