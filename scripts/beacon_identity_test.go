package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tanzir71/beacon/internal/platform/config"
	"github.com/tanzir71/beacon/internal/platform/state"
)

func TestBeaconStorageDoesNotUseDropserveDirectories(t *testing.T) {
	configuration := config.Default()
	if configuration.Dashboard.Title != "Beacon" || configuration.Discovery.MDNSName != "beacon" {
		t.Errorf("product defaults are not Beacon's: %#v", configuration)
	}
	if len(configuration.Server.AppsRoots) != 1 || filepath.Base(filepath.Dir(configuration.Server.AppsRoots[0])) != "Beacon" {
		t.Errorf("Apps folder is not isolated: %v", configuration.Server.AppsRoots)
	}
	for name, getPath := range map[string]func() (string, error){"config": config.DefaultPath, "state": state.DefaultPath} {
		path, err := getPath()
		if err != nil {
			t.Fatal(err)
		}
		if !strings.EqualFold(filepath.Base(filepath.Dir(path)), "beacon") {
			t.Errorf("%s uses another product's directory: %s", name, path)
		}
	}
}

func TestBeaconInstallationIdentifiersAreIndependent(t *testing.T) {
	for path, required := range map[string][]string{
		"../go.mod": {"module github.com/tanzir71/beacon"},
		"../internal/platform/autostart/autostart_windows.go": {`const taskName = "Beacon"`},
		"../internal/platform/autostart/autostart_linux.go":   {`const systemdUnitName = "beacon.service"`},
		"../internal/platform/autostart/launchd.go":           {`const launchAgentLabel = "dev.beacon.agent"`},
		"../cmd/beacon/cli_commands.go":                       {"name=Beacon"},
		"../packaging/windows/beacon.iss": {
			"AppName=Beacon", `DefaultDirName={autopf}\Beacon`,
			`Source: "{#SourceDir}\beacon.exe"`, `Source: "{#SourceDir}\beacon-cli.exe"`,
			`Name: "{localappdata}\Beacon"`, `Name: "{userappdata}\Beacon"`,
		},
	} {
		t.Run(path, func(t *testing.T) {
			data, err := os.ReadFile(path) // #nosec G304 -- paths are the fixed repository files listed above.
			if err != nil {
				t.Fatal(err)
			}
			for _, marker := range required {
				if !strings.Contains(string(data), marker) {
					t.Errorf("missing independent identifier %q", marker)
				}
			}
			if strings.HasSuffix(path, ".iss") {
				for _, forbidden := range []string{"BC7032D9-0DC2-4CA8-B936-B0BF5E271450", "Drop" + "serve", "drop" + "serve"} {
					if strings.Contains(string(data), forbidden) {
						t.Errorf("installer retains another product's identifier %q", forbidden)
					}
				}
			}
		})
	}
	if got := buildSpecs("windows"); got[0].name != "beacon.exe" || got[1].name != "beacon-cli.exe" {
		t.Errorf("Windows outputs are not isolated: %#v", got)
	}
}
