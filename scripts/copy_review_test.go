package main

import (
	"os"
	"strings"
	"testing"
)

func TestPrimaryCopyFollowsTheVoiceRules(t *testing.T) {
	t.Parallel()
	files := map[string][]string{
		"../cmd/beacon/local_https.go": {
			"local certificate authority",
			"trust store",
		},
		"../README.md": {
			"document root",
			"Edit vhost",
			"several hundred MB",
		},
		"../internal/dashboard/assets/app.js": {
			"Optional Beacon runtime.",
			"Not available on this platform",
			"Click to copy connection string",
			"HTTP ${response.status}",
		},
		"../internal/dashboard/assets/index.html": {
			"certificate authority created on this computer",
			"trust store",
		},
		"../internal/dashboard/dashboard.go": {
			"app slug exactly",
			"Tailscale Funnel is not available",
			"Tailscale Serve is not available",
		},
		"../internal/discovery/funnel.go": {"app slug exactly"},
		"../internal/firstrun/assets/example/index.html": {
			"delete the whole folder whenever you like",
		},
		"../internal/php/pool.go": {"PHP runtime is not running"},
		"../internal/runtimes/manager.go": {
			"FastCGI worker pool",
			"Optional Beacon runtime.",
		},
		"../internal/scanner/scanner.go": {
			"Apps root",
			"URL-safe",
			"was not mounted",
		},
		"../internal/doctor/doctor.go": {"✅", "⚠️", "❌"},
	}
	for path, forbidden := range files {
		content, err := os.ReadFile(path) // #nosec G304 -- paths are fixed checked-in product copy sources.
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		for _, phrase := range forbidden {
			if strings.Contains(string(content), phrase) {
				t.Errorf("%s still contains §9 violation %q", path, phrase)
			}
		}
	}

	dashboard, err := os.ReadFile("../internal/dashboard/assets/app.js") // #nosec G304 -- fixed checked-in UI source.
	if err != nil {
		t.Fatalf("read dashboard copy: %v", err)
	}
	for _, required := range []string{
		"Removing the PHP pack deletes the downloaded PHP files. Your apps and their files are untouched.",
		"Beacon-managed database data. Your apps and their files are untouched.",
		"Stopping trust makes browsers on this computer warn about Beacon again. Local HTTPS, Beacon's certificate files, and your apps are unchanged.",
	} {
		if !strings.Contains(string(dashboard), required) {
			t.Errorf("destructive add-on copy no longer explains what remains: missing %q", required)
		}
	}
	for _, required := range []string{"warningDismiss", "dismissedWarningText"} {
		if !strings.Contains(string(dashboard), required) {
			t.Errorf("ordinary dashboard warnings are not dismissible: missing %q", required)
		}
	}
	dashboardHTML, err := os.ReadFile("../internal/dashboard/assets/index.html") // #nosec G304 -- fixed checked-in UI source.
	if err != nil {
		t.Fatalf("read dashboard HTML copy: %v", err)
	}
	if !strings.Contains(string(dashboardHTML), `id="warning-dismiss"`) {
		t.Error("ordinary dashboard warning has no Dismiss action")
	}

	trustCommand, err := os.ReadFile("../cmd/beacon/local_https.go") // #nosec G304 -- fixed checked-in CLI source.
	if err != nil {
		t.Fatalf("read trust command copy: %v", err)
	}
	if !strings.Contains(string(trustCommand), "Beacon's certificate files and your apps are unchanged.") {
		t.Error("trust removal copy does not explain what remains unchanged")
	}
}
