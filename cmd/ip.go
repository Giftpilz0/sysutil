package cmd

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/spf13/cobra"
)

// Define a struct to represent the response from the IP lookup API.
type IPInfo struct {
	Ip  string `json:"ip"`
	Loc string `json:"loc"`
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
		// Notify the user that the information is being fetched
		fmt.Println("\nFetching information, please wait...\n____________________")

		// Define the API URL
		apiURL := "https://1.1.1.1/cdn-cgi/trace"

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

		// Parse the response
		lines := strings.Split(strings.TrimSpace(string(responseBody)), "\n")
		data := make(map[string]string)

		for _, line := range lines {
			parts := strings.Split(line, "=")
			if len(parts) != 2 {
				continue
			}

			data[parts[0]] = parts[1]
		}

		// Initialize a IPInfo struct to store the data
		var ipInfo IPInfo

		for key, value := range data {
			switch key {
			case "ip":
				ipInfo.Ip = value
			case "loc":
				ipInfo.Loc = value
			}
		}

		// Print the retrieved IP information
		fmt.Println("IP Address:", ipInfo.Ip)
		fmt.Println("Country:", ipInfo.Loc)
	},
}
