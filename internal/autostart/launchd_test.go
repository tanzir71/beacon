package autostart

import (
	"strings"
	"testing"
)

func TestLaunchAgentRunsInBackgroundAndRestartsOnFailure(t *testing.T) {
	plist := string(makeLaunchAgent(`/Applications/Beacon & Tools/beacon`))
	for _, required := range []string{
		`<key>Label</key>`,
		`<string>dev.beacon.agent</string>`,
		`<key>ProgramArguments</key>`,
		`<string>/Applications/Beacon &amp; Tools/beacon</string>`,
		`<string>--background</string>`,
		`<key>RunAtLoad</key>`,
		`<true/>`,
		`<key>KeepAlive</key>`,
		`<key>SuccessfulExit</key>`,
		`<false/>`,
		`<key>ProcessType</key>`,
		`<string>Background</string>`,
	} {
		if !strings.Contains(plist, required) {
			t.Errorf("launch agent does not contain %q:\n%s", required, plist)
		}
	}
}
