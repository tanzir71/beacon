package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPriorProjectDocumentsAreHistorical(t *testing.T) {
	for _, name := range []string{"DROPSERVE-HANDOVER.md", "BLOCKED-m8-windows-trust-confirmation.md"} {
		if _, err := os.Stat(filepath.Join("..", name)); !os.IsNotExist(err) {
			t.Errorf("prior project document remains at root: %s (%v)", name, err)
		}
		if _, err := os.Stat(filepath.Join("..", "docs", "history", name)); err != nil {
			t.Errorf("historical document missing: %s (%v)", name, err)
		}
	}
	for number := 1; number <= 14; number++ {
		prefix := fmt.Sprintf("%03d-*.md", number)
		matches, err := filepath.Glob(filepath.Join("..", "docs", "history", "adr", prefix))
		if err != nil || len(matches) != 1 {
			t.Errorf("expected one preserved ADR %03d, got %v (%v)", number, matches, err)
		}
		active, err := filepath.Glob(filepath.Join("..", "docs", "adr", prefix))
		if err != nil || len(active) != 0 {
			t.Errorf("prior ADR %03d remains active: %v (%v)", number, active, err)
		}
	}
	for _, path := range []string{
		"../docs/history/README.md", "../docs/adr/015-one-app.md",
		"../docs/adr/017-delivery-paths.md", "../docs/adr/018-mesh-storage.md",
	} {
		if _, err := os.Stat(path); err != nil {
			t.Errorf("missing product decision or historical notice %s: %v", path, err)
		}
	}
}

func TestAppDocumentationDefinesOneProduct(t *testing.T) {
	for path, markers := range map[string][]string{
		"../README.md":               {"Blackout Beacon is one app", "messages", "people", "sites", "off-grid", "U1", "Go 1.26"},
		"../CONTRIBUTING.md":         {"I1", "I12", "TestServiceFailureIsContained", "[U1]", "make check"},
		"../SECURITY.md":             {"forward secrecy", "mailbox", "panic wipe", "recipient", "not implemented"},
		"../docs/adr/015-one-app.md": {"withdrawn first draft", "sites", "links", "events", "ui"},
	} {
		data, err := os.ReadFile(path) // #nosec G304 -- fixed checked-in documentation listed above.
		if err != nil {
			t.Error(err)
			continue
		}
		for _, marker := range markers {
			if !strings.Contains(string(data), marker) {
				t.Errorf("%s does not explain %q", path, marker)
			}
		}
	}
}
