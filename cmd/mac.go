package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
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

	defaultMac, err := getDefaultMACAddress()
	if err != nil {
		log.Fatalf("Error retrieving system MAC address: %v", err)
	}

	macCmd.Flags().StringVarP(&macAddress, "mac", "m", defaultMac, "Set MAC address to look up")
}

// Define the 'mac' subcommand.
var macCmd = &cobra.Command{
	Use:   "mac",
	Short: "Get information about a MAC address",
	Args:  cobra.MaximumNArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		// Notify the user that the information is being fetched
		fmt.Println("\nFetching information, please wait...\n____________________")

		// Get MAC information
		macInfo, err := getMACInfo(macAddress)
		if err != nil {
			log.Fatal("Error getting MAC information:", err)
		}

		// Print the retrieved MAC information
		fmt.Println("MacPrefix:", macInfo.MacPrefix)
		fmt.Println("Company:", macInfo.Company)
		fmt.Println("Country:", macInfo.Country)
		fmt.Println("Address:", macInfo.Address)
	},
}

// Function to shorten the MAC address by removing non-essential characters.
func shortenMAC(macAddress string) string {
	if len(macAddress) < 6 {
		return macAddress
	}

	return macAddress[:6]
}

// Function to get MAC information for a given MAC address.
func getMACInfo(macAddress string) (MACInfo, error) {
	// Shorten the MAC address by removing non-essential characters
	macAddress = shortenMAC(strings.ReplaceAll(macAddress, ":", ""))

	// Construct the API URL for the given MAC address
	apiURL := "https://api.maclookup.app/v2/macs/" + macAddress

	// Send an HTTP GET request to the API
	resp, err := http.Get(apiURL)
	if err != nil {
		return MACInfo{}, err
	}
	defer resp.Body.Close()

	// Read the entire response body into a byte slice
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return MACInfo{}, err
	}

	// Initialize a MACInfo struct to store the parsed JSON data
	var macInfo MACInfo

	// Unmarshal the JSON response into the MacInfo struct
	err = json.Unmarshal(responseBody, &macInfo)
	if err != nil {
		return MACInfo{}, err
	}

	return macInfo, nil
}

// Function to get the default MAC address of the system.
func getDefaultMACAddress() (string, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}

	for _, iface := range interfaces {
		// Skip loopback interfaces and interfaces without a MAC address.
		if iface.Flags&net.FlagLoopback == 0 && len(iface.HardwareAddr) > 0 {
			return iface.HardwareAddr.String(), nil
		}
	}

	return "", fmt.Errorf("no valid network interface found")
}
