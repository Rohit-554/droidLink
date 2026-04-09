package adb

import (
	"testing"
)

func TestParseDevices(t *testing.T) {
	input := `List of devices attached
192.168.1.42:5555  device  product:bluejay model:Pixel_6a device:bluejay transport_id:1
emulator-5554      offline
`
	devices := parseDevices(input)

	if len(devices) != 2 {
		t.Fatalf("expected 2 devices, got %d", len(devices))
	}

	if devices[0].Serial != "192.168.1.42:5555" {
		t.Errorf("unexpected serial: %s", devices[0].Serial)
	}
	if devices[0].State != "device" {
		t.Errorf("unexpected state: %s", devices[0].State)
	}
	if devices[0].Model != "Pixel 6a" {
		t.Errorf("unexpected model: %s", devices[0].Model)
	}

	if devices[1].Serial != "emulator-5554" {
		t.Errorf("unexpected serial: %s", devices[1].Serial)
	}
	if devices[1].State != "offline" {
		t.Errorf("unexpected state: %s", devices[1].State)
	}
}

func TestParseDevicesEmpty(t *testing.T) {
	input := "List of devices attached\n"
	devices := parseDevices(input)
	if len(devices) != 0 {
		t.Fatalf("expected 0 devices, got %d", len(devices))
	}
}
