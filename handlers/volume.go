package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

type VolumeInfo struct {
	Device string  `json:"device"`
	Level  float64 `json:"level"`
	Muted  bool    `json:"muted"`
}

type VolumeAction struct {
	Device string `json:"device"`
	Muted  bool   `json:"muted"`
	Adjust string `json:"adjust"`
}

func GetVolumeInfo() []VolumeInfo {
	var volumeInfos []VolumeInfo

	devices := []string{"@DEFAULT_SINK@", "@DEFAULT_SOURCE@"}

	for _, device := range devices {
		getVolumeCommand := exec.Command("wpctl", "get-volume", device)
		volumeOutput, _ := getVolumeCommand.Output()
		re := regexp.MustCompile(`Volume:\s([\d.]+)`)
		volumeRegex := re.FindStringSubmatch(string(volumeOutput))
		volumeLevel, _ := strconv.ParseFloat(volumeRegex[1], 2)
		muted := strings.Contains(string(volumeOutput), "[MUTED]")

		volumeInfos = append(volumeInfos, VolumeInfo{
			Device: device,
			Level:  volumeLevel,
			Muted:  muted,
		})
	}

	return volumeInfos
}

func SetVolumeHelper(volumeActions []VolumeAction) {
	for entry := range volumeActions {
		if volumeActions[entry].Device == "" {
			volumeActions[entry].Device = "@DEFAULT_SINK@"
		}

		volumeFloat, err := strconv.ParseFloat(volumeActions[entry].Adjust, 2)
		if err != nil {
			fmt.Printf("SetVolumeHelper: %s isnt a float\n", volumeActions[entry].Adjust)
		} else {
			if volumeFloat >= 1 {
				volumeActions[entry].Adjust = "1"
			} else if volumeFloat < 0 {
				volumeActions[entry].Adjust = "0"
			}
		}

		if volumeActions[entry].Muted == true {
			toggleMuteCommand := exec.Command("wpctl", "set-mute", volumeActions[entry].Device, "1")
			toggleMuteCommand.Run()
		} else {
			toggleMuteCommand := exec.Command("wpctl", "set-mute", volumeActions[entry].Device, "0")
			toggleMuteCommand.Run()
		}

		setVolumeCommand := exec.Command("wpctl", "set-volume", volumeActions[entry].Device, volumeActions[entry].Adjust)
		setVolumeCommand.Run()

	}
}

func GetVolume(w http.ResponseWriter, r *http.Request) {
	volumeInfo := GetVolumeInfo()
	json.NewEncoder(w).Encode(volumeInfo)
}

func SetVolume(w http.ResponseWriter, r *http.Request) {
	var volumeActions []VolumeAction
	err := json.NewDecoder(r.Body).Decode(&volumeActions)
	if err != nil {
		return
	}

	SetVolumeHelper(volumeActions)
}
