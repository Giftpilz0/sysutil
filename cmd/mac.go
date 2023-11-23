package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/spf13/cobra"
)

// Define global variables for command-line arguments.
var macAddress string

// Define a struct to represent the JSON response from the MAC lookup API.
type MACInfo struct {
	MacPrefix string `json:"macPrefix"`
	Company   string `json:"company"`
	Address   string `json:"address"`
	Country   string `json:"country"`
}

// Initialize the 'mac' subcommand and add its flags.
func init() {
	rootCmd.AddCommand(macCmd)
	macCmd.Flags().StringVarP(&macAddress, "mac", "m", "000000", "Set MAC address to look up")
}

// Function to shorten the MAC address by removing non-essential characters.
func shortenMAC(macAddress string) string {
	if len(macAddress) < 6 {
		return macAddress
	}
	return macAddress[:6]
}

// Define the 'mac' subcommand.
var macCmd = &cobra.Command{
	Use:   "mac",
	Short: "Get information about a MAC address",
	Args:  cobra.MaximumNArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		// Notify the user that the information is being fetched
		fmt.Println("\nFetching information, please wait...\n____________________")

		// Shorten the MAC address by removing non-essential characters
		macAddress = shortenMAC(strings.ReplaceAll(macAddress, ":", ""))

		// Construct the API URL for the given MAC address
		apiURL := "https://api.maclookup.app/v2/macs/" + macAddress

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

		// Initialize a MACInfo struct to store the parsed JSON data
		var macInfo MACInfo

		// Unmarshal the JSON response into the MacInfo struct
		err = json.Unmarshal(responseBody, &macInfo)
		if err != nil {
			log.Fatal("Error parsing JSON:", err)
		}

		// Print the retrieved MAC information
		fmt.Println("MacPrefix:", macInfo.MacPrefix)
		fmt.Println("Company:", macInfo.Company)
		fmt.Println("Country:", macInfo.Country)
		fmt.Println("Address:", macInfo.Address)
	},
}
