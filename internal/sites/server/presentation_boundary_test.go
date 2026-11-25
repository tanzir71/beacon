package server_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/tanzir71/beacon/internal/sites/indexer"
	"github.com/tanzir71/beacon/internal/sites/scanner"
	beaconr "github.com/tanzir71/beacon/internal/sites/server"
	"github.com/tanzir71/beacon/internal/ui"
	"github.com/tanzir71/beacon/internal/ui/dashboard"
)

func TestSitePublicationDoesNotDependOnUIState(t *testing.T) {
	root, stateDirectory := t.TempDir(), t.TempDir()
	writeDashboardFixture(t, root, "first", "First", "")
	if err := os.WriteFile(filepath.Join(stateDirectory, "dashboard.json"), []byte("broken UI preferences"), 0o600); err != nil {
		t.Fatal(err)
	}
	var published indexer.Snapshot
	server, err := beaconr.NewWithOptions(beaconr.Options{
		Scanner: scanner.Options{Roots: []string{root}}, IndexPath: filepath.Join(stateDirectory, "index.json"),
		Publish: func(snapshot indexer.Snapshot) { published = snapshot },
	})
	if err != nil {
		t.Fatalf("UI state prevented Sites startup: %v", err)
	}
	cleanupServer(t, server)
	if len(published.Entries) != 1 || published.Entries[0].Slug != "first" {
		t.Fatalf("raw site snapshot = %#v", published)
	}
	request := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "http://beacon.test/first/", nil)
	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("site = %d: %s", response.Code, response.Body.String())
	}
	if _, err := ui.NewSiteView(filepath.Join(stateDirectory, "index.json"), nil); err == nil {
		t.Fatal("corrupt preferences did not fail the independently owned UI")
	}
	if _, err := os.Stat(filepath.Join(stateDirectory, "index.json")); !os.IsNotExist(err) {
		t.Fatalf("Sites wrote the UI-owned index: %v", err)
	}
}

func TestUIRebuildFailureDoesNotPreventNewSiteRoutes(t *testing.T) {
	root := t.TempDir()
	updates := make(chan indexer.Snapshot, 8)
	server, err := beaconr.NewWithOptions(beaconr.Options{
		Scanner: scanner.Options{Roots: []string{root}},
		Publish: func(snapshot indexer.Snapshot) { updates <- snapshot },
	})
	if err != nil {
		t.Fatal(err)
	}
	cleanupServer(t, server)
	stateDirectory := t.TempDir()
	blocked := filepath.Join(stateDirectory, "not-a-directory")
	view, err := ui.NewSiteView(filepath.Join(blocked, "index.json"), nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(blocked, []byte("block UI persistence"), 0o600); err != nil {
		t.Fatal(err)
	}
	writeDashboardFixture(t, root, "new-site", "New Site", "")
	if err := server.Reconcile(); err != nil {
		t.Fatalf("UI failure prevented Sites reconciliation: %v", err)
	}
	var latest indexer.Snapshot
	for range 2 {
		select {
		case latest = <-updates:
		case <-time.After(5 * time.Second):
			t.Fatal("Sites did not publish its snapshot")
		}
	}
	if err := view.Publish(latest, dashboard.Options{}); err == nil {
		t.Fatal("expected independent UI rebuild to fail on blocked persistence")
	}
	request := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "http://beacon.test/new-site/", nil)
	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("new site = %d: %s", response.Code, response.Body.String())
	}
}

func TestIndependentUIOutlivesSiteOwner(t *testing.T) {
	var data indexer.Snapshot
	server, err := beaconr.NewWithOptions(beaconr.Options{
		Scanner: scanner.Options{Roots: []string{t.TempDir()}},
		Publish: func(snapshot indexer.Snapshot) { data = snapshot },
	})
	if err != nil {
		t.Fatal(err)
	}
	cleanupServer(t, server)
	view, err := ui.NewSiteView("", nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := view.Publish(data, dashboard.Options{}); err != nil {
		t.Fatal(err)
	}
	if err := server.Close(); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/", "/_beacon/api/status"} {
		request := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "http://beacon.test"+path, nil)
		response := httptest.NewRecorder()
		view.ServeHTTP(response, request)
		if response.Code != http.StatusOK {
			t.Errorf("independent UI %s stopped with Sites: %d", path, response.Code)
		}
	}
}
