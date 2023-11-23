package cmd

import (
	"log"

	"github.com/spf13/cobra"
)

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

// Define the root command name and add a short description.
var rootCmd = &cobra.Command{
	Use:   "sysutil",
	Short: "sysutil is a collection of utilities to simplify various tasks.",
}
