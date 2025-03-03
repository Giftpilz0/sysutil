package cmd

import (
	"log"
	"net/http"

	"github.com/giftpilz0/sysutil/handlers"
	"github.com/gorilla/mux"
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

		router := mux.NewRouter()

		router.HandleFunc("/volume", handlers.GetVolume).Methods("GET")
		router.HandleFunc("/volume", handlers.SetVolume).Methods("POST")

		router.HandleFunc("/network", handlers.GetNetwork).Methods("GET")
		router.HandleFunc("/network/toggle/wifi", handlers.ToggleWifi).Methods("GET")

		router.HandleFunc("/battery", handlers.GetBattery).Methods("GET")

		log.Fatal(http.ListenAndServe(":8080", router))
	},
}
