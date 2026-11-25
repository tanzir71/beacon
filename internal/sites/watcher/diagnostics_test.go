package watcher

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestDiagnosticsPreserveReconcileErrorsAndRecovery(t *testing.T) {
	var fail atomic.Bool
	fail.Store(true)
	watcher, err := New(Options{ReconcileInterval: time.Hour, Reconcile: func() error {
		if fail.Load() {
			return errors.New("injected reconciliation failure")
		}
		return nil
	}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := watcher.Close(); err != nil {
			t.Error(err)
		}
	})
	watcher.recordDebounce("fixture write", nil)
	watcher.recordDebounce("", errors.New("injected native error"))
	watcher.reconcile()
	state := watcher.Diagnostics()
	if state.Events != 1 || state.NativeErrors != 1 || state.DebounceResets != 2 || !state.DebouncePending ||
		state.LastNativeError != "injected native error" || state.LastEvent != "fixture write" ||
		state.ReconcilesStarted != 1 || state.ReconcilesCompleted != 1 || state.ReconcileRunning ||
		state.LastReconcileError != "injected reconciliation failure" {
		t.Fatalf("failure diagnostics = %+v", state)
	}
	fail.Store(false)
	watcher.reconcile()
	state = watcher.Diagnostics()
	if state.ReconcilesCompleted != 2 || state.LastReconcileError != "" {
		t.Fatalf("successful reconcile did not clear its prior error: %+v", state)
	}
	state.Events = 999
	if watcher.Diagnostics().Events != 1 {
		t.Fatal("caller could mutate diagnostics")
	}
}

func TestDiagnosticsCanObserveAnInflightReconcile(t *testing.T) {
	entered, release, done := make(chan struct{}), make(chan struct{}), make(chan struct{})
	unblock := sync.OnceFunc(func() { close(release) })
	watcher, err := New(Options{ReconcileInterval: time.Hour, Reconcile: func() error {
		close(entered)
		<-release
		return nil
	}})
	if err != nil {
		t.Fatal(err)
	}
	go func() { watcher.reconcile(); close(done) }()
	t.Cleanup(func() {
		unblock()
		<-done
		if err := watcher.Close(); err != nil {
			t.Error(err)
		}
	})
	select {
	case <-entered:
	case <-time.After(5 * time.Second):
		t.Fatal("reconcile did not enter")
	}
	state := watcher.Diagnostics()
	if !state.ReconcileRunning || state.ReconcilesStarted != 1 || state.ReconcilesCompleted != 0 || state.ReconcileStartedAt.IsZero() {
		t.Fatalf("in-flight diagnostics = %+v", state)
	}
	unblock()
	<-done
	if err := watcher.Close(); err != nil {
		t.Fatal(err)
	}
	if state := watcher.Diagnostics(); state.ReconcileRunning || state.ReconcilesCompleted != 1 || !state.Stopped {
		t.Fatalf("completion diagnostics = %+v", state)
	}
}
