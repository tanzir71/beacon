// Package ui owns interface publication independently of site routing.
package ui

import (
	"context"
	"errors"
	"net/http"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/tanzir71/beacon/internal/sites/indexer"
	"github.com/tanzir71/beacon/internal/sites/sqlitebrowser"
	"github.com/tanzir71/beacon/internal/ui/dashboard"
	"github.com/tanzir71/beacon/internal/ui/preferences"
)

type siteViewSnapshot struct {
	data    indexer.Snapshot
	options dashboard.Options
	handler http.Handler
}

// SiteView owns site presentation, preferences and its last good HTTP handler.
// It opens no listeners and starts no workers; its service owner controls when
// raw site data is rendered and how publication failures are handled.
type SiteView struct {
	mu          sync.Mutex
	current     atomic.Pointer[siteViewSnapshot]
	indexPath   string
	preferences *preferences.Store
	onChange    func()
}

// NewSiteView loads UI preferences without scanning or starting any sites.
func NewSiteView(indexPath string, onChange func()) (*SiteView, error) {
	view := &SiteView{indexPath: indexPath, onChange: onChange}
	if indexPath != "" {
		store, err := preferences.Open(filepath.Join(filepath.Dir(indexPath), "dashboard.json"))
		if err != nil {
			return nil, err
		}
		view.preferences = store
	}
	return view, nil
}

// Publish clones site data, applies UI preferences and replaces only this view.
// A failure leaves the last good handler in place and affects no site routes.
func (view *SiteView) Publish(data indexer.Snapshot, options dashboard.Options) error {
	view.mu.Lock()
	defer view.mu.Unlock()
	return view.publishLocked(data.Clone(), options)
}

func (view *SiteView) publishLocked(data indexer.Snapshot, options dashboard.Options) error {
	entries := append([]indexer.Entry{}, data.Entries...)
	for index := range entries {
		if view.preferences == nil {
			break
		}
		settings := view.preferences.Get(entries[index].Slug)
		if settings.Pinned != nil {
			entries[index].Pinned = *settings.Pinned
		}
		if settings.Hidden != nil {
			entries[index].Hidden = *settings.Hidden
		}
		if !settings.LastUsed.IsZero() {
			entries[index].LastUsed = settings.LastUsed.Unix()
		}
	}
	sort.SliceStable(entries, func(first, second int) bool {
		if entries[first].Pinned != entries[second].Pinned {
			return entries[first].Pinned
		}
		if entries[first].LastUsed != entries[second].LastUsed {
			return entries[first].LastUsed > entries[second].LastUsed
		}
		if entries[first].MTime != entries[second].MTime {
			return entries[first].MTime > entries[second].MTime
		}
		return strings.ToLower(entries[first].Name) < strings.ToLower(entries[second].Name)
	})
	options.Warnings = append([]string(nil), options.Warnings...)
	baseOptions := options
	options.Warnings = append(append([]string{}, options.Warnings...), data.Warnings...)
	options.AppWarnings = data.AppWarnings
	if options.ChangeAppSettings == nil {
		options.ChangeAppSettings = view.changeAppSettings
	}
	if options.BrowseDatabase == nil {
		options.BrowseDatabase = func(ctx context.Context, slug, file string) (sqlitebrowser.Snapshot, error) {
			files, found := data.Databases[slug]
			if !found {
				return sqlitebrowser.Snapshot{}, errors.New("app has no discovered databases")
			}
			path, found := files[file]
			if !found {
				return sqlitebrowser.Snapshot{}, errors.New("database was not discovered in this app")
			}
			return sqlitebrowser.Browse(ctx, path)
		}
	}
	handler, err := dashboard.NewWithOptions(entries, options)
	if err != nil {
		return err
	}
	if view.indexPath != "" {
		if err := indexer.Save(view.indexPath, entries); err != nil {
			return err
		}
	}
	view.current.Store(&siteViewSnapshot{data: data, options: baseOptions, handler: handler})
	return nil
}

// ServeHTTP serves the latest successful UI publication, independent of Sites.
func (view *SiteView) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	current := view.current.Load()
	if current == nil {
		http.Error(response, "Beacon is preparing the interface.", http.StatusServiceUnavailable)
		return
	}
	current.handler.ServeHTTP(response, request)
}

// Touch records known-site use in UI state without rescanning site folders.
func (view *SiteView) Touch(slug string) error {
	return view.updatePreferences(slug, func() error { return view.preferences.Touch(slug) })
}

func (view *SiteView) changeAppSettings(_ context.Context, slug string, change dashboard.AppSettingsChange) error {
	return view.updatePreferences(slug, func() error { return view.preferences.Set(slug, change.Pinned, change.Hidden) })
}

func (view *SiteView) updatePreferences(slug string, change func() error) error {
	view.mu.Lock()
	current := view.current.Load()
	if view.preferences == nil || current == nil {
		view.mu.Unlock()
		return errors.New("interface preferences are unavailable")
	}
	found := false
	for _, entry := range current.data.Entries {
		if entry.Slug == slug {
			found = true
			break
		}
	}
	if !found {
		view.mu.Unlock()
		return nil
	}
	err := change()
	if err == nil {
		err = view.publishLocked(current.data, current.options)
	}
	view.mu.Unlock()
	if err == nil && view.onChange != nil {
		view.onChange()
	}
	return err
}
