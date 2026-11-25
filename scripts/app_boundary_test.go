package main

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestSharedHTTPTransportIsNotOwnedBySites(t *testing.T) {
	const directory = "../internal/platform/httptransport"
	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatal(err)
	}
	files := 0
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		files++
		file, err := parser.ParseFile(token.NewFileSet(), filepath.Join(directory, entry.Name()), nil, parser.ImportsOnly)
		if err != nil {
			t.Fatal(err)
		}
		for _, imported := range file.Imports {
			path, err := strconv.Unquote(imported.Path.Value)
			if err != nil {
				t.Fatal(err)
			}
			if strings.HasPrefix(path, "github.com/tanzir71/beacon/internal/") {
				t.Errorf("shared HTTP transport depends on an app service: %s imports %s", entry.Name(), path)
			}
		}
	}
	if files == 0 {
		t.Fatal("shared transport package has no implementation")
	}
	for _, name := range []string{"listener_runtime.go", "https_runtime.go"} {
		if _, err := os.Stat(filepath.Join("../internal/sites/server", name)); !os.IsNotExist(err) {
			t.Errorf("Sites still owns shared transport %s (%v)", name, err)
		}
	}
}
