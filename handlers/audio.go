package handlers

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

const (
	pactlCmd = "pactl"
)

// AudioInfo represents aggregated information for an audio device,
// whether it’s an output (sink) or an input (source) device.
type AudioInfo struct {
	Name        string `json:"name"`
	Volume      int    `json:"volume"` // e.g. "100%"
	Mute        bool   `json:"mute"`
	Default     bool   `json:"default,omitempty"`
	Description string `json:"description"`
	Nickname    string `json:"nickname"`
}

// RawVolumeChannel represents an individual channel's volume details from pactl.
type RawVolumeChannel struct {
	DB           string `json:"db"`
	Value        int64  `json:"value"`
	ValuePercent string `json:"value_percent"`
}

// RawVolumeChannel represents an individual channel's volume details from pactl.
type RawDeviceProperties struct {
	Nickname string `json:"device.nick"`
}

// rawDevice is a common structure for unmarshaling JSON output for both sinks and sources.
type rawDevice struct {
	Name        string                      `json:"name"`
	Volume      map[string]RawVolumeChannel `json:"volume"`
	Properties  RawDeviceProperties         `json:"properties"`
	Mute        bool                        `json:"mute"`
	Description string                      `json:"description"`
}

// VolumeAction defines an action for adjusting volume and mute settings,
// and optionally setting the default device for either input (source) or output (sink).
type VolumeAction struct {
	Device  string `json:"device"`  // if empty, a default device will be chosen based on Type
	Adjust  int    `json:"adjust"`  // volume as an int (0 to 100)
	Muted   bool   `json:"muted"`   // true to mute
	Default bool   `json:"default"` // if true, set device as the default
	Type    string `json:"type"`    // "sink" or "source"; defaults to "sink" if empty
}

// aggregateVolume calculates an aggregated volume percentage from multiple channels.
func aggregateVolume(volume map[string]RawVolumeChannel) int {
	total := 0
	count := 0
	for _, channel := range volume {
		percentStr := strings.TrimSuffix(channel.ValuePercent, "%")
		if val, err := strconv.Atoi(percentStr); err == nil {
			total += val
			count++
		}
	}
	if count == 0 {
		return 0
	}
	avg := total / count
	return avg
}

// getAudioInfo retrieves audio devices info from pactl (sinks or sources)
// and marks the default device based on pactl get-default-sink/source.
func getAudioInfo(deviceType string) ([]AudioInfo, error) {
	cmd := exec.Command(pactlCmd, "--format", "json", "list", deviceType)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("error executing pactl for %s: %w", deviceType, err)
	}

	var devices []rawDevice
	if err := json.Unmarshal(output, &devices); err != nil {
		return nil, fmt.Errorf("error parsing pactl %s JSON: %w", deviceType, err)
	}

	audioInfos := make([]AudioInfo, 0, len(devices))
	for _, dev := range devices {
		audioInfos = append(audioInfos, AudioInfo{
			Name:        dev.Name,
			Volume:      aggregateVolume(dev.Volume),
			Mute:        dev.Mute,
			Description: dev.Description,
			Nickname:    dev.Properties.Nickname,
		})
	}

	// Determine the default device for sinks or sources.
	var defArg string
	if deviceType == "sinks" {
		defArg = "get-default-sink"
	} else if deviceType == "sources" {
		defArg = "get-default-source"
	}

	if defArg != "" {
		cmdDef := exec.Command(pactlCmd, defArg)
		defOutput, err := cmdDef.Output()
		if err != nil {
			return audioInfos, fmt.Errorf("error retrieving default device using %s: %w", defArg, err)
		}
		defaultName := strings.TrimSpace(string(defOutput))
		for i := range audioInfos {
			if audioInfos[i].Name == defaultName {
				audioInfos[i].Default = true
			}
		}
	}

	return audioInfos, nil
}

// GetVolumeInfo retrieves output devices (sinks) with aggregated volume.
func GetVolumeInfo() ([]AudioInfo, error) {
	return getAudioInfo("sinks")
}

// GetInputInfo retrieves input devices (sources) with aggregated volume.
func GetInputInfo() ([]AudioInfo, error) {
	return getAudioInfo("sources")
}

// ProcessAudioActions processes a JSON input that specifies volume/mute adjustments,
// and optionally sets the default audio device for both inputs and outputs.
func ProcessAudioActions(actionsJSON []byte) error {
	var actions []VolumeAction
	if err := json.Unmarshal(actionsJSON, &actions); err != nil {
		return fmt.Errorf("failed to unmarshal volume actions: %w", err)
	}

	for _, action := range actions {
		// Default the device type to "sink" if not provided.
		if action.Type == "" {
			action.Type = "sink"
		}

		// If no device is specified, use the default device for the type.
		if action.Device == "" {
			if action.Type == "source" {
				action.Device = "@DEFAULT_SOURCE@"
			} else {
				action.Device = "@DEFAULT_SINK@"
			}
		}

		// Clamp the volume adjustment between 0 and 100.
		if action.Adjust < 0 {
			action.Adjust = 0
		} else if action.Adjust > 100 {
			action.Adjust = 100
		}

		// Set mute state using pactl.
		muteVal := "0"
		if action.Muted {
			muteVal = "1"
		}
		muteCmd := exec.Command("pactl", fmt.Sprintf("set-%s-mute", action.Type), action.Device, muteVal)
		if err := muteCmd.Run(); err != nil {
			fmt.Printf("ProcessAudioActions: failed to set mute for %s %s: %v\n", action.Type, action.Device, err)
			continue
		}

		// Set volume using pactl.
		volumeStr := strconv.Itoa(action.Adjust) + "%"
		setVolumeCmd := exec.Command("pactl", fmt.Sprintf("set-%s-volume", action.Type), action.Device, volumeStr)
		if err := setVolumeCmd.Run(); err != nil {
			fmt.Printf("ProcessAudioActions: failed to set volume for %s %s: %v\n", action.Type, action.Device, err)
			continue
		}

		// If Default flag is set, update the default device using pactl.
		if action.Default {
			setDefaultCmd := exec.Command("pactl", fmt.Sprintf("set-default-%s", action.Type), action.Device)
			if err := setDefaultCmd.Run(); err != nil {
				fmt.Printf("ProcessAudioActions: failed to set default for %s %s: %v\n", action.Type, action.Device, err)
				continue
			}
		}
	}

	return nil
}
