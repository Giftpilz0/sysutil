package cmd

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var (
	flagArguments string
)

type Host struct {
	Arguments   string `yaml:"arguments"`
	Application string `yaml:"application"`
}

type HostConfig struct {
	Hosts map[string]Host `yaml:"hosts"`
}

func init() {
	rootCmd.AddCommand(wofiwaypipeCmd)
	wofiwaypipeCmd.Flags().StringVarP(&flagArguments, "arguments", "a", "", "Commandline arguments for waypipe")
}

var wofiwaypipeCmd = &cobra.Command{
	Use:   "wofiwaypipe",
	Args:  cobra.MaximumNArgs(0),
	Short: "Launch a waypipe connection using wofi",
	Run: func(cmd *cobra.Command, args []string) {
		waypipeConfigFile := os.ExpandEnv("/home/$USER/.ssh/waypipe.yaml")

		hosts, err := waypipeGetHosts(waypipeConfigFile)
		if err != nil {
			log.Fatalf("Error reading waypipe config file: %v", err)
		}

		selectedHost, err := waypipeShowWofi(hosts)
		if err != nil {
			log.Fatalf("Error displaying wofi: %v", err)
		}

		hostConf, ok := hosts[selectedHost]
		if !ok {
			log.Fatalf("Selected host %s doesn't exist in config", selectedHost)
		}

		// If the flag is provided, override the YAML-based arguments.
		finalArgs := hostConf.Arguments
		if flagArguments != "" {
			finalArgs = flagArguments
		}

		err = waypipeToHost(finalArgs, selectedHost, hostConf.Application)
		if err != nil {
			log.Fatalf("Error executing waypipe command: %v", err)
		}
	},
}

// waypipeGetHosts parses the YAML file into a HostConfig struct and returns the Hosts map.
func waypipeGetHosts(waypipeConfigFile string) (map[string]Host, error) {
	file, err := os.Open(waypipeConfigFile)
	if err != nil {
		return nil, fmt.Errorf("could not open waypipe config file: %w", err)
	}
	defer file.Close()

	var config HostConfig
	if err = yaml.NewDecoder(file).Decode(&config); err != nil {
		return nil, fmt.Errorf("could not parse YAML file: %w", err)
	}

	return config.Hosts, nil
}

// waypipeShowWofi displays the host names via wofi and returns the selected host name.
func waypipeShowWofi(hosts map[string]Host) (string, error) {
	var hostNames []string
	for name := range hosts {
		hostNames = append(hostNames, name)
	}
	sort.Strings(hostNames)

	cmd := exec.Command("wofi", "--prompt", "Waypipe hosts:", "--dmenu", "--insensitive")
	cmd.Stdin = strings.NewReader(strings.Join(hostNames, "\n"))

	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(output)), nil
}

// waypipeToHost executes the waypipe command using the provided arguments, host, and application.
// It splits any extra arguments into separate tokens and appends fixed tokens ("ssh", host, application).
func waypipeToHost(argsStr, host, application string) error {
	// Start with any extra arguments (split into fields).
	args := []string{}
	if argsStr != "" {
		args = append(args, strings.Fields(argsStr)...)
	}
	// Append the fixed command arguments.
	args = append(args, "ssh", host, application)

	cmd := exec.Command("waypipe", args...)
	// Pipe the standard streams so that the command output is visible.
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	return cmd.Run()
}
