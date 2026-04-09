package daemon

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"

	"github.com/Rohit-554/droidLink/internal/adb"
	"github.com/Rohit-554/droidLink/internal/connection"
	"github.com/Rohit-554/droidLink/internal/store"
)

const (
	socketFile = "daemon.sock"

	CmdDevices = "devices"
	CmdStatus  = "status"
	CmdStop    = "stop"
)

// Request is the IPC message sent from the CLI to the daemon.
type Request struct {
	Command string `json:"command"`
	Serial  string `json:"serial,omitempty"`
	APKPath string `json:"apk_path,omitempty"`
}

// Response is the IPC message returned from the daemon to the CLI.
type Response struct {
	OK      bool        `json:"ok"`
	Error   string      `json:"error,omitempty"`
	Payload interface{} `json:"payload,omitempty"`
}

// Daemon manages device connections and handles IPC requests from the CLI.
type Daemon struct {
	adb         *adb.Client
	devicePool  *connection.Manager
	deviceStore *store.DeviceStore
	socketPath  string
	stop        chan struct{}
	listener    net.Listener
}

func New(socketDir string) (*Daemon, error) {
	adbClient, err := adb.New()
	if err != nil {
		return nil, fmt.Errorf("initialising adb: %w", err)
	}

	deviceStore, err := store.NewDeviceStore()
	if err != nil {
		return nil, fmt.Errorf("initialising device store: %w", err)
	}

	return &Daemon{
		adb:         adbClient,
		devicePool:  connection.NewManager(adbClient),
		deviceStore: deviceStore,
		socketPath:  filepath.Join(socketDir, socketFile),
		stop:        make(chan struct{}),
	}, nil
}

// Start resumes tracking all previously paired devices and begins listening for CLI commands.
func (d *Daemon) Start() error {
	d.resumePairedDevices()

	listener, err := d.openSocket()
	if err != nil {
		return err
	}
	d.listener = listener

	log.Printf("[daemon] listening on %s", d.socketPath)
	d.acceptConnections(listener)
	listener.Close()
	return nil
}

// Stop signals the daemon to shut down and unblocks the accept loop.
func (d *Daemon) Stop() {
	close(d.stop)
	if d.listener != nil {
		d.listener.Close()
	}
}

func (d *Daemon) resumePairedDevices() {
	for _, device := range d.deviceStore.All() {
		log.Printf("[daemon] resuming device %s (%s:%d)", device.Serial, device.Host, device.Port)
		if err := d.adb.Connect(device.Host, device.Port); err != nil {
			log.Printf("[daemon] could not connect to %s on startup: %v", device.Serial, err)
		}
		d.devicePool.Add(device.Serial, device.Host, device.Port)
	}
}

func (d *Daemon) openSocket() (net.Listener, error) {
	_ = os.Remove(d.socketPath)
	listener, err := net.Listen("unix", d.socketPath)
	if err != nil {
		return nil, fmt.Errorf("opening IPC socket at %s: %w", d.socketPath, err)
	}
	return listener, nil
}

func (d *Daemon) acceptConnections(listener net.Listener) {
	for {
		conn, err := listener.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return
			}
			log.Printf("[daemon] accept error: %v", err)
			continue
		}
		go d.handleConnection(conn)
	}
}

func (d *Daemon) handleConnection(conn net.Conn) {
	defer conn.Close()

	var req Request
	if err := json.NewDecoder(conn).Decode(&req); err != nil {
		writeResponse(conn, Response{OK: false, Error: "invalid request: " + err.Error()})
		return
	}

	resp := d.dispatch(req)
	writeResponse(conn, resp)
}

func (d *Daemon) dispatch(req Request) Response {
	switch req.Command {
	case CmdDevices:
		return d.listDevices()
	case CmdStatus:
		return Response{OK: true, Payload: "daemon running"}
	default:
		return Response{OK: false, Error: fmt.Sprintf("unknown command: %s", req.Command)}
	}
}

func (d *Daemon) listDevices() Response {
	return Response{OK: true, Payload: d.devicePool.Devices()}
}

func writeResponse(conn net.Conn, resp Response) {
	if err := json.NewEncoder(conn).Encode(resp); err != nil {
		log.Printf("[daemon] failed to write response: %v", err)
	}
}

// SocketPath returns the default daemon socket path under ~/.droidlink.
func SocketPath() (string, error) {
	dir, err := store.DroidlinkDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, socketFile), nil
}

// SendCommand sends a single request to a running daemon and returns the response.
func SendCommand(socketPath string, req Request) (Response, error) {
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		return Response{}, fmt.Errorf("connecting to daemon: %w", err)
	}
	defer conn.Close()

	if err := json.NewEncoder(conn).Encode(req); err != nil {
		return Response{}, fmt.Errorf("sending command: %w", err)
	}

	var resp Response
	if err := json.NewDecoder(conn).Decode(&resp); err != nil {
		return Response{}, fmt.Errorf("reading response: %w", err)
	}
	return resp, nil
}
