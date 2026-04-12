package handlers

import "github.com/godbus/dbus/v5"

// getProperty is a helper function to retrieve a DBus property for a given interface and property name.
func GetProperty(obj dbus.BusObject, iface, prop string) (dbus.Variant, error) {
	call := obj.Call("org.freedesktop.DBus.Properties.Get", 0, iface, prop)
	if call.Err != nil {
		return dbus.Variant{}, call.Err
	}
	var v dbus.Variant
	if err := call.Store(&v); err != nil {
		return dbus.Variant{}, err
	}
	return v, nil
}
