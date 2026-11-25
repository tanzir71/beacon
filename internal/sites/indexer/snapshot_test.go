package indexer

import "testing"

func TestSnapshotCloneOwnsAllMutableData(t *testing.T) {
	original := Snapshot{
		Entries:  []Entry{{Slug: "notes", Tags: []string{"tag"}, Databases: []string{"data.db"}, fileNames: []string{"page.html"}}},
		Warnings: []string{"warning"}, AppWarnings: map[string][]string{"notes": {"site warning"}},
		Databases: map[string]map[string]string{"notes": {"data.db": "/allowed/data.db"}},
	}
	cloned := original.Clone()
	cloned.Entries[0].Slug = "changed"
	cloned.Entries[0].Tags[0] = "changed"
	cloned.Entries[0].Databases[0] = "changed"
	cloned.Entries[0].fileNames[0] = "changed"
	cloned.Warnings[0] = "changed"
	cloned.AppWarnings["notes"][0] = "changed"
	cloned.Databases["notes"]["data.db"] = "/outside/data.db"
	if original.Entries[0].Slug != "notes" || original.Entries[0].Tags[0] != "tag" ||
		original.Entries[0].Databases[0] != "data.db" || original.Entries[0].fileNames[0] != "page.html" ||
		original.Warnings[0] != "warning" || original.AppWarnings["notes"][0] != "site warning" ||
		original.Databases["notes"]["data.db"] != "/allowed/data.db" {
		t.Fatalf("snapshot clone changed its source: %#v", original)
	}
}
