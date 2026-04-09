package installer

import (
	"fmt"
	"log"
	"time"

	"github.com/Rohit-554/droidLink/internal/connection"
)

type apkInstaller interface {
	Install(serial, apkPath string) error
}

type devicePool interface {
	State(serial string) (connection.State, error)
}

const (
	maxInstallAttempts    = 3
	reconnectWaitTimeout  = 30 * time.Second
	reconnectPollInterval = 500 * time.Millisecond
)

var retryBackoff = 2 * time.Second

// InstallJob represents a request to install an APK on a specific device.
type InstallJob struct {
	Serial  string
	APKPath string
}

// InstallResult reports the outcome of an install job.
type InstallResult struct {
	Job InstallJob
	Err error
}

func (r InstallResult) Succeeded() bool { return r.Err == nil }

// Queue serialises APK installs, waiting for reconnecting devices before attempting.
type Queue struct {
	adb        apkInstaller
	devicePool devicePool
	pending    chan InstallJob
	results    chan InstallResult
}

func NewQueue(client apkInstaller, pool devicePool) *Queue {
	return &Queue{
		adb:        client,
		devicePool: pool,
		pending:    make(chan InstallJob, 32),
		results:    make(chan InstallResult, 32),
	}
}

// Enqueue adds an install job to the queue. Non-blocking up to buffer capacity.
func (q *Queue) Enqueue(job InstallJob) {
	q.pending <- job
}

// Results returns the channel on which completed job outcomes are published.
func (q *Queue) Results() <-chan InstallResult {
	return q.results
}

// Run processes queued jobs until the stop channel is closed.
func (q *Queue) Run(stop <-chan struct{}) {
	for {
		select {
		case <-stop:
			return
		case job := <-q.pending:
			result := q.processJob(job, stop)
			select {
			case q.results <- result:
			case <-stop:
				return
			}
		}
	}
}

func (q *Queue) processJob(job InstallJob, stop <-chan struct{}) InstallResult {
	if err := q.waitUntilDeviceIsReady(job.Serial, stop); err != nil {
		return InstallResult{Job: job, Err: err}
	}
	return InstallResult{Job: job, Err: q.installWithRetry(job)}
}

func (q *Queue) waitUntilDeviceIsReady(serial string, stop <-chan struct{}) error {
	deadline := time.Now().Add(reconnectWaitTimeout)
	timer := time.NewTimer(reconnectPollInterval)
	defer timer.Stop()

	for time.Now().Before(deadline) {
		state, err := q.devicePool.State(serial)
		if err != nil {
			return fmt.Errorf("device %s not managed: %w", serial, err)
		}
		if state == connection.StateConnected {
			return nil
		}
		log.Printf("[installer] waiting for %s to reconnect...", serial)
		timer.Reset(reconnectPollInterval)
		select {
		case <-timer.C:
		case <-stop:
			return fmt.Errorf("queue stopped while waiting for %s", serial)
		}
	}
	return fmt.Errorf("device %s did not reconnect within %s", serial, reconnectWaitTimeout)
}

func (q *Queue) installWithRetry(job InstallJob) error {
	var lastErr error
	for attempt := 1; attempt <= maxInstallAttempts; attempt++ {
		log.Printf("[installer] installing %s on %s (attempt %d/%d)", job.APKPath, job.Serial, attempt, maxInstallAttempts)
		err := q.adb.Install(job.Serial, job.APKPath)
		if err == nil {
			return nil
		}
		lastErr = err
		log.Printf("[installer] attempt %d failed: %v", attempt, err)
		if attempt < maxInstallAttempts {
			time.Sleep(retryBackoff)
		}
	}
	return fmt.Errorf("install failed after %d attempts: %w", maxInstallAttempts, lastErr)
}
