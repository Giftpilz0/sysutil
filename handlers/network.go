package handlers

import (
	"fmt"

	"github.com/godbus/dbus/v5"
)

const (
	nmService         = "org.freedesktop.NetworkManager"
	nmPath            = "/org/freedesktop/NetworkManager"
	deviceInterface   = "org.freedesktop.NetworkManager.Device"
	wirelessInterface = "org.freedesktop.NetworkManager.Device.Wireless"
	apInterface       = "org.freedesktop.NetworkManager.AccessPoint"

	// NetworkManager uses 2 for Wi-Fi devices.
	deviceTypeWifi = 2
)

// NetworkDevice represents a network device with detailed properties.
type NetworkDevice struct {
	DeviceName   string `json:"deviceName"`
	Interface    string `json:"interface"`
	IpAddress    string `json:"ipAddress,omitempty"`
	WifiSSID     string `json:"wifiSSID,omitempty"`
	DeviceType   uint32 `json:"deviceType"`
	WifiStrength uint8  `json:"wifiStrength,omitempty"`
}

// GetNetworkDevices connects to DBus and retrieves detailed network information using NetworkManager.
func GetNetworkDevices() ([]NetworkDevice, error) {
	conn, err := dbus.SystemBus()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to system DBus: %w", err)
	}

	nm := conn.Object(nmService, dbus.ObjectPath(nmPath))

	// Get the list of network devices.
	var devicePaths []dbus.ObjectPath
	if err := nm.Call("org.freedesktop.NetworkManager.GetDevices", 0).Store(&devicePaths); err != nil {
		return nil, fmt.Errorf("failed to get network devices: %w", err)
	}

	var devices []NetworkDevice
	for _, path := range devicePaths {
		devObj := conn.Object(nmService, path)
		deviceName := string(path)

		// Retrieve the interface name.
		ifaceStr := ""
		if ifaceVar, err := GetProperty(devObj, deviceInterface, "Interface"); err == nil {
			if str, ok := ifaceVar.Value().(string); ok {
				ifaceStr = str
			}
		}

		// Retrieve the IPv4 address from AddressData instead of Ip4Address.
		ipStr := ""
		if ip4ConfigVar, err := GetProperty(devObj, "org.freedesktop.NetworkManager.Device", "Ip4Config"); err == nil {
			if ip4ConfigPath, ok := ip4ConfigVar.Value().(dbus.ObjectPath); ok && ip4ConfigPath != "/" && string(ip4ConfigPath) != "" {
				ip4ConfigObj := conn.Object("org.freedesktop.NetworkManager", ip4ConfigPath)
				// Get the AddressData property from the IP4Config object.
				addrDataVar, err := GetProperty(ip4ConfigObj, "org.freedesktop.NetworkManager.IP4Config", "AddressData")
				if err == nil {
					// The AddressData property is expected to be a slice of dictionaries.
					if addrData, ok := addrDataVar.Value().([]map[string]dbus.Variant); ok && len(addrData) > 0 {
						if addrVar, exists := addrData[0]["address"]; exists {
							if addr, ok := addrVar.Value().(string); ok {
								ipStr = addr
							}
						}
					}
				}
			}
		}

		// Retrieve the device type.
		var deviceType uint32
		if typeVar, err := GetProperty(devObj, deviceInterface, "DeviceType"); err == nil {
			if dt, ok := typeVar.Value().(uint32); ok {
				deviceType = dt
			}
		}

		// If this is a Wi-Fi device, retrieve additional wireless properties.
		wifiSSID := ""
		wifiStrength := uint8(0)
		if deviceType == deviceTypeWifi {
			if apVar, err := GetProperty(devObj, wirelessInterface, "ActiveAccessPoint"); err == nil {
				if apPath, ok := apVar.Value().(dbus.ObjectPath); ok && apPath != "/" && string(apPath) != "" {
					apObj := conn.Object(nmService, apPath)
					// Retrieve the Wi-Fi SSID.
					if ssidVar, err := GetProperty(apObj, apInterface, "Ssid"); err == nil {
						if ssidBytes, ok := ssidVar.Value().([]byte); ok {
							wifiSSID = string(ssidBytes)
						}
					}
					// Retrieve the Wi-Fi signal strength.
					if strengthVar, err := GetProperty(apObj, apInterface, "Strength"); err == nil {
						switch v := strengthVar.Value().(type) {
						case uint8:
							wifiStrength = v
						case uint32:
							wifiStrength = uint8(v)
						}
					}
				}
			}
		}

		devices = append(devices, NetworkDevice{
			DeviceName:   deviceName,
			Interface:    ifaceStr,
			IpAddress:    ipStr,
			WifiSSID:     wifiSSID,
			DeviceType:   deviceType,
			WifiStrength: wifiStrength,
		})
	}

	return devices, nil
}
