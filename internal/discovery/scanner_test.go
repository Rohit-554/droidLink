package discovery

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/grandcat/zeroconf"
)

func TestToDiscoveredDeviceWithIPv4(t *testing.T) {
	entry := &zeroconf.ServiceEntry{
		HostName: "pixel6.local.",
		Port:     5555,
		AddrIPv4: []net.IP{net.ParseIP("192.168.1.42")},
	}

	device, found := toDiscoveredDevice(entry)

	if !found {
		t.Fatal("expected device to be found")
	}
	if device.Hostname != "pixel6.local." {
		t.Errorf("unexpected hostname: %s", device.Hostname)
	}
	if device.Port != 5555 {
		t.Errorf("unexpected port: %d", device.Port)
	}
	if device.IP.String() != "192.168.1.42" {
		t.Errorf("unexpected IP: %s", device.IP)
	}
}

func TestToDiscoveredDeviceWithNoIPv4ReturnsNotFound(t *testing.T) {
	entry := &zeroconf.ServiceEntry{
		HostName: "pixel6.local.",
		Port:     5555,
		AddrIPv4: []net.IP{},
	}

	_, found := toDiscoveredDevice(entry)

	if found {
		t.Fatal("expected device not found when no IPv4 address")
	}
}

func TestDiscoveredDeviceString(t *testing.T) {
	d := DiscoveredDevice{
		Hostname: "pixel6.local.",
		IP:       net.ParseIP("192.168.1.42"),
		Port:     5555,
	}
	s := d.String()
	if s == "" {
		t.Fatal("expected non-empty String()")
	}
}

func TestCollectDiscoveredDevicesFromChannel(t *testing.T) {
	entries := make(chan *zeroconf.ServiceEntry, 3)
	entries <- &zeroconf.ServiceEntry{
		HostName: "device1.local.", Port: 5555,
		AddrIPv4: []net.IP{net.ParseIP("10.0.0.1")},
	}
	entries <- &zeroconf.ServiceEntry{
		HostName: "device2.local.", Port: 5556,
		AddrIPv4: []net.IP{net.ParseIP("10.0.0.2")},
	}
	// entry with no IPv4 — should be skipped
	entries <- &zeroconf.ServiceEntry{
		HostName: "ipv6only.local.", Port: 5557,
		AddrIPv4: []net.IP{},
	}
	close(entries)

	ctx := context.Background()
	devices := collectDiscoveredDevices(ctx, entries)

	if len(devices) != 2 {
		t.Fatalf("expected 2 devices, got %d", len(devices))
	}
	if devices[0].Hostname != "device1.local." {
		t.Errorf("unexpected first device: %s", devices[0].Hostname)
	}
	if devices[1].Hostname != "device2.local." {
		t.Errorf("unexpected second device: %s", devices[1].Hostname)
	}
}

func TestCollectDiscoveredDevicesStopsOnContextCancel(t *testing.T) {
	entries := make(chan *zeroconf.ServiceEntry) // unbuffered, never sends
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()

	devices := collectDiscoveredDevices(ctx, entries)

	if len(devices) != 0 {
		t.Fatalf("expected 0 devices on cancelled context, got %d", len(devices))
	}
}
