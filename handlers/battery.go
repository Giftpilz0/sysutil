package handlers

import (
	"fmt"

	"github.com/godbus/dbus/v5"
)

const (
	upowerService         = "org.freedesktop.UPower"
	upowerPath            = "/org/freedesktop/UPower"
	upowerDeviceInterface = "org.freedesktop.UPower.Device"

	// UPower device type for battery.
	batteryDeviceType int32 = 2
)

// Battery holds battery status information.
type Battery struct {
	Percentage float64 `json:"percentage"`
	State      string  `json:"state"`
}

// BatteryStateToString converts a battery state (integer) into a human‑readable string.
func BatteryStateToString(state int32) string {
	switch state {
	case 0:
		return "Unknown"
	case 1:
		return "Charging"
	case 2:
		return "Discharging"
	case 3:
		return "Empty"
	case 4:
		return "Fully charged"
	case 5:
		return "Pending"
	case 6:
		return "Not charging"
	default:
		return "Unknown"
	}
}

// GetBatteryStatus retrieves battery information using UPower via DBus.
func GetBatteryStatus() (Battery, error) {
	var battery Battery

	conn, err := dbus.SystemBus()
	if err != nil {
		return battery, fmt.Errorf("failed to connect to system DBus for battery: %w", err)
	}

	upowerObj := conn.Object(upowerService, dbus.ObjectPath(upowerPath))

	// Enumerate UPower devices.
	var devicePaths []dbus.ObjectPath
	if err = upowerObj.Call("org.freedesktop.UPower.EnumerateDevices", 0).Store(&devicePaths); err != nil {
		return battery, fmt.Errorf("failed to enumerate UPower devices: %w", err)
	}

	for _, path := range devicePaths {
		devObj := conn.Object(upowerService, path)

		// Retrieve the device type.
		typeVar, err := GetProperty(devObj, upowerDeviceInterface, "Type")
		if err != nil {
			continue
		}

		var devType int32
		switch t := typeVar.Value().(type) {
		case int32:
			devType = t
		case uint32:
			devType = int32(t)
		default:
			continue
		}

		// Check if the device is a battery.
		if devType != batteryDeviceType {
			continue
		}

		// Retrieve battery percentage.
		percVar, err := GetProperty(devObj, upowerDeviceInterface, "Percentage")
		if err != nil {
			return battery, fmt.Errorf("failed to get battery percentage: %w", err)
		}
		percentage, ok := percVar.Value().(float64)
		if !ok {
			continue
		}

		// Retrieve battery state.
		stateVar, err := GetProperty(devObj, upowerDeviceInterface, "State")
		if err != nil {
			return battery, fmt.Errorf("failed to get battery state: %w", err)
		}
		var stateInt int32
		switch s := stateVar.Value().(type) {
		case int32:
			stateInt = s
		case uint32:
			stateInt = int32(s)
		default:
			stateInt = 0
		}

		battery.Percentage = percentage
		battery.State = BatteryStateToString(stateInt)
		return battery, nil
	}

	return battery, nil
}
