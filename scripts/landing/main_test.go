package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPublishedLandingIsCurrent(t *testing.T) {
	if err := generate("../..", true); err != nil {
		t.Fatal(err)
	}
}

func TestPagesRootHasAnActualBilingualLanding(t *testing.T) {
	for _, path := range []string{"index.html", "en/index.html", "bn/index.html"} {
		data, err := os.ReadFile(filepath.Join("../..", path)) // #nosec G304 -- fixed repository outputs.
		if err != nil {
			t.Fatal(err)
		}
		for _, marker := range []string{"<!DOCTYPE html>", "<h1>", "id=\"build\"", "data-language=\"bn\"", "data-language=\"en\""} {
			if !strings.Contains(string(data), marker) {
				t.Errorf("%s lacks %q", path, marker)
			}
		}
		if strings.Count(string(data), "<details>") != 6 || strings.Contains(string(data), "/releases/latest/download/") {
			t.Errorf("%s has incomplete FAQs or an unverified download", path)
		}
		total := len(data)
		for _, asset := range []string{"landing.css", "landing.js", "anek-latin.woff2"} {
			info, err := os.Stat(filepath.Join("../../assets", asset))
			if err != nil {
				t.Fatal(err)
			}
			total += int(info.Size())
		}
		if total > 150*1024 {
			t.Errorf("%s initial assets = %d bytes, over 150 KiB", path, total)
		}
		t.Logf("%s initial HTML, CSS, JS and font: %d bytes", path, total)
	}
}
