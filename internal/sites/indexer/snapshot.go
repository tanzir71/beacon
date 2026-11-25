package indexer

// Snapshot is site data published independently of interface rendering.
// Database paths are internal read-only capabilities, never browser-supplied.
type Snapshot struct {
	Entries     []Entry
	Warnings    []string
	AppWarnings map[string][]string
	Databases   map[string]map[string]string
}

// Clone gives a consumer exclusive ownership of every mutable collection.
func (snapshot Snapshot) Clone() Snapshot {
	cloned := Snapshot{
		Entries:     append([]Entry(nil), snapshot.Entries...),
		Warnings:    append([]string(nil), snapshot.Warnings...),
		AppWarnings: make(map[string][]string, len(snapshot.AppWarnings)),
		Databases:   make(map[string]map[string]string, len(snapshot.Databases)),
	}
	for index := range cloned.Entries {
		cloned.Entries[index].Tags = append([]string(nil), snapshot.Entries[index].Tags...)
		cloned.Entries[index].Databases = append([]string(nil), snapshot.Entries[index].Databases...)
		cloned.Entries[index].fileNames = append([]string(nil), snapshot.Entries[index].fileNames...)
	}
	for slug, warnings := range snapshot.AppWarnings {
		cloned.AppWarnings[slug] = append([]string(nil), warnings...)
	}
	for slug, paths := range snapshot.Databases {
		cloned.Databases[slug] = make(map[string]string, len(paths))
		for name, path := range paths {
			cloned.Databases[slug][name] = path
		}
	}
	return cloned
}
