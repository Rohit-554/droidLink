package installer

import (
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Rohit-554/droidLink/internal/connection"
)

type fakeInstaller struct {
	attempts  int
	failUntil int
	err       error
}

func (f *fakeInstaller) Install(serial, apkPath string) error {
	f.attempts++
	if f.attempts <= f.failUntil {
		return f.err
	}
	return nil
}

type fakePool struct {
	state connection.State
	err   error
}

func (f *fakePool) State(serial string) (connection.State, error) {
	return f.state, f.err
}

func newTestQueue(installer apkInstaller, pool devicePool) *Queue {
	return &Queue{
		adb:        installer,
		devicePool: pool,
		pending:    make(chan InstallJob, 32),
		results:    make(chan InstallResult, 32),
	}
}

func TestInstallWithRetrySucceedsOnFirstAttempt(t *testing.T) {
	q := newTestQueue(&fakeInstaller{}, &fakePool{state: connection.StateConnected})
	job := InstallJob{Serial: "10.0.0.1:5555", APKPath: "app.apk"}

	if err := q.installWithRetry(job); err != nil {
		t.Fatalf("expected success, got: %v", err)
	}
}

func TestInstallWithRetrySucceedsAfterTransientFailure(t *testing.T) {
	fake := &fakeInstaller{failUntil: 2, err: errors.New("install error")}
	q := newTestQueue(fake, &fakePool{state: connection.StateConnected})
	job := InstallJob{Serial: "10.0.0.1:5555", APKPath: "app.apk"}

	// speed up retries for test
	origBackoff := retryBackoff
	retryBackoff = 0
	defer func() { retryBackoff = origBackoff }()

	if err := q.installWithRetry(job); err != nil {
		t.Fatalf("expected success after retry, got: %v", err)
	}
	if fake.attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", fake.attempts)
	}
}

func TestInstallWithRetryFailsAfterMaxAttempts(t *testing.T) {
	fake := &fakeInstaller{failUntil: maxInstallAttempts, err: errors.New("persistent error")}
	q := newTestQueue(fake, &fakePool{state: connection.StateConnected})
	job := InstallJob{Serial: "10.0.0.1:5555", APKPath: "app.apk"}

	origBackoff := retryBackoff
	retryBackoff = 0
	defer func() { retryBackoff = origBackoff }()

	if err := q.installWithRetry(job); err == nil {
		t.Fatal("expected failure after max attempts, got nil")
	}
}

func TestWaitUntilDeviceIsReadyReturnsImmediatelyWhenConnected(t *testing.T) {
	q := newTestQueue(nil, &fakePool{state: connection.StateConnected})
	stop := make(chan struct{})

	if err := q.waitUntilDeviceIsReady("10.0.0.1:5555", stop); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestWaitUntilDeviceIsReadyReturnsErrorOnUnmanagedDevice(t *testing.T) {
	q := newTestQueue(nil, &fakePool{err: errors.New("not managed")})
	stop := make(chan struct{})

	if err := q.waitUntilDeviceIsReady("ghost:5555", stop); err == nil {
		t.Fatal("expected error for unmanaged device")
	}
}

func TestWaitUntilDeviceIsReadyRespectsStop(t *testing.T) {
	q := newTestQueue(nil, &fakePool{state: connection.StateReconnecting})
	stop := make(chan struct{})

	go func() {
		time.Sleep(20 * time.Millisecond)
		close(stop)
	}()

	if err := q.waitUntilDeviceIsReady("10.0.0.1:5555", stop); err == nil {
		t.Fatal("expected error when stop is closed")
	}
}

func TestQueueRunProcessesJobsAndPublishesResults(t *testing.T) {
	q := newTestQueue(&fakeInstaller{}, &fakePool{state: connection.StateConnected})
	stop := make(chan struct{})
	go q.Run(stop)

	q.Enqueue(InstallJob{Serial: "10.0.0.1:5555", APKPath: "app.apk"})

	select {
	case result := <-q.Results():
		if !result.Succeeded() {
			t.Fatalf("expected success, got: %v", result.Err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for result")
	}

	close(stop)
}

func TestQueueRunStopsCleanly(t *testing.T) {
	q := newTestQueue(&fakeInstaller{}, &fakePool{state: connection.StateConnected})
	stop := make(chan struct{})

	var exited atomic.Bool
	go func() {
		q.Run(stop)
		exited.Store(true)
	}()

	close(stop)
	time.Sleep(50 * time.Millisecond)

	if !exited.Load() {
		t.Fatal("Run did not exit after stop was closed")
	}
}
