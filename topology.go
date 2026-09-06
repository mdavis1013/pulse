package main

import (
	"sort"
	"strings"
)

// deviceGroup is a cluster of services believed to belong to the same
// physical device.
type deviceGroup struct {
	Name     string
	Services []DeviceRecord
}

// groupKey recovers the human-readable device name from an instance
// name, by removing the service-type suffix every instance name ends
// with (e.g. "._airplay._tcp.local").
func groupKey(d DeviceRecord) string {
	name := d.Instance
	suffix := "." + d.ServiceType
	name = strings.TrimSuffix(name, suffix)
	return name
}

// groupDevices clusters a flat device list by inferred physical device.
func groupDevices(devices []DeviceRecord) []deviceGroup {
	groups := map[string][]DeviceRecord{}
	for _, d := range devices {
		key := groupKey(d)
		groups[key] = append(groups[key], d)
	}

	out := make([]deviceGroup, 0, len(groups))
	for name, services := range groups {
		out = append(out, deviceGroup{Name: name, Services: services})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}