package cmd

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/exec"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

var (
	terminal   string
	sshCommand string
)

// Initialize the 'wofissh' subcommand.
func init() {
	rootCmd.AddCommand(wofisshCmd)
	wofisshCmd.Flags().StringVarP(&terminal, "terminal", "t", "", "Terminal command to use")
	wofisshCmd.Flags().StringVarP(&sshCommand, "ssh-command", "s", "ssh", "SSH command to use")
}

// Define the 'wofissh' subcommand.
var wofisshCmd = &cobra.Command{
	Use:   "wofissh",
	Args:  cobra.MaximumNArgs(0),
	Short: "Launch an SSH connection using wofi",
	Run: func(cmd *cobra.Command, args []string) {
		// Get hosts from the ssh config file
		sshConfigFile := os.ExpandEnv("/home/$USER/.ssh/config")
		hosts, err := getHosts(sshConfigFile)
		if err != nil {
			log.Fatalf("Error reading SSH config file: %v", err)
		}

		// Show wofi dialog to select a host
		selectedHost, err := showWofi(hosts)
		if err != nil {
			log.Fatalf("Error displaying wofi: %v", err)
		}

		// Execute SSH to the selected host in the specified terminal
		err = sshToHost(selectedHost, terminal, sshCommand)
		if err != nil {
			log.Fatalf("Error executing SSH command: %v", err)
		}
	},
}

// Function to get a list of all hosts from the ssh config.
func getHosts(sshConfigFile string) ([]string, error) {
	var hosts []string

	// Open the SSH config file
	sshFile, err := os.Open(sshConfigFile)
	if err != nil {
		return nil, fmt.Errorf("could not open SSH config file: %w", err)
	}
	defer sshFile.Close()

	// Read the file line by line
	scanner := bufio.NewScanner(sshFile)
	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "Host ") {
			parts := strings.Fields(line)
			hosts = append(hosts, parts[1:]...)
		}
	}

	sort.Strings(hosts)

	return hosts, nil
}

// Function to run the wofi command with the list of hosts.
func showWofi(hosts []string) (string, error) {
	hostsString := strings.Join(hosts, "\n")

	cmd := exec.Command("wofi", "--prompt", "SSH hosts:", "--dmenu", "--insensitive")
	cmd.Stdin = strings.NewReader(hostsString)

	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(output)), nil
}

// Function to execute the SSH command in the specified terminal.
func sshToHost(host string, terminal string, sshCommand string) error {
	err := exec.Command("sh", "-c", fmt.Sprintf("%s '%s %s'", terminal, sshCommand, host)).Run()

	return err
}
