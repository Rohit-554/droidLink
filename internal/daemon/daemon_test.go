package daemon

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/Rohit-554/droidLink/internal/connection"
	"github.com/Rohit-554/droidLink/internal/store"
)

func newTestDaemon(t *testing.T) (*Daemon, string) {
	t.Helper()
	dir := t.TempDir()
	socketPath := filepath.Join(dir, socketFile)

	deviceStore, err := store.NewDeviceStoreAt(dir)
	if err != nil {
		t.Fatalf("creating test device store: %v", err)
	}
	pool := connection.NewManager(nil)

	return newForTesting(pool, deviceStore, socketPath), socketPath
}

func TestDispatchDevices(t *testing.T) {
	d, _ := newTestDaemon(t)

	resp := d.dispatch(Request{Command: CmdDevices})

	if !resp.OK {
		t.Fatalf("expected OK response, got error: %s", resp.Error)
	}
}

func TestDispatchStatus(t *testing.T) {
	d, _ := newTestDaemon(t)

	resp := d.dispatch(Request{Command: CmdStatus})

	if !resp.OK {
		t.Fatalf("expected OK response, got error: %s", resp.Error)
	}
	if resp.Payload != "daemon running" {
		t.Errorf("unexpected payload: %v", resp.Payload)
	}
}

func TestDispatchStop(t *testing.T) {
	d, _ := newTestDaemon(t)

	resp := d.dispatch(Request{Command: CmdStop})

	if !resp.OK {
		t.Fatalf("expected OK response, got error: %s", resp.Error)
	}
}

func TestDispatchUnknownCommandReturnsError(t *testing.T) {
	d, _ := newTestDaemon(t)

	resp := d.dispatch(Request{Command: "explode"})

	if resp.OK {
		t.Fatal("expected error response for unknown command")
	}
}

func TestRegisterDeviceValidation(t *testing.T) {
	d, _ := newTestDaemon(t)

	cases := []struct {
		name string
		req  Request
		wantOK bool
	}{
		{"missing serial", Request{Command: CmdRegisterDevice, Host: "10.0.0.1", Port: 5555}, false},
		{"missing host", Request{Command: CmdRegisterDevice, Serial: "10.0.0.1:5555", Port: 5555}, false},
		{"missing port", Request{Command: CmdRegisterDevice, Serial: "10.0.0.1:5555", Host: "10.0.0.1"}, false},
		{"valid", Request{Command: CmdRegisterDevice, Serial: "10.0.0.1:5555", Host: "10.0.0.1", Port: 5555}, true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			resp := d.registerDevice(c.req)
			if resp.OK != c.wantOK {
				t.Errorf("expected OK=%v, got OK=%v (error: %s)", c.wantOK, resp.OK, resp.Error)
			}
		})
	}
}

func TestStartStopViaIPC(t *testing.T) {
	d, socketPath := newTestDaemon(t)

	started := make(chan struct{})
	go func() {
		close(started)
		if err := d.Start(); err != nil {
			// expected after Stop() closes the listener
		}
	}()

	<-started
	time.Sleep(50 * time.Millisecond) // let Start() open socket

	resp, err := SendCommand(socketPath, Request{Command: CmdStatus})
	if err != nil {
		t.Fatalf("SendCommand failed: %v", err)
	}
	if !resp.OK {
		t.Fatalf("expected OK, got error: %s", resp.Error)
	}

	d.Stop()
	time.Sleep(50 * time.Millisecond)

	if _, err := SendCommand(socketPath, Request{Command: CmdStatus}); err == nil {
		t.Fatal("expected connection error after daemon stopped")
	}
}

func TestSocketPathUsedDroidlinkDir(t *testing.T) {
	path, err := SocketPath()
	if err != nil {
		t.Fatalf("SocketPath: %v", err)
	}
	if filepath.Base(path) != socketFile {
		t.Errorf("expected socket filename %s, got %s", socketFile, filepath.Base(path))
	}
}

func TestSendCommandFailsWhenDaemonNotRunning(t *testing.T) {
	_, err := SendCommand("/tmp/nonexistent-droidlink.sock", Request{Command: CmdStatus})
	if err == nil {
		t.Fatal("expected error when daemon is not running")
	}
}

