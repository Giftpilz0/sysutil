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

type HostEntry struct {
	Arguments   string `yaml:"arguments"`
	Application string `yaml:"application"`
}

type HostConfig struct {
	Hosts map[string][]HostEntry `yaml:"hosts"`
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

		selectedHost, selectedEntry, err := waypipeShowWofi(hosts)
		if err != nil {
			log.Fatalf("Error displaying wofi: %v", err)
		}

		finalArgs := selectedEntry.Arguments
		if flagArguments != "" {
			finalArgs = flagArguments
		}

		err = callWaypipe(finalArgs, selectedHost, selectedEntry.Application)
		if err != nil {
			log.Fatalf("Error executing waypipe command: %v", err)
		}
	},
}

func waypipeGetHosts(configPath string) (map[string][]HostEntry, error) {
	file, err := os.Open(configPath)
	if err != nil {
		return nil, fmt.Errorf("could not open config file: %w", err)
	}
	defer file.Close()

	var config HostConfig
	if err = yaml.NewDecoder(file).Decode(&config); err != nil {
		return nil, fmt.Errorf("could not parse YAML file: %w", err)
	}

	return config.Hosts, nil
}

type displayChoice struct {
	HostName string
	Entry    HostEntry
	Display  string
}

func waypipeShowWofi(hosts map[string][]HostEntry) (string, HostEntry, error) {
	var choices []displayChoice

	var keys []string
	for key := range hosts {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		for _, entry := range hosts[key] {
			display := fmt.Sprintf("%s (%s)", key, entry.Application)
			choices = append(choices, displayChoice{HostName: key, Entry: entry, Display: display})
		}
	}

	// Prepare the input for wofi.
	var displayList []string
	for _, choice := range choices {
		displayList = append(displayList, choice.Display)
	}

	cmd := exec.Command("wofi", "--prompt", "Waypipe hosts:", "--dmenu", "--insensitive")
	cmd.Stdin = strings.NewReader(strings.Join(displayList, "\n"))

	output, err := cmd.Output()
	if err != nil {
		return "", HostEntry{}, err
	}

	selected := strings.TrimSpace(string(output))
	for _, choice := range choices {
		if choice.Display == selected {
			return choice.HostName, choice.Entry, nil
		}
	}

	return "", HostEntry{}, fmt.Errorf("no matching entry found for selection %q", selected)
}

func callWaypipe(argsStr, host, application string) error {
	args := []string{}
	if argsStr != "" {
		args = append(args, strings.Fields(argsStr)...)
	}
	args = append(args, "ssh", host, application)

	cmd := exec.Command("waypipe", args...)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	return cmd.Run()
}
