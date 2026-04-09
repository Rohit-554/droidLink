package adb

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// Client wraps the adb binary.
type Client struct {
	adbPath string
}

func New() (*Client, error) {
	path, err := exec.LookPath("adb")
	if err != nil {
		return nil, fmt.Errorf("adb not found in PATH: %w", err)
	}
	return &Client{adbPath: path}, nil
}

func NewWithPath(path string) *Client {
	return &Client{adbPath: path}
}

func (c *Client) Connect(host string, port int) error {
	out, err := c.run("connect", deviceAddr(host, port))
	if err != nil {
		return err
	}
	// adb exits 0 even on connection failure — check the output text
	if strings.Contains(out, "failed") || strings.Contains(out, "unable") {
		return fmt.Errorf("connect failed: %s", out)
	}
	return nil
}

func (c *Client) Disconnect(host string, port int) error {
	_, err := c.run("disconnect", deviceAddr(host, port))
	return err
}

func (c *Client) Ping(serial string) error {
	_, err := c.run("-s", serial, "shell", "echo", "ping")
	return err
}

func (c *Client) Devices() ([]Device, error) {
	out, err := c.run("devices", "-l")
	if err != nil {
		return nil, err
	}
	return parseDevices(out), nil
}

func (c *Client) Install(serial, apkPath string) error {
	_, err := c.run("-s", serial, "install", "-r", apkPath)
	return err
}

func (c *Client) WaitForDevice(serial string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if err := c.Ping(serial); err == nil {
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("device %s not available after %s", serial, timeout)
}

// Device represents a connected ADB device.
type Device struct {
	Serial string
	State  string // "device", "offline", "unauthorized"
	Model  string
}

func (c *Client) run(args ...string) (string, error) {
	cmd := exec.Command(c.adbPath, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = strings.TrimSpace(stdout.String())
		}
		return "", fmt.Errorf("adb %s: %s", strings.Join(args, " "), msg)
	}
	return strings.TrimSpace(stdout.String()), nil
}

func deviceAddr(host string, port int) string {
	return fmt.Sprintf("%s:%d", host, port)
}

func parseDevices(output string) []Device {
	var devices []Device
	lines := strings.Split(output, "\n")
	// adb devices -l always emits a header line before any device entries
	for _, line := range lines[1:] {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		d := Device{
			Serial: fields[0],
			State:  fields[1],
		}
		// key:value pairs follow state, e.g. "model:Pixel_6"
		for _, f := range fields[2:] {
			if strings.HasPrefix(f, "model:") {
				d.Model = strings.ReplaceAll(strings.TrimPrefix(f, "model:"), "_", " ")
				break
			}
		}
		devices = append(devices, d)
	}
	return devices
}
