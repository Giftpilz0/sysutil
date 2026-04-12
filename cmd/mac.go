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

var macAddress string

type MACInfo struct {
	MacPrefix string `json:"macPrefix"`
	Company   string `json:"company"`
	Address   string `json:"address"`
	Country   string `json:"country"`
}

func init() {
	rootCmd.AddCommand(macCmd)

	defaultMac, err := getDefaultMACAddress()
	if err != nil {
		log.Fatalf("Error retrieving system MAC address: %v", err)
	}

	macCmd.Flags().StringVarP(&macAddress, "mac", "m", defaultMac, "Set MAC address to look up")
}

var macCmd = &cobra.Command{
	Use:   "mac",
	Short: "Get information about a MAC address",
	Args:  cobra.MaximumNArgs(0),
	Run: func(cmd *cobra.Command, args []string) {

		macInfo, err := getMACInfo(macAddress)
		if err != nil {
			log.Fatal("Error getting MAC information:", err)
		}

		fmt.Println("MacPrefix:", macInfo.MacPrefix)
		fmt.Println("Company:", macInfo.Company)
		fmt.Println("Country:", macInfo.Country)
		fmt.Println("Address:", macInfo.Address)
	},
}

func getMACInfo(macAddress string) (MACInfo, error) {

	macAddress = strings.ReplaceAll(macAddress, ":", "")[:6]
	apiURL := "https://api.maclookup.app/v2/macs/" + macAddress

	resp, err := http.Get(apiURL)
	if err != nil {
		return MACInfo{}, err
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return MACInfo{}, err
	}

	var macInfo MACInfo
	err = json.Unmarshal(responseBody, &macInfo)
	if err != nil {
		return MACInfo{}, err
	}

	return macInfo, nil
}

func getDefaultMACAddress() (string, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}

	for _, iface := range interfaces {
		if iface.Flags&net.FlagLoopback == 0 && len(iface.HardwareAddr) > 0 {
			return iface.HardwareAddr.String(), nil
		}
	}

	return "", fmt.Errorf("no valid network interface found")
}
