package discovery

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/grandcat/zeroconf"
)

const (
	adbTLSService = "_adb-tls-connect._tcp"
	mdnsDomain    = "local."
	scanTimeout   = 5 * time.Second
)

// DiscoveredDevice is an Android device found via mDNS on the local network.
type DiscoveredDevice struct {
	Hostname string
	IP       net.IP
	Port     int
}

func (d DiscoveredDevice) String() string {
	return fmt.Sprintf("%s (%s:%d)", d.Hostname, d.IP, d.Port)
}

// Scanner finds Android devices advertising ADB over WiFi via mDNS.
type Scanner struct {
	timeout time.Duration
}

func NewScanner() *Scanner {
	return &Scanner{timeout: scanTimeout}
}

// ScanForDevices browses the local network and returns all ADB-advertised devices found within the timeout.
func (s *Scanner) ScanForDevices(ctx context.Context) ([]DiscoveredDevice, error) {
	resolver, err := zeroconf.NewResolver(nil)
	if err != nil {
		return nil, fmt.Errorf("creating mDNS resolver: %w", err)
	}

	entries := make(chan *zeroconf.ServiceEntry)
	scanCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	if err := resolver.Browse(scanCtx, adbTLSService, mdnsDomain, entries); err != nil {
		return nil, fmt.Errorf("browsing mDNS service %s: %w", adbTLSService, err)
	}

	return collectDiscoveredDevices(scanCtx, entries), nil
}

func collectDiscoveredDevices(ctx context.Context, entries <-chan *zeroconf.ServiceEntry) []DiscoveredDevice {
	var devices []DiscoveredDevice
	for {
		select {
		case entry, ok := <-entries:
			if !ok {
				return devices
			}
			if d, found := toDiscoveredDevice(entry); found {
				devices = append(devices, d)
			}
		case <-ctx.Done():
			return devices
		}
	}
}

func toDiscoveredDevice(entry *zeroconf.ServiceEntry) (DiscoveredDevice, bool) {
	if len(entry.AddrIPv4) == 0 {
		return DiscoveredDevice{}, false
	}
	return DiscoveredDevice{
		Hostname: entry.HostName,
		IP:       entry.AddrIPv4[0],
		Port:     entry.Port,
	}, true
}
