package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

const devicesFile = "devices.json"

var ErrDeviceNotFound = errors.New("device not found")

// PairedDevice holds the persistent identity of a paired Android device.
type PairedDevice struct {
	Serial string `json:"serial"`
	Host   string `json:"host"`
	Port   int    `json:"port"`
	Model  string `json:"model"`
}

// DeviceStore persists paired devices to ~/.droidlink/devices.json.
type DeviceStore struct {
	mu      sync.RWMutex
	dataDir string
	devices map[string]PairedDevice
}

func NewDeviceStore() (*DeviceStore, error) {
	dir, err := DroidlinkDir()
	if err != nil {
		return nil, err
	}
	s := &DeviceStore{
		dataDir: dir,
		devices: make(map[string]PairedDevice),
	}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *DeviceStore) Save(device PairedDevice) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.devices[device.Serial] = device
	return s.flush()
}

func (s *DeviceStore) Remove(serial string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.devices[serial]; !exists {
		return fmt.Errorf("%w: %s", ErrDeviceNotFound, serial)
	}
	delete(s.devices, serial)
	return s.flush()
}

func (s *DeviceStore) All() []PairedDevice {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]PairedDevice, 0, len(s.devices))
	for _, d := range s.devices {
		out = append(out, d)
	}
	return out
}

func (s *DeviceStore) Find(serial string) (PairedDevice, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	d, ok := s.devices[serial]
	if !ok {
		return PairedDevice{}, fmt.Errorf("%w: %s", ErrDeviceNotFound, serial)
	}
	return d, nil
}

func (s *DeviceStore) load() error {
	path := filepath.Join(s.dataDir, devicesFile)
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("reading device store: %w", err)
	}

	var list []PairedDevice
	if err := json.Unmarshal(data, &list); err != nil {
		return fmt.Errorf("parsing device store: %w", err)
	}
	for _, d := range list {
		s.devices[d.Serial] = d
	}
	return nil
}

func (s *DeviceStore) flush() error {
	list := make([]PairedDevice, 0, len(s.devices))
	for _, d := range s.devices {
		list = append(list, d)
	}

	data, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding device store: %w", err)
	}

	path := filepath.Join(s.dataDir, devicesFile)
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("writing device store: %w", err)
	}
	return nil
}

// DroidlinkDir returns ~/.droidlink, creating it if it does not exist.
func DroidlinkDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolving home directory: %w", err)
	}
	dir := filepath.Join(home, ".droidlink")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", fmt.Errorf("creating droidlink directory: %w", err)
	}
	return dir, nil
}
