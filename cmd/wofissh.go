package cmd

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

var (
	terminal   string
	sshCommand string
)

func init() {
	rootCmd.AddCommand(wofisshCmd)
	wofisshCmd.Flags().StringVarP(&terminal, "terminal", "t", "", "Terminal command to use (example: kitty +kitten ssh)")
}

var wofisshCmd = &cobra.Command{
	Use:   "wofissh",
	Args:  cobra.MaximumNArgs(0),
	Short: "Launch an SSH connection using wofi",
	Run: func(cmd *cobra.Command, args []string) {

		sshConfigFile := os.ExpandEnv("/home/$USER/.ssh/config")

		hosts, err := getHosts(sshConfigFile)
		if err != nil {
			log.Fatalf("Error reading SSH config file: %v", err)
		}

		selectedHost, err := showWofi(hosts)
		if err != nil {
			log.Fatalf("Error displaying wofi: %v", err)
		}

		err = sshToHost(selectedHost, terminal)
		if err != nil {
			log.Fatalf("Error executing SSH command: %v", err)
		}
	},
}

func getHosts(sshConfigFile string) ([]string, error) {
	var hosts []string

	sshFile, err := os.Open(sshConfigFile)
	if err != nil {
		return nil, fmt.Errorf("could not open SSH config file %s: %w", sshConfigFile, err)
	}
	defer sshFile.Close()

	scanner := bufio.NewScanner(sshFile)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if strings.HasPrefix(line, "Include ") {
			fields := strings.Fields(line)
			if len(fields) < 2 {
				continue
			}

			for _, incrementalPath := range fields[1:] {
				if !filepath.IsAbs(incrementalPath) {
					incrementalPath = filepath.Join(filepath.Dir(sshConfigFile), incrementalPath)
				}

				matches, err := filepath.Glob(incrementalPath)
				if err != nil {
					log.Printf("Warning: failed to expand glob %q: %v", incrementalPath, err)
					continue
				}

				for _, match := range matches {
					subHosts, err := getHosts(match)
					if err != nil {
						log.Printf("Warning: failed to parse included file %q: %v", match, err)
						continue
					}
					hosts = append(hosts, subHosts...)
				}
			}
			continue
		}

		if strings.HasPrefix(line, "Host ") {
			fields := strings.Fields(line)
			hosts = append(hosts, fields[1:]...)
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
func sshToHost(host, terminal string) error {
	return exec.Command("sh", "-c", fmt.Sprintf("%s '%s'", terminal, host)).Run()
}
