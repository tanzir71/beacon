package supervisor

import (
	"errors"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/tanzir71/beacon/internal/sites/app"
)

func TestChangeNotificationsDoNotBlockOrAccumulate(t *testing.T) {
	entered, second, release := make(chan struct{}), make(chan struct{}), make(chan struct{})
	unblock := sync.OnceFunc(func() { close(release) })
	var calls atomic.Int64
	manager := NewManager(Options{OnChange: func() {
		switch calls.Add(1) {
		case 1:
			close(entered)
			<-release
		case 2:
			close(second)
		}
	}})
	returned := make(chan struct{})
	go func() { manager.notify(); close(returned) }()
	t.Cleanup(func() {
		unblock()
		if err := manager.Close(); err != nil {
			t.Error(err)
		}
		waitNotification(t, returned, "notification caller cleanup")
	})
	waitNotification(t, entered, "first callback")
	waitNotification(t, returned, "notification return while callback is blocked")
	if !manager.Diagnostics("").NotificationRunning {
		t.Fatal("diagnostics did not identify the blocked callback")
	}
	for range 1000 {
		manager.notify()
	}
	if state := manager.Diagnostics(""); state.NotificationPending != 1 {
		t.Fatalf("coalesced notification backlog = %+v", state)
	}
	unblock()
	waitNotification(t, second, "coalesced latest change")
	if err := manager.Close(); err != nil {
		t.Fatal(err)
	}
	manager.notify()
	if calls.Load() != 2 {
		t.Fatalf("callback calls = %d, want two coalesced notifications", calls.Load())
	}
}

func TestCloseJoinsNotificationsForEveryCaller(t *testing.T) {
	entered, release := make(chan struct{}), make(chan struct{})
	unblock := sync.OnceFunc(func() { close(release) })
	manager := NewManager(Options{OnChange: func() { close(entered); <-release }})
	notified := make(chan struct{})
	go func() { manager.notify(); close(notified) }()
	closed := []chan struct{}{make(chan struct{}), make(chan struct{})}
	t.Cleanup(func() {
		unblock()
		if err := manager.Close(); err != nil {
			t.Error(err)
		}
		waitNotification(t, notified, "notification caller cleanup")
	})
	waitNotification(t, entered, "callback entry")
	for _, done := range closed {
		go func() {
			defer close(done)
			if err := manager.Close(); err != nil {
				t.Errorf("close manager: %v", err)
			}
		}()
	}
	waitNotification(t, manager.ctx.Done(), "manager cancellation")
	for _, done := range closed {
		select {
		case <-done:
			t.Fatal("Close returned before callback cleanup")
		case <-time.After(20 * time.Millisecond):
		}
	}
	unblock()
	for _, done := range closed {
		waitNotification(t, done, "joined manager closure")
	}
}

func TestNotificationPanicReachesItsServiceOwner(t *testing.T) {
	var calls atomic.Int64
	recovered := make(chan struct{})
	manager := NewManager(Options{OnChange: func() {
		if calls.Add(1) == 1 {
			panic("injected callback failure")
		}
		close(recovered)
	}})
	t.Cleanup(func() {
		if err := manager.Close(); err == nil || !strings.Contains(err.Error(), "injected callback failure") {
			t.Errorf("Close lost the worker failure: %v", err)
		}
	})
	manager.notify()
	select {
	case err := <-manager.NotificationErrors():
		if err == nil || !strings.Contains(err.Error(), "injected callback failure") {
			t.Fatalf("callback failure was not preserved: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("callback panic did not reach its service owner")
	}
	manager.notify()
	waitNotification(t, recovered, "notification after callback panic")
	if err := manager.Close(); err == nil || !strings.Contains(err.Error(), "injected callback failure") {
		t.Fatalf("Close lost the worker failure: %v", err)
	}
	if _, open := <-manager.NotificationErrors(); open {
		t.Fatal("notification error stream stayed open after joined shutdown")
	}
}

func TestManagerCloseCancelsAndJoinsRestartBackoff(t *testing.T) {
	starting := make(chan struct{})
	manager := NewManager(Options{
		RestartDelays: []time.Duration{time.Hour},
		OnChange:      func() { close(starting) },
	})
	t.Cleanup(func() {
		if err := manager.Close(); err != nil {
			t.Error(err)
		}
	})
	process := newProcess(app.App{Path: t.TempDir(), Slug: "backoff-fixture"})
	process.done = make(chan error, 1)
	process.done <- errors.New("injected process exit")
	process.mount = newManagedHandler(http.NotFoundHandler())
	manager.processes[filepath.Clean(process.application.Path)] = process
	monitored := make(chan struct{})
	manager.monitors.Go(func() {
		defer close(monitored)
		manager.monitor(process)
	})
	waitNotification(t, starting, "restart backoff")
	if err := manager.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-monitored:
	default:
		t.Fatal("Close returned without joining the backoff monitor")
	}
}

func TestRestartMonitorAdvancesWhileChangeCallbackIsBlocked(t *testing.T) {
	entered, release := make(chan struct{}), make(chan struct{})
	notified := make(chan struct{})
	unblock := sync.OnceFunc(func() { close(release) })
	var first sync.Once
	manager := NewManager(Options{
		RestartDelays: []time.Duration{time.Millisecond},
		OnChange:      func() { first.Do(func() { close(entered); <-release }) },
	})
	t.Cleanup(func() {
		unblock()
		if err := manager.Close(); err != nil {
			t.Error(err)
		}
		waitNotification(t, notified, "initial change caller cleanup")
	})
	go func() { manager.notify(); close(notified) }()
	waitNotification(t, entered, "blocked change callback")
	// Force a deterministic replacement failure before any port or child process
	// is acquired, so this unit proof has no external fixture dependencies.
	manager.portErr = errors.New("injected replacement failure")
	process := newProcess(app.App{Path: t.TempDir(), Slug: "restart-fixture"})
	process.done = make(chan error, 1)
	process.done <- errors.New("injected process exit")
	process.mount = newManagedHandler(http.NotFoundHandler())
	key := filepath.Clean(process.application.Path)
	manager.processes[key] = process
	monitored := make(chan struct{})
	manager.monitors.Go(func() {
		defer close(monitored)
		manager.monitor(process)
	})
	waitNotification(t, monitored, "replacement attempt despite blocked notification")
	manager.mu.Lock()
	status := manager.processes[key].status
	manager.mu.Unlock()
	if status != "crashed" {
		t.Fatalf("replacement attempt did not finish: %s", status)
	}
}

func TestCloseCancelsBeforeWaitingForStartupLock(t *testing.T) {
	manager := NewManager(Options{})
	manager.mu.Lock()
	unlock := sync.OnceFunc(manager.mu.Unlock)
	closed := make(chan struct{})
	go func() {
		defer close(closed)
		if err := manager.Close(); err != nil {
			t.Errorf("close manager: %v", err)
		}
	}()
	t.Cleanup(func() {
		unlock()
		waitNotification(t, closed, "manager shutdown cleanup")
	})
	waitNotification(t, manager.ctx.Done(), "startup cancellation before manager lock")
	unlock()
	waitNotification(t, closed, "manager shutdown")
}

func TestCommandDiagnosticsDoNotWaitForTheStartupLock(t *testing.T) {
	manager := NewManager(Options{})
	t.Cleanup(func() {
		if err := manager.Close(); err != nil {
			t.Error(err)
		}
	})
	process := newProcess(app.App{Slug: "fixture"})
	process.port, process.attempts = 12345, 2
	finished := manager.beginOperation(process, "test start")
	defer finished()
	manager.mu.Lock()
	defer manager.mu.Unlock()
	states := make(chan Diagnostics, 1)
	go func() { states <- manager.Diagnostics("fixture") }()
	select {
	case state := <-states:
		if !state.ManagerBusy || state.Found || state.Operation != "test start" || state.OperationApp != "fixture" ||
			state.OperationAttempt != 2 || state.OperationPort != 12345 {
			t.Fatalf("busy manager diagnostics = %+v", state)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("diagnostics waited for a startup-held lock")
	}
}

func waitNotification(t *testing.T, signal <-chan struct{}, description string) {
	t.Helper()
	select {
	case <-signal:
	case <-time.After(5 * time.Second):
		t.Fatalf("timed out waiting for %s", description)
	}
}
