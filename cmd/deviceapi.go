package cmd

import (
	"log"
	"net/http"

	"github.com/giftpilz0/sysutil/handlers"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(deviceapiCmd)
}

var deviceapiCmd = &cobra.Command{
	Use:   "deviceapi",
	Short: "Get informations and control some device functions (volume, network...)",
	Args:  cobra.MaximumNArgs(0),
	Run: func(cmd *cobra.Command, args []string) {

		// Register HTTP handlers.
		http.HandleFunc("/network", handlers.NetworkHandler)
		http.HandleFunc("/battery", handlers.BatteryHandler)
		http.HandleFunc("/audio/outputs", handlers.AudioOutputsHandler)
		http.HandleFunc("/audio/inputs", handlers.AudioInputsHandler)
		http.HandleFunc("/audio/actions", handlers.AudioActionsHandler)

		log.Println("HTTP server listening on :8080")
		if err := http.ListenAndServe(":8080", nil); err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}
	},
}
