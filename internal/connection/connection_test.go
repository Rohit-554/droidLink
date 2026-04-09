package connection

import (
	"testing"
	"time"
)

func TestStateString(t *testing.T) {
	cases := []struct {
		state State
		want  string
	}{
		{StateConnected, "connected"},
		{StateReconnecting, "reconnecting"},
		{StateDisconnected, "disconnected"},
	}
	for _, c := range cases {
		if got := c.state.String(); got != c.want {
			t.Errorf("State(%d).String() = %q, want %q", c.state, got, c.want)
		}
	}
}

func TestDeviceStateTransitions(t *testing.T) {
	d := &Device{
		Serial: "192.168.1.1:5555",
		Host:   "192.168.1.1",
		Port:   5555,
		state:  StateConnected,
	}

	if d.getState() != StateConnected {
		t.Fatal("expected initial state to be connected")
	}

	d.setState(StateReconnecting)
	if d.getState() != StateReconnecting {
		t.Fatal("expected state to be reconnecting")
	}

	d.setState(StateDisconnected)
	if d.getState() != StateDisconnected {
		t.Fatal("expected state to be disconnected")
	}
}

func TestManagerAddRemove(t *testing.T) {
	// Use a nil adb client — heartbeat will fail pings immediately,
	// but we just test add/remove lifecycle here.
	m := NewManager(nil)

	m.Add("10.0.0.1:5555", "10.0.0.1", 5555)

	statuses := m.Devices()
	if len(statuses) != 1 {
		t.Fatalf("expected 1 device, got %d", len(statuses))
	}
	if statuses[0].Serial != "10.0.0.1:5555" {
		t.Errorf("unexpected serial: %s", statuses[0].Serial)
	}

	// duplicate add should be a no-op
	m.Add("10.0.0.1:5555", "10.0.0.1", 5555)
	if len(m.Devices()) != 1 {
		t.Fatal("duplicate add should not increase device count")
	}

	m.Remove("10.0.0.1:5555")

	// give goroutine a moment to exit
	time.Sleep(10 * time.Millisecond)

	if len(m.Devices()) != 0 {
		t.Fatal("expected 0 devices after remove")
	}
}

func TestManagerStateUnknownDevice(t *testing.T) {
	m := NewManager(nil)
	_, err := m.State("ghost:5555")
	if err == nil {
		t.Fatal("expected error for unmanaged device")
	}
}
