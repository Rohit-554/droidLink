package store

import (
	"os"
	"testing"
)

func newTempStore(t *testing.T) *DeviceStore {
	t.Helper()
	dir := t.TempDir()
	s := &DeviceStore{
		dataDir: dir,
		devices: make(map[string]PairedDevice),
	}
	return s
}

func TestSaveAndFind(t *testing.T) {
	s := newTempStore(t)

	device := PairedDevice{Serial: "10.0.0.1:5555", Host: "10.0.0.1", Port: 5555, Model: "Pixel 6"}
	if err := s.Save(device); err != nil {
		t.Fatalf("Save: %v", err)
	}

	found, err := s.Find(device.Serial)
	if err != nil {
		t.Fatalf("Find: %v", err)
	}
	if found.Model != "Pixel 6" {
		t.Errorf("unexpected model: %s", found.Model)
	}
}

func TestRemove(t *testing.T) {
	s := newTempStore(t)

	device := PairedDevice{Serial: "10.0.0.2:5555", Host: "10.0.0.2", Port: 5555}
	_ = s.Save(device)

	if err := s.Remove(device.Serial); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if _, err := s.Find(device.Serial); err == nil {
		t.Fatal("expected error after remove, got nil")
	}
}

func TestRemoveUnknownDeviceReturnsError(t *testing.T) {
	s := newTempStore(t)
	if err := s.Remove("ghost:5555"); err == nil {
		t.Fatal("expected error removing unknown device")
	}
}

func TestPersistenceAcrossReload(t *testing.T) {
	dir := t.TempDir()

	first := &DeviceStore{dataDir: dir, devices: make(map[string]PairedDevice)}
	_ = first.Save(PairedDevice{Serial: "10.0.0.3:5555", Host: "10.0.0.3", Port: 5555, Model: "Pixel 7"})

	second := &DeviceStore{dataDir: dir, devices: make(map[string]PairedDevice)}
	if err := second.load(); err != nil {
		t.Fatalf("load: %v", err)
	}

	found, err := second.Find("10.0.0.3:5555")
	if err != nil {
		t.Fatalf("Find after reload: %v", err)
	}
	if found.Model != "Pixel 7" {
		t.Errorf("unexpected model after reload: %s", found.Model)
	}
}

func TestLoadMissingFileIsNotAnError(t *testing.T) {
	dir := t.TempDir()
	_ = os.Remove(dir + "/devices.json")

	s := &DeviceStore{dataDir: dir, devices: make(map[string]PairedDevice)}
	if err := s.load(); err != nil {
		t.Fatalf("expected no error for missing file, got: %v", err)
	}
}
