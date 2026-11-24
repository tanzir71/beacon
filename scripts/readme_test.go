package main

import (
	"os"
	"strings"
	"testing"
)

func TestNoSupersededHostingLandingIsPublished(t *testing.T) {
	t.Parallel()
	readme, err := os.ReadFile("../README.md") // #nosec G304 -- fixed checked-in README.
	if err != nil {
		t.Fatalf("read README: %v", err)
	}
	if strings.Contains(string(readme), "docs/screenshots/") {
		t.Error("README still advertises the superseded hosting dashboard")
	}
	for _, path := range []string{"../docs/index.html", "../docs/screenshots"} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Errorf("superseded hosting page/image path remains published: %s (%v)", path, err)
		}
	}
}

func TestREADMEPublishesKeylessReleaseVerificationCommand(t *testing.T) {
	t.Parallel()
	readme, err := os.ReadFile("../README.md") // #nosec G304 -- fixed checked-in README.
	if err != nil {
		t.Fatalf("read README: %v", err)
	}
	text := string(readme)
	for _, marker := range []string{
		"cosign verify-blob",
		"--bundle \"$ARTIFACT.sigstore.json\"",
		"--certificate-identity-regexp",
		"--certificate-oidc-issuer https://token.actions.githubusercontent.com",
	} {
		if !strings.Contains(text, marker) {
			t.Errorf("README verification instructions are missing %q", marker)
		}
	}
}
