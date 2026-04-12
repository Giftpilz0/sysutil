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

type SSHHost struct {
	Host        string
	DefaultUser string
	ExtraUsers  []string
}

func init() {
	rootCmd.AddCommand(sshlauncherCmd)
	sshlauncherCmd.Flags().StringVarP(&terminal, "terminal", "t", "", "Terminal command to use (example: kitty +kitten ssh)")
}

var sshlauncherCmd = &cobra.Command{
	Use:   "sshlauncher",
	Args:  cobra.MaximumNArgs(0),
	Short: "Launch an SSH connection using fuzzel",
	Run: func(cmd *cobra.Command, args []string) {

		sshConfigFile := os.ExpandEnv("/home/$USER/.ssh/config")

		hosts, err := getSSHHosts(sshConfigFile)
		if err != nil {
			log.Fatalf("Error reading SSH config file: %v", err)
		}

		selectedHost, err := showFuzzelHosts(hosts)
		if err != nil {
			log.Fatalf("Error displaying fuzzel: %v", err)
		}

		hostEntry := hosts[selectedHost]
		user, err := showFuzzelUsers(hostEntry)
		if err != nil {
			log.Fatalf("Error displaying fuzzel: %v", err)
		}

		err = sshToHost(user, selectedHost, terminal)
		if err != nil {
			log.Fatalf("Error executing SSH command: %v", err)
		}
	},
}

func getSSHHosts(sshConfigFile string) (map[string]SSHHost, error) {
	hosts := make(map[string]SSHHost)

	sshFile, err := os.Open(sshConfigFile)
	if err != nil {
		return nil, fmt.Errorf("could not open SSH config file %s: %w", sshConfigFile, err)
	}
	defer sshFile.Close()

	var currentHost string
	scanner := bufio.NewScanner(sshFile)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if strings.HasPrefix(line, "Include ") {
			fields := strings.Fields(line)
			if len(fields) < 2 {
				continue
			}

			for _, includePath := range fields[1:] {
				if !filepath.IsAbs(includePath) {
					includePath = filepath.Join(filepath.Dir(sshConfigFile), includePath)
				}

				matches, err := filepath.Glob(includePath)
				if err != nil {
					log.Printf("Warning: failed to expand glob %q: %v", includePath, err)
					continue
				}

				for _, match := range matches {
					subHosts, err := getSSHHosts(match)
					if err != nil {
						log.Printf("Warning: failed to parse included file %q: %v", match, err)
						continue
					}
					for k, v := range subHosts {
						hosts[k] = v
					}
				}
			}
			continue
		}

		if strings.HasPrefix(line, "Host ") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				currentHost = fields[1]
				hosts[currentHost] = SSHHost{Host: currentHost}
			}
			continue
		}

		if currentHost == "" {
			continue
		}

		if strings.HasPrefix(line, "User ") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				entry := hosts[currentHost]
				entry.DefaultUser = fields[1]
				hosts[currentHost] = entry
			}
			continue
		}

		if after, ok := strings.CutPrefix(line, "# fuzzel-users:"); ok {
			commentPart := strings.TrimSpace(after)
			commentPart = strings.TrimSpace(commentPart)
			users := strings.Split(commentPart, ",")
			var extraUsers []string
			for _, u := range users {
				u = strings.TrimSpace(u)
				if u != "" {
					extraUsers = append(extraUsers, u)
				}
			}
			if len(extraUsers) > 0 {
				entry := hosts[currentHost]
				entry.ExtraUsers = extraUsers
				hosts[currentHost] = entry
			}
			continue
		}
	}

	return hosts, nil
}

func showFuzzelHosts(hosts map[string]SSHHost) (string, error) {
	var hostList []string
	for host := range hosts {
		hostList = append(hostList, host)
	}
	sort.Strings(hostList)

	hostsString := strings.Join(hostList, "\n")

	cmd := exec.Command("fuzzel", "--prompt", "SSH hosts:", "--dmenu", "-i")
	cmd.Stdin = strings.NewReader(hostsString)

	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(output)), nil
}

func showFuzzelUsers(host SSHHost) (string, error) {
	var userChoices []string
	seen := make(map[string]bool)

	if host.DefaultUser != "" && !seen[host.DefaultUser] {
		userChoices = append(userChoices, host.DefaultUser+" (default)")
		seen[host.DefaultUser] = true
	}

	for _, user := range host.ExtraUsers {
		if !seen[user] {
			userChoices = append(userChoices, user)
			seen[user] = true
		}
	}

	if len(userChoices) == 0 {
		return "root", nil
	}

	if len(userChoices) == 1 {
		return strings.TrimSuffix(userChoices[0], " (default)"), nil
	}

	usersString := strings.Join(userChoices, "\n")

	cmd := exec.Command("fuzzel", "--prompt", "SSH user:", "--dmenu", "-i")
	cmd.Stdin = strings.NewReader(usersString)

	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	selected := strings.TrimSpace(string(output))
	selected = strings.TrimSuffix(selected, " (default)")
	return selected, nil
}

func sshToHost(user, host, terminal string) error {
	target := host
	if user != "" {
		target = user + "@" + host
	}
	return exec.Command("sh", "-c", fmt.Sprintf("%s '%s'", terminal, target)).Run()
}
