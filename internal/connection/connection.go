package connection

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/Rohit-554/droidLink/internal/adb"
)

const (
	heartbeatInterval     = 4 * time.Second
	maxMisses             = 3
	reconnectTimeout      = 30 * time.Second
	reconnectRetryInterval = 2 * time.Second
)

// State represents the connection state of a device.
type State int

const (
	StateConnected    State = iota
	StateReconnecting State = iota
	StateDisconnected State = iota
)

func (s State) String() string {
	switch s {
	case StateConnected:
		return "connected"
	case StateReconnecting:
		return "reconnecting"
	case StateDisconnected:
		return "disconnected"
	default:
		return "unknown"
	}
}

// Device holds runtime connection state for a paired device.
type Device struct {
	Serial string
	Host   string
	Port   int

	mu     sync.RWMutex
	state  State
	misses int
}

func (d *Device) getState() State {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.state
}

func (d *Device) setState(s State) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.state = s
}

func (d *Device) resetMisses() {
	d.mu.Lock()
	d.misses = 0
	d.mu.Unlock()
}

// Manager manages heartbeat loops for all connected devices.
type Manager struct {
	adb     *adb.Client
	mu      sync.RWMutex
	devices map[string]*Device
	stopChs map[string]chan struct{}
}

func NewManager(client *adb.Client) *Manager {
	return &Manager{
		adb:     client,
		devices: make(map[string]*Device),
		stopChs: make(map[string]chan struct{}),
	}
}

// Add registers a device and starts its heartbeat loop.
func (m *Manager) Add(serial, host string, port int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.devices[serial]; exists {
		return
	}

	d := &Device{
		Serial: serial,
		Host:   host,
		Port:   port,
		state:  StateConnected,
	}
	stop := make(chan struct{})
	m.devices[serial] = d
	m.stopChs[serial] = stop

	go m.watchDeviceHealth(d, stop)
}

// Remove stops the heartbeat and removes the device.
func (m *Manager) Remove(serial string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if stop, ok := m.stopChs[serial]; ok {
		close(stop)
		delete(m.stopChs, serial)
	}
	delete(m.devices, serial)
}

// Stop stops all heartbeat loops and clears the device pool.
func (m *Manager) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for serial, stop := range m.stopChs {
		close(stop)
		delete(m.stopChs, serial)
	}
	m.devices = make(map[string]*Device)
}

// State returns the current connection state of a device.
func (m *Manager) State(serial string) (State, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	d, ok := m.devices[serial]
	if !ok {
		return StateDisconnected, fmt.Errorf("device %s not managed", serial)
	}
	return d.getState(), nil
}

// Devices returns a snapshot of all managed devices and their states.
func (m *Manager) Devices() []DeviceStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()

	out := make([]DeviceStatus, 0, len(m.devices))
	for _, d := range m.devices {
		out = append(out, DeviceStatus{
			Serial: d.Serial,
			Host:   d.Host,
			Port:   d.Port,
			State:  d.getState(),
		})
	}
	return out
}

// DeviceStatus is a read-only snapshot of a device's state.
type DeviceStatus struct {
	Serial string
	Host   string
	Port   int
	State  State
}

func (m *Manager) watchDeviceHealth(d *Device, stop <-chan struct{}) {
	ticker := time.NewTicker(heartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			m.handleHeartbeatTick(d, stop)
		}
	}
}

func (m *Manager) handleHeartbeatTick(d *Device, stop <-chan struct{}) {
	if err := m.adb.Ping(d.Serial); err != nil {
		m.recordMissedPing(d, stop)
	} else {
		m.recordSuccessfulPing(d)
	}
}

func (m *Manager) recordMissedPing(d *Device, stop <-chan struct{}) {
	d.mu.Lock()
	d.misses++
	misses := d.misses
	shouldReconnect := misses >= maxMisses && d.state == StateConnected
	if shouldReconnect {
		d.state = StateReconnecting
	}
	d.mu.Unlock()

	log.Printf("[heartbeat] %s missed ping %d/%d", d.Serial, misses, maxMisses)

	if shouldReconnect {
		go m.restoreConnection(d, stop)
	}
}

func (m *Manager) recordSuccessfulPing(d *Device) {
	d.mu.Lock()
	d.misses = 0
	d.state = StateConnected
	d.mu.Unlock()
}

func (m *Manager) restoreConnection(d *Device, stop <-chan struct{}) {
	log.Printf("[reconnect] attempting to reconnect %s (%s:%d)", d.Serial, d.Host, d.Port)

	deadline := time.Now().Add(reconnectTimeout)
	for time.Now().Before(deadline) {
		select {
		case <-stop:
			return
		default:
		}

		if err := m.adb.Connect(d.Host, d.Port); err != nil {
			log.Printf("[reconnect] connect error for %s: %v", d.Serial, err)
			time.Sleep(reconnectRetryInterval)
			continue
		}

		if err := m.adb.WaitForDevice(d.Serial, 5*time.Second); err != nil {
			log.Printf("[reconnect] wait error for %s: %v", d.Serial, err)
			time.Sleep(reconnectRetryInterval)
			continue
		}

		d.resetMisses()
		d.setState(StateConnected)
		log.Printf("[reconnect] %s reconnected successfully", d.Serial)
		return
	}

	log.Printf("[reconnect] gave up on %s after %s", d.Serial, reconnectTimeout)
	d.setState(StateDisconnected)
}
