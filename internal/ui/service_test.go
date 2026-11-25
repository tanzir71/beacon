package ui

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/tanzir71/beacon/internal/sites/indexer"
	"github.com/tanzir71/beacon/internal/ui/dashboard"
)

func TestUIServiceCoalescesWithoutWaitingForRendering(t *testing.T) {
	entered, release := make(chan struct{}), make(chan struct{})
	var first sync.Once
	unblock := sync.OnceFunc(func() { close(release) })
	service := NewService(ServiceOptions{Dashboard: func() dashboard.Options {
		first.Do(func() { close(entered); <-release })
		return dashboard.Options{}
	}})
	service.Publish(indexer.Snapshot{Entries: []indexer.Entry{{Slug: "notes", Name: "Before"}}})
	startUIService(t, service)
	t.Cleanup(unblock)
	select {
	case <-entered:
	case <-time.After(5 * time.Second):
		t.Fatal("UI did not start rendering")
	}
	published := make(chan struct{})
	go func() {
		defer close(published)
		data := indexer.Snapshot{Entries: []indexer.Entry{{Slug: "notes", Tags: []string{"original"}}}}
		for number := range 1000 {
			data.Entries[0].Name = fmt.Sprintf("site-%d", number)
			service.Publish(data)
		}
		data.Entries[0].Tags[0] = "mutated"
	}()
	select {
	case <-published:
	case <-time.After(5 * time.Second):
		t.Fatal("site publication waited for the blocked UI renderer")
	}
	if len(service.changed) != 1 {
		t.Fatalf("notification backlog = %d, want one coalesced notification", len(service.changed))
	}
	unblock()
	waitForUI(t, func() bool {
		if !service.Ready() {
			return false
		}
		entries := siteViewEntries(t, service)
		return len(entries) == 1 && entries[0].Name == "site-999"
	})
	if entries := siteViewEntries(t, service); entries[0].Tags[0] != "original" {
		t.Fatal("publisher could mutate the UI's pending snapshot")
	}
}

func TestUIServiceRecoversOnANewRunAfterPreferenceFailure(t *testing.T) {
	directory := t.TempDir()
	preferencesPath := filepath.Join(directory, "dashboard.json")
	if err := os.WriteFile(preferencesPath, []byte("invalid JSON"), 0o600); err != nil {
		t.Fatal(err)
	}
	service := NewService(ServiceOptions{IndexPath: filepath.Join(directory, "index.json")})
	service.Publish(indexer.Snapshot{Entries: []indexer.Entry{{Slug: "before", Name: "Before"}}})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := service.Run(ctx); err == nil {
		t.Fatal("corrupt UI preferences did not fail the UI service")
	}
	assertUIUnavailable(t, service)
	if err := os.WriteFile(preferencesPath, []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	service.Publish(indexer.Snapshot{Entries: []indexer.Entry{{Slug: "after", Name: "After"}}})
	stop, done := startUIService(t, service)
	waitForUI(t, service.Ready)
	if entries := siteViewEntries(t, service); len(entries) != 1 || entries[0].Slug != "after" {
		t.Fatalf("restarted UI lost the latest data: %#v", entries)
	}
	assertUIRunRejected(t, service)
	stop()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("UI service did not stop")
	}
	assertUIUnavailable(t, service)
}

func TestUIServiceCancelsAndJoinsInflightRequests(t *testing.T) {
	entered, canceled, release := make(chan struct{}), make(chan struct{}), make(chan struct{})
	unblock := sync.OnceFunc(func() { close(release) })
	service := NewService(ServiceOptions{Dashboard: func() dashboard.Options {
		return dashboard.Options{OpenFolder: func(ctx context.Context, _ string) error {
			close(entered)
			<-ctx.Done()
			close(canceled)
			<-release
			return ctx.Err()
		}}
	}})
	service.Publish(indexer.Snapshot{Entries: []indexer.Entry{{Slug: "notes", Name: "Notes"}}})
	stop, done := startUIService(t, service)
	t.Cleanup(unblock)
	waitForUI(t, service.Ready)
	requestDone := startUIFolderRequest(t, service)
	select {
	case <-entered:
	case <-time.After(5 * time.Second):
		t.Fatal("UI request did not enter its callback")
	}
	stop()
	select {
	case <-canceled:
	case <-time.After(5 * time.Second):
		t.Fatal("service shutdown did not cancel its request context")
	}
	assertUIUnavailable(t, service)
	assertUIRunRejected(t, service)
	select {
	case <-done:
		t.Fatal("UI returned before its in-flight callback finished")
	case <-time.After(20 * time.Millisecond):
	}
	unblock()
	select {
	case <-requestDone:
	case <-time.After(5 * time.Second):
		t.Fatal("UI request did not finish cleanup")
	}
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("UI did not join its completed request")
	}
}

func TestUIRendererFailureJoinsRequestsBeforeRestart(t *testing.T) {
	for _, failure := range []string{"error", "panic"} {
		t.Run(failure, func(t *testing.T) {
			entered, canceled, release := make(chan struct{}), make(chan struct{}), make(chan struct{})
			unblock := sync.OnceFunc(func() { close(release) })
			var fail atomic.Bool
			service := NewService(ServiceOptions{
				IndexPath: filepath.Join(t.TempDir(), "index.json"),
				Dashboard: func() dashboard.Options {
					if fail.Load() {
						panic("injected renderer panic")
					}
					return dashboard.Options{OpenFolder: func(ctx context.Context, _ string) error {
						close(entered)
						<-ctx.Done()
						close(canceled)
						<-release
						return ctx.Err()
					}}
				},
			})
			service.Publish(indexer.Snapshot{Entries: []indexer.Entry{{Slug: "notes", Name: "Before"}}})
			ctx, cancel := context.WithCancel(context.Background())
			done, result := make(chan struct{}), make(chan error, 1)
			go func() {
				defer close(done)
				defer func() {
					if recovered := recover(); recovered != nil {
						result <- fmt.Errorf("recovered renderer failure: %v", recovered)
					}
				}()
				result <- service.Run(ctx)
			}()
			t.Cleanup(func() {
				unblock()
				cancel()
				select {
				case <-done:
				case <-time.After(5 * time.Second):
					t.Error("failed UI did not join cleanup")
				}
			})
			waitForUI(t, service.Ready)
			requestDone := startUIFolderRequest(t, service)
			select {
			case <-entered:
			case <-time.After(5 * time.Second):
				t.Fatal("request did not enter before the renderer fault")
			}
			if failure == "error" {
				blocked := filepath.Join(t.TempDir(), "not-a-directory")
				if err := os.WriteFile(blocked, []byte("block UI persistence"), 0o600); err != nil {
					t.Fatal(err)
				}
				view := service.current.Load().view
				view.mu.Lock()
				view.indexPath = filepath.Join(blocked, "index.json")
				view.mu.Unlock()
			} else {
				fail.Store(true)
			}
			service.Refresh()
			select {
			case <-canceled:
			case <-time.After(5 * time.Second):
				t.Fatal("renderer failure did not cancel its admitted request")
			}
			if ctx.Err() != nil {
				t.Fatal("request was canceled by the parent instead of renderer failure")
			}
			assertUIUnavailable(t, service)
			select {
			case <-done:
				t.Fatal("failed renderer returned before request cleanup")
			default:
			}
			assertUIRunRejected(t, service)
			service.Publish(indexer.Snapshot{Entries: []indexer.Entry{{Slug: "notes", Name: "After"}}})
			unblock()
			select {
			case <-requestDone:
			case <-time.After(5 * time.Second):
				t.Fatal("request did not finish after cleanup release")
			}
			select {
			case <-done:
			case <-time.After(5 * time.Second):
				t.Fatal("failed renderer did not finish joining")
			}
			want := "publish interface"
			if failure == "panic" {
				want = "injected renderer panic"
			}
			if err := <-result; err == nil || !strings.Contains(err.Error(), want) {
				t.Fatalf("renderer fault was not preserved: %v", err)
			}
			fail.Store(false)
			startUIService(t, service)
			waitForUI(t, service.Ready)
			if entries := siteViewEntries(t, service); len(entries) != 1 || entries[0].Name != "After" {
				t.Fatalf("replacement lost the queued snapshot: %#v", entries)
			}
		})
	}
}

func TestUIServiceRefreshReadsCurrentOptions(t *testing.T) {
	var title atomic.Value
	title.Store("First")
	service := NewService(ServiceOptions{Dashboard: func() dashboard.Options {
		return dashboard.Options{Title: title.Load().(string)}
	}})
	startUIService(t, service)
	waitForUI(t, service.Ready)
	title.Store("Second")
	service.Refresh()
	waitForUI(t, func() bool {
		request := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "http://beacon.test/", nil)
		response := httptest.NewRecorder()
		service.ServeHTTP(response, request)
		return strings.Contains(response.Body.String(), "<title>Second</title>")
	})
}

func TestCanceledUIRunDoesNotLoadPreferences(t *testing.T) {
	path := filepath.Join(t.TempDir(), "dashboard.json")
	if err := os.WriteFile(path, []byte("invalid JSON"), 0o600); err != nil {
		t.Fatal(err)
	}
	service := NewService(ServiceOptions{IndexPath: filepath.Join(filepath.Dir(path), "index.json")})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := service.Run(ctx); err != nil {
		t.Fatalf("canceled UI tried to load preferences: %v", err)
	}
	assertUIUnavailable(t, service)
}

func assertUIRunRejected(t *testing.T, service *Service) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	result := make(chan error, 1)
	go func() {
		defer func() {
			if failure := recover(); failure != nil {
				result <- fmt.Errorf("overlapping Run panicked: %v", failure)
			}
		}()
		result <- service.Run(ctx)
	}()
	select {
	case err := <-result:
		if err == nil || !strings.Contains(err.Error(), "already running") {
			t.Fatalf("overlapping Run = %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("replacement entered before old invocation completed")
	}
}

func startUIFolderRequest(t *testing.T, service *Service) <-chan struct{} {
	t.Helper()
	statusRequest := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "http://beacon.test/_beacon/api/status", nil)
	statusResponse := httptest.NewRecorder()
	service.ServeHTTP(statusResponse, statusRequest)
	var status struct {
		CSRFToken string `json:"csrf_token"`
	}
	if err := json.Unmarshal(statusResponse.Body.Bytes(), &status); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "http://beacon.test/_beacon/api/open-folder", strings.NewReader(`{"slug":"notes"}`))
	request.Header.Set("Origin", "http://beacon.test")
	request.Header.Set("X-Beacon-CSRF", status.CSRFToken)
	done := make(chan struct{})
	go func() {
		defer close(done)
		service.ServeHTTP(httptest.NewRecorder(), request)
	}()
	return done
}

func startUIService(t *testing.T, service *Service) (context.CancelFunc, <-chan struct{}) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		if err := service.Run(ctx); err != nil {
			t.Errorf("run UI: %v", err)
		}
	}()
	t.Cleanup(func() {
		cancel()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Error("UI service did not shut down")
		}
	})
	return cancel, done
}

func assertUIUnavailable(t *testing.T, service *Service) {
	t.Helper()
	request := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "http://beacon.test/", nil)
	response := httptest.NewRecorder()
	service.ServeHTTP(response, request)
	if response.Code != http.StatusServiceUnavailable || service.Ready() {
		t.Fatalf("inactive UI = %d, ready=%v", response.Code, service.Ready())
	}
}

func waitForUI(t *testing.T, condition func() bool) {
	t.Helper()
	deadline := time.NewTimer(5 * time.Second)
	defer deadline.Stop()
	ticker := time.NewTicker(time.Millisecond)
	defer ticker.Stop()
	for !condition() {
		select {
		case <-ticker.C:
		case <-deadline.C:
			t.Fatal("UI state did not settle")
		}
	}
}
