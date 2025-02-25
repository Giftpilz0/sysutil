package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/spf13/cobra"
)

type IPInfo struct {
	IP       string `json:"ip"`
	City     string `json:"city"`
	Country  string `json:"country"`
	Timezone string `json:"timezone"`
	Org      string `json:"org"`
}

func init() {
	rootCmd.AddCommand(ipCmd)
}

var ipCmd = &cobra.Command{
	Use:   "ip",
	Short: "Get information about your public IP address",
	Args:  cobra.MaximumNArgs(0),
	Run: func(cmd *cobra.Command, args []string) {

		apiURL := "https://ipinfo.io/json"

		resp, err := http.Get(apiURL)
		if err != nil {
			log.Fatal("Error fetching the API:", err)
		}
		defer resp.Body.Close()

		responseBody, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Fatal("Error reading the API response:", err)
		}

		var ipInfo IPInfo
		err = json.Unmarshal(responseBody, &ipInfo)
		if err != nil {
			log.Fatal("Error fetching the API:", err)
		}

		fmt.Println("IP Address:", ipInfo.IP)
		fmt.Println("City:", ipInfo.City)
		fmt.Println("Country:", ipInfo.Country)
		fmt.Println("Provider:", ipInfo.Org)
		fmt.Println("Timezone:", ipInfo.Timezone)
	},
}
