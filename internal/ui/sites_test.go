package ui

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/tanzir71/beacon/internal/sites/indexer"
	"github.com/tanzir71/beacon/internal/ui/dashboard"
)

func TestSiteViewOwnsPreferencesAndClonesPublishedData(t *testing.T) {
	indexPath := filepath.Join(t.TempDir(), "index.json")
	changes := 0
	view, err := NewSiteView(indexPath, func() { changes++ })
	if err != nil {
		t.Fatal(err)
	}
	data := indexer.Snapshot{Entries: []indexer.Entry{
		{Slug: "alpha", Name: "Alpha", Tags: []string{"original"}},
		{Slug: "zeta", Name: "Zeta"},
	}}
	if err := view.Publish(data, dashboard.Options{}); err != nil {
		t.Fatal(err)
	}
	data.Entries[0].Tags[0] = "mutated"
	data.Entries[0].Name = "Changed"
	if err := view.Touch("zeta"); err != nil {
		t.Fatal(err)
	}
	entries := siteViewEntries(t, view)
	if len(entries) != 2 || entries[0].Slug != "zeta" || entries[1].Name != "Alpha" || entries[1].Tags[0] != "original" {
		t.Fatalf("UI data ownership or last-use order = %#v", entries)
	}
	if changes != 1 {
		t.Fatalf("UI preference notifications = %d, want 1", changes)
	}
	view, err = NewSiteView(indexPath, nil)
	if err != nil {
		t.Fatal(err)
	}
	data.Entries[0].Name = "Alpha"
	if err := view.Publish(data, dashboard.Options{}); err != nil {
		t.Fatal(err)
	}
	if entries := siteViewEntries(t, view); entries[0].Slug != "zeta" || entries[0].LastUsed == 0 {
		t.Fatalf("UI did not persist last use: %#v", entries)
	}
}

func TestSiteViewKeepsLastGoodHandlerWhenRebuildFails(t *testing.T) {
	view, err := NewSiteView("", nil)
	if err != nil {
		t.Fatal(err)
	}
	data := indexer.Snapshot{Entries: []indexer.Entry{{Slug: "before", Name: "Before"}}}
	if err := view.Publish(data, dashboard.Options{}); err != nil {
		t.Fatal(err)
	}
	blocked := filepath.Join(t.TempDir(), "not-a-directory")
	if err := os.WriteFile(blocked, []byte("block persistence"), 0o600); err != nil {
		t.Fatal(err)
	}
	view.indexPath = filepath.Join(blocked, "index.json")
	if err := view.Publish(indexer.Snapshot{Entries: []indexer.Entry{{Slug: "after", Name: "After"}}}, dashboard.Options{}); err == nil {
		t.Fatal("UI rebuild unexpectedly succeeded")
	}
	if entries := siteViewEntries(t, view); len(entries) != 1 || entries[0].Slug != "before" {
		t.Fatalf("failed rebuild replaced the last good UI: %#v", entries)
	}
}

func siteViewEntries(t *testing.T, view http.Handler) []indexer.Entry {
	t.Helper()
	request := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "http://beacon.test/_beacon/api/apps", nil)
	response := httptest.NewRecorder()
	view.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("UI apps = %d: %s", response.Code, response.Body.String())
	}
	var entries []indexer.Entry
	if err := json.Unmarshal(response.Body.Bytes(), &entries); err != nil {
		t.Fatal(err)
	}
	return entries
}
