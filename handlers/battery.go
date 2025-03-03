package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/godbus/dbus/v5"
)

type BatteryInfo struct {
	DeviceName     string
	Temperature    float64
	Percentage     float64
	IsRechargeable bool
	IsPresent      bool
}

func GetBatteryInfo() []BatteryInfo {
	conn, err := dbus.ConnectSystemBus()
	if err != nil {
		fmt.Println(err)
		return nil
	}
	defer conn.Close()

	var batteryInfos []BatteryInfo
	var devicePaths []dbus.ObjectPath

	// Enumerate devices
	err = conn.Object("org.freedesktop.UPower", "/org/freedesktop/UPower").
		Call("org.freedesktop.UPower.EnumerateDevices", 0).Store(&devicePaths)
	if err != nil {
		fmt.Println(err)
		return nil
	}

	// Iterate over each device and get its properties
	for _, path := range devicePaths {
		var temperature float64
		var percentage float64
		var isRechargeable bool
		var isPresent bool

		obj := conn.Object("org.freedesktop.UPower", path)

		// Get device properties
		temperatureVariant, err := obj.GetProperty("org.freedesktop.UPower.Device.Temperature")
		if err != nil {
			fmt.Println(err)
			continue
		}
		temperature = temperatureVariant.Value().(float64)

		percentageVariant, err := obj.GetProperty("org.freedesktop.UPower.Device.Percentage")
		if err != nil {
			fmt.Println(err)
			continue
		}
		percentage = percentageVariant.Value().(float64)

		isRechargeableVariant, err := obj.GetProperty("org.freedesktop.UPower.Device.IsRechargeable")
		if err != nil {
			fmt.Println(err)
			continue
		}
		isRechargeable = isRechargeableVariant.Value().(bool)

		isPresentVariant, err := obj.GetProperty("org.freedesktop.UPower.Device.IsPresent")
		if err != nil {
			fmt.Println(err)
			continue
		}
		isPresent = isPresentVariant.Value().(bool)

		// Append battery info
		batteryInfos = append(batteryInfos, BatteryInfo{
			DeviceName:     string(path),
			Temperature:    temperature,
			Percentage:     percentage,
			IsRechargeable: isRechargeable,
			IsPresent:      isPresent,
		})
	}

	return batteryInfos
}

func GetBattery(w http.ResponseWriter, _ *http.Request) {
	batteryInfos := GetBatteryInfo()
	json.NewEncoder(w).Encode(batteryInfos)
}
