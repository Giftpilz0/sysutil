package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/spf13/cobra"
)

// Define a struct to represent the response from the IP lookup API.
type IPInfo struct {
	IP       string
	City     string
	Country  string
	Timezone string
	Org      string
}

// Initialize the 'ip' subcommand.
func init() {
	rootCmd.AddCommand(ipCmd)
}

// Define the 'ip' subcommand.
var ipCmd = &cobra.Command{
	Use:   "ip",
	Short: "Get information about your public IP address",
	Args:  cobra.MaximumNArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		// Define the API URL
		apiURL := "https://ipinfo.io/json"

		// Send an HTTP GET request to the API
		resp, err := http.Get(apiURL)
		if err != nil {
			log.Fatal("Error fetching the API:", err)
		}
		defer resp.Body.Close()

		// Read the entire response body into a byte slice
		responseBody, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Fatal("Error reading the API response:", err)
		}

		var ipInfo IPInfo
		err = json.Unmarshal(responseBody, &ipInfo)
		if err != nil {
			log.Fatal("Error fetching the API:", err)
		}

		// Print the retrieved IP information
		fmt.Println("IP Address:", ipInfo.IP)
		fmt.Println("City:", ipInfo.City)
		fmt.Println("Country:", ipInfo.Country)
		fmt.Println("Provider:", ipInfo.Org)
		fmt.Println("Timezone:", ipInfo.Timezone)
	},
}
