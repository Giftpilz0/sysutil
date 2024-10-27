package handlers

import (
	"fmt"
	"net/http"

	"github.com/godbus/dbus/v5"
)

// DeviceInfo holds network information about a device, including name, IP address,
// network name, Wi-Fi SSID, signal strength, and device type.
type DeviceInfo struct {
	DeviceName     string // Name of the network device (e.g., "wlan0").
	IpAddress      string // IPv4 address of the device.
	NetworkName    string // Network name the device is connected to.
	WifiSSID       string // SSID of the Wi-Fi network.
	SignalStrength int    // Signal strength of the network connection.
	DeviceType     int    // Device type; typically Wi-Fi.
	WifiStrength   int    // Wi-Fi signal strength (if applicable).
}

// NM_DEVICE_TYPE_WIFI is a constant representing Wi-Fi device type.
const NM_DEVICE_TYPE_WIFI = 2

// GetNetworkInfo retrieves network information for all devices connected to the system.
// It connects to the system's D-Bus to query the NetworkManager service for device data.
// Returns a slice of DeviceInfo objects containing data for each detected device.
func GetNetworkInfo(r *http.Request) []DeviceInfo {
	conn, _ := dbus.ConnectSystemBus()
	defer conn.Close()
	var deviceInfos []DeviceInfo

	var devicePaths []dbus.ObjectPath
	conn.Object("org.freedesktop.NetworkManager", "/org/freedesktop/NetworkManager").
		Call("org.freedesktop.NetworkManager.GetAllDevices", 0).
		Store(&devicePaths)

	for _, devicePath := range devicePaths {
		var deviceInfo DeviceInfo

		// Get device name
		conn.Object("org.freedesktop.NetworkManager", devicePath).
			Call("org.freedesktop.DBus.Properties.Get", 0, "org.freedesktop.NetworkManager.Device", "Interface").
			Store(&deviceInfo.DeviceName)

		// Get device type
		conn.Object("org.freedesktop.NetworkManager", devicePath).
			Call("org.freedesktop.DBus.Properties.Get", 0, "org.freedesktop.NetworkManager.Device", "DeviceType").
			Store(&deviceInfo.DeviceType)

		// Get IP4 configuration path
		var ip4ConfigPath dbus.ObjectPath
		conn.Object("org.freedesktop.NetworkManager", devicePath).
			Call("org.freedesktop.DBus.Properties.Get", 0, "org.freedesktop.NetworkManager.Device", "Ip4Config").
			Store(&ip4ConfigPath)

		var addressData []map[string]dbus.Variant
		conn.Object("org.freedesktop.NetworkManager", ip4ConfigPath).
			Call("org.freedesktop.DBus.Properties.Get", 0, "org.freedesktop.NetworkManager.IP4Config", "AddressData").
			Store(&addressData)

		if len(addressData) > 0 {
			if ip, ok := addressData[0]["address"]; ok {
				deviceInfo.IpAddress = ip.Value().(string)
			}
		}

		// If device type is Wi-Fi, get additional info
		if deviceInfo.DeviceType == NM_DEVICE_TYPE_WIFI {
			var activeApPath dbus.ObjectPath
			conn.Object("org.freedesktop.NetworkManager", devicePath).
				Call("org.freedesktop.DBus.Properties.Get", 0, "org.freedesktop.NetworkManager.Device.Wireless", "ActiveAccessPoint").
				Store(&activeApPath)

			// Get the SSID, which is stored as []byte
			var ssid []byte
			conn.Object("org.freedesktop.NetworkManager", activeApPath).
				Call("org.freedesktop.DBus.Properties.Get", 0, "org.freedesktop.NetworkManager.AccessPoint", "Ssid").
				Store(&ssid)

			// Convert SSID from []byte to string
			deviceInfo.WifiSSID = string(ssid)

			conn.Object("org.freedesktop.NetworkManager", activeApPath).
				Call("org.freedesktop.DBus.Properties.Get", 0, "org.freedesktop.NetworkManager.AccessPoint", "Strength").
				Store(&deviceInfo.WifiStrength)
		}
		deviceInfos = append(deviceInfos, deviceInfo)
	}
	return deviceInfos
}

// GetSSID handles HTTP requests to get the SSID of the connected Wi-Fi network.
// Responds with the SSID string.
func GetSSID(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "%s", GetSSIDHelper(r))
}

// GetSSIDHelper iterates through network devices and returns the SSID of the first Wi-Fi device found.
func GetSSIDHelper(r *http.Request) string {
	var ssid string
	for _, device := range GetNetworkInfo(r) {
		if device.WifiSSID != "" {
			ssid = device.WifiSSID
		}
	}
	return ssid
}

// GetWifiSignalStrength handles HTTP requests to retrieve the Wi-Fi signal strength.
// Responds with the signal strength as an integer.
func GetWifiSignalStrength(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "%d", GetWifiSignalStrengthHelper(r))
}

// GetWifiSignalStrengthHelper retrieves the signal strength of the first Wi-Fi network device found.
func GetWifiSignalStrengthHelper(r *http.Request) int {
	var signalStrength int
	for _, device := range GetNetworkInfo(r) {
		if device.WifiStrength != 0 {
			signalStrength = device.WifiStrength
		}
	}
	return signalStrength
}

// GetIP handles HTTP requests to get the IP address of the device.
// Responds with the IP address as a string.
func GetIP(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "%s", GetIPHelper(r))
}

// GetIPHelper retrieves the IP address of the first network device.
func GetIPHelper(r *http.Request) string {
	var ipAddress string
	for _, device := range GetNetworkInfo(r) {
		if device.IpAddress == "" || device.IpAddress == "127.0.0.1" {
			continue
		}
		ipAddress = device.IpAddress
	}
	return ipAddress
}

// ToggleWifi handles HTTP requests to toggle the Wi-Fi status (enabled/disabled).
// Responds with a boolean indicating the new Wi-Fi status.
func ToggleWifi(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "%t", ToggleWifiHelper(r))
}

// ToggleWifiHelper toggles the Wi-Fi status (enabled/disabled) and returns the new status as a boolean.
func ToggleWifiHelper(r *http.Request) bool {
	conn, _ := dbus.ConnectSystemBus()
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
