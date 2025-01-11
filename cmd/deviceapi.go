package cmd

import (
	"log"
	"net/http"

	"github.com/giftpilz0/sysutil/handlers"
	"github.com/gorilla/mux"
	"github.com/spf13/cobra"
)

// Initialize the 'deviceapi' subcommand.
func init() {
	snapshotUpdated = true

	rootCmd.AddCommand(deviceapiCmd)
}

// Define the 'deviceapi' subcommand.
var deviceapiCmd = &cobra.Command{
	Use:   "deviceapi",
	Short: "Get informations and control some device functions (volume, network...)",
	Args:  cobra.MaximumNArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		router := mux.NewRouter()

		router.HandleFunc("/volume", handlers.GetVolume).Methods("GET")
		router.HandleFunc("/volume", handlers.SetVolume).Methods("POST")
		router.HandleFunc("/volume/toggle/mute", handlers.ToggleVolumeMute).Methods("GET")
		router.HandleFunc("/network/ssid", handlers.GetSSID).Methods("GET")
		router.HandleFunc("/network/signal", handlers.GetWifiSignalStrength).Methods("GET")
		router.HandleFunc("/network/ip", handlers.GetIP).Methods("GET")
		router.HandleFunc("/network/toggle/wifi", handlers.ToggleWifi).Methods("GET")

		log.Fatal(http.ListenAndServe(":8080", router))
	},
}
