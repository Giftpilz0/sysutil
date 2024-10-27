package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"regexp"
	"strconv"
)

// Volume represents a volume control entity with a device name and a level.
type Volume struct {
	Device string // Name of the audio device
	Level  string // Desired volume level as a string
}

// ParseVolumeJSON reads the HTTP request body and unmarshals it into a Volume struct.
func ParseVolumeJSON(r *http.Request) *Volume {
	var volume Volume
	responseBody, _ := io.ReadAll(r.Body)
	json.Unmarshal(responseBody, &volume)
	return &volume
}

// GetVolume writes the current volume level of the specified audio device to the response.
func GetVolume(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "%f", GetVolumeHelper(r))
}

// GetVolumeHelper retrieves the current volume level for the specified audio device.
func GetVolumeHelper(r *http.Request) float64 {
	volume := ParseVolumeJSON(r)
	if volume.Device == "" {
		volume.Device = "@DEFAULT_SINK@"
	}
	getVolumeCommand := exec.Command("wpctl", "get-volume", volume.Device)
	volumeOutput, _ := getVolumeCommand.Output()
	re := regexp.MustCompile(`Volume:\s([\d.]+)`) // Regex to extract volume level
	volumeRegex := re.FindStringSubmatch(string(volumeOutput))
	volumeLevel, _ := strconv.ParseFloat(volumeRegex[1], 64) // Convert extracted string to float
	return volumeLevel
}

// SetVolume writes the new volume level to the specified audio device and returns the new level.
func SetVolume(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "%f", SetVolumeHelper(r))
}

// SetVolumeHelper sets the volume level for the specified audio device and returns the updated level.
func SetVolumeHelper(r *http.Request) float64 {
	volume := ParseVolumeJSON(r)
	if volume.Device == "" {
		volume.Device = "@DEFAULT_SINK@"
	}
	setVolumeCommand := exec.Command("wpctl", "set-volume", volume.Device, volume.Level)
	setVolumeCommand.Run()
	return GetVolumeHelper(r)
}

// ToggleVolumeMute toggles the mute state of the specified audio device.
func ToggleVolumeMute(w http.ResponseWriter, r *http.Request) {
	volume := ParseVolumeJSON(r)
	if volume.Device == "" {
		volume.Device = "@DEFAULT_SINK@"
	}
	toggleVolumeMuteCommand := exec.Command("wpctl", "set-mute", volume.Device, "toggle")
	toggleVolumeMuteCommand.Run()
}
