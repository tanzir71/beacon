package server

import (
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/tanzir71/beacon/internal/sites/app"
	"github.com/tanzir71/beacon/internal/sites/scanner"
	"github.com/tanzir71/beacon/internal/sites/supervisor"
	"github.com/tanzir71/beacon/internal/sites/watcher"
)

func TestConstructorFailureJoinsAcquiredManager(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "index.php"), []byte("<?php echo 'fixture';"), 0o600); err != nil {
		t.Fatal(err)
	}
	want := errors.New("injected PHP handler failure")
	var manager *supervisor.Manager
	server, err := newWithManager(Options{
		Scanner: scanner.Options{Registered: []string{root}},
		PHPHandler: func(app.App) (http.Handler, error) {
			return nil, want
		},
	}, func(options supervisor.Options) *supervisor.Manager {
		manager = supervisor.NewManager(options)
		return manager
	})
	if manager == nil {
		t.Fatal("constructor failed before acquiring its manager")
	}
	t.Cleanup(func() {
		if err := manager.Close(); err != nil {
			t.Error(err)
		}
	})
	if server != nil || !errors.Is(err, want) {
		t.Fatalf("constructor = %v, %v; want nil and original failure", server, err)
	}
	select {
	case failure, open := <-manager.NotificationErrors():
		if open || failure != nil {
			t.Fatalf("unexpected notification failure after rollback: %v, open=%v", failure, open)
		}
	default:
		t.Fatal("constructor returned before joining its acquired manager")
	}
}

func TestServerCloseCancelsStartupBeforeJoiningWatcher(t *testing.T) {
	manager := supervisor.NewManager(supervisor.Options{})
	entered := make(chan struct{})
	var once sync.Once
	application := app.App{Path: t.TempDir(), Slug: "lazy-fixture"}
	liveWatcher, err := watcher.New(watcher.Options{
		ReconcileInterval: time.Millisecond,
		Reconcile: func() error {
			once.Do(func() { close(entered) })
			ticker := time.NewTicker(time.Millisecond)
			defer ticker.Stop()
			for {
				if _, err := manager.Handler(application); err != nil {
					return nil
				}
				<-ticker.C
			}
		},
	})
	if err != nil {
		if closeErr := manager.Close(); closeErr != nil {
			t.Error(closeErr)
		}
		t.Fatal(err)
	}
	server := &Server{supervisor: manager, watcher: liveWatcher}
	closed := make(chan struct{})
	t.Cleanup(func() {
		manager.Cancel()
		if err := server.Close(); err != nil {
			t.Error(err)
		}
		select {
		case <-closed:
		case <-time.After(5 * time.Second):
			t.Error("server shutdown did not finish")
		}
	})
	select {
	case <-entered:
	case <-time.After(5 * time.Second):
		t.Fatal("watcher did not enter its reconciliation")
	}
	go func() {
		defer close(closed)
		if err := server.Close(); err != nil {
			t.Errorf("close server: %v", err)
		}
	}()
	select {
	case <-closed:
	case <-time.After(5 * time.Second):
		t.Fatal("server waited for reconciliation before sending startup cancellation")
	}
}
