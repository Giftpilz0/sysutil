package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/godbus/dbus/v5"
)

// NM_DEVICE_TYPE_WIFI | Wi-Fi device type.
const NM_DEVICE_TYPE_WIFI = 2

type NetworkInfo struct {
	DeviceName   string
	Interface    string
	IpAddress    string
	WifiSSID     string
	DeviceType   uint32
	WifiStrength uint8
}

func GetNetworkInfo() []NetworkInfo {
	conn, err := dbus.ConnectSystemBus()
	if err != nil {
		fmt.Println(err)
		return nil
	}
	defer conn.Close()

	var networkInfos []NetworkInfo
	var devicePaths []dbus.ObjectPath

	// Enumerate devices
	err = conn.Object("org.freedesktop.NetworkManager", "/org/freedesktop/NetworkManager").
		Call("org.freedesktop.NetworkManager.GetAllDevices", 0).Store(&devicePaths)
	if err != nil {
		fmt.Println(err)
		return nil
	}

	// Iterate over each device and get its properties
	for _, path := range devicePaths {
		var interfaceName string
		var deviceType uint32
		var ipAddress string
		var wifiSSID string
		var wifiStrength uint8

		var addressData []map[string]dbus.Variant
		var ip4ConfigPath dbus.ObjectPath
		var activeApPath dbus.ObjectPath

		obj := conn.Object("org.freedesktop.NetworkManager", path)

		// Get device properties
		interfaceNameVariant, err := obj.GetProperty("org.freedesktop.NetworkManager.Device.Interface")
		if err != nil {
			fmt.Println(err)
			continue
		}
		interfaceName = interfaceNameVariant.Value().(string)

		deviceTypeVariant, err := obj.GetProperty("org.freedesktop.NetworkManager.Device.DeviceType")
		if err != nil {
			fmt.Println(err)
			continue
		}
		deviceType = deviceTypeVariant.Value().(uint32)

		ip4ConfigPathVariant, err := obj.GetProperty("org.freedesktop.NetworkManager.Device.Ip4Config")
		if err != nil {
			fmt.Println(err)
			continue
		}
		ip4ConfigPath = ip4ConfigPathVariant.Value().(dbus.ObjectPath)

		addressDataVariant, err := conn.Object("org.freedesktop.NetworkManager", ip4ConfigPath).GetProperty("org.freedesktop.NetworkManager.IP4Config.AddressData")
		if err != nil {
			fmt.Println(err)
			continue
		}
		addressData = addressDataVariant.Value().([]map[string]dbus.Variant)

		if len(addressData) > 0 {
			ipAddress = addressData[0]["address"].Value().(string)
		}

		// If device type is Wi-Fi, get additional info
		if deviceType == NM_DEVICE_TYPE_WIFI {

			activeApPathVariant, err := obj.GetProperty("org.freedesktop.NetworkManager.Device.Wireless.ActiveAccessPoint")
			if err != nil {
				fmt.Println(err)
				continue
			}
			activeApPath = activeApPathVariant.Value().(dbus.ObjectPath)

			wifiSSIDVariant, err := conn.Object("org.freedesktop.NetworkManager", activeApPath).GetProperty("org.freedesktop.NetworkManager.AccessPoint.Ssid")
			if err != nil {
				fmt.Println(err)
				continue
			}
			wifiSSID = string(wifiSSIDVariant.Value().([]byte))

			wifiStrengthVariant, err := conn.Object("org.freedesktop.NetworkManager", activeApPath).GetProperty("org.freedesktop.NetworkManager.AccessPoint.Strength")
			if err != nil {
				fmt.Println(err)
				continue
			}
			wifiStrength = wifiStrengthVariant.Value().(uint8)
		}

		// Append network info
		networkInfos = append(networkInfos, NetworkInfo{
			DeviceName:   string(path),
			Interface:    interfaceName,
			IpAddress:    ipAddress,
			WifiSSID:     wifiSSID,
			DeviceType:   deviceType,
			WifiStrength: wifiStrength,
		})
	}

	return networkInfos
}

func GetNetwork(w http.ResponseWriter, _ *http.Request) {
	networkInfos := GetNetworkInfo()
	json.NewEncoder(w).Encode(networkInfos)
}

func ToggleWifiHelper(r *http.Request) bool {
	conn, err := dbus.ConnectSystemBus()
	if err != nil {
		fmt.Println(err)
	}
	defer conn.Close()

	var enabled bool
	var enable bool

	conn.Object("org.freedesktop.NetworkManager", "/org/freedesktop/NetworkManager").
		Call("org.freedesktop.DBus.Properties.Get", 0, "org.freedesktop.NetworkManager", "WirelessEnabled").
		Store(&enabled)

	if enabled {
		enable = false
	} else {
		enable = true
	}

	conn.Object("org.freedesktop.NetworkManager", "/org/freedesktop/NetworkManager").
		Call("org.freedesktop.DBus.Properties.Set", 0, "org.freedesktop.NetworkManager", "WirelessEnabled", dbus.MakeVariant(enable)).
		Store()

	return enable
}

func ToggleWifi(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "%t", ToggleWifiHelper(r))
}
