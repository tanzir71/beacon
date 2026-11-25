// Package server composes scanning, static handlers, and the atomic router.
package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/tanzir71/beacon/internal/links/discovery"
	"github.com/tanzir71/beacon/internal/sites/app"
	"github.com/tanzir71/beacon/internal/sites/indexer"
	"github.com/tanzir71/beacon/internal/sites/router"
	"github.com/tanzir71/beacon/internal/sites/scanner"
	staticserver "github.com/tanzir71/beacon/internal/sites/static"
	"github.com/tanzir71/beacon/internal/sites/supervisor"
	"github.com/tanzir71/beacon/internal/sites/watcher"
	"github.com/tanzir71/beacon/internal/ui"
	"github.com/tanzir71/beacon/internal/ui/dashboard"
)

type snapshot struct {
	scan scanner.Result
}

type liveServerConfig struct {
	pinToRoot string
}

// Server keeps immutable scan and handler snapshots behind atomic swaps.
type Server struct {
	router      *router.Router
	http        http.Handler
	snapshot    atomic.Pointer[snapshot]
	watcher     *watcher.Watcher
	options     Options
	reconcileMu sync.Mutex
	rebuilds    atomic.Uint64
	events      *eventHub
	supervisor  *supervisor.Manager
	view        *ui.SiteView
	liveConfig  atomic.Pointer[liveServerConfig]
	activityMu  sync.Mutex
	activityWG  sync.WaitGroup
	closing     bool
	logClients  chan struct{}
}

// Options configures scanning and optional machine-state persistence.
type Options struct {
	Scanner              scanner.Options
	IndexPath            string
	DashboardTitle       string
	DashboardTheme       string
	PinToRoot            string
	Supervisor           supervisor.Options
	AsyncCommandStart    bool
	Warnings             []string
	Discovery            func() discovery.Snapshot
	Funnel               *discovery.FunnelManager
	SetTailscaleServe    func(context.Context, bool) error
	LocalHTTPSStatus     func() dashboard.LocalHTTPSStatus
	SetLocalHTTPS        func(context.Context, bool) error
	SetLocalTrust        func(bool) error
	RootCertificate      func() ([]byte, error)
	DismissNetworkChange func() error
	PHPHandler           func(app.App) (http.Handler, error)
	Addons               func() []dashboard.AddonStatus
	ChangeAddon          func(context.Context, string, string) error
	RestartApp           func(context.Context, string) error
	OpenFolder           func(context.Context, string) error
	Update               func() dashboard.UpdateNotice
	// Publish transfers raw site data to an external UI owner. It must enqueue
	// without blocking or rendering. Nil preserves standalone server composition.
	Publish func(indexer.Snapshot)
}

// New scans the configured roots and registered apps, then mounts every app.
func New(options scanner.Options) (*Server, error) {
	return NewWithOptions(Options{Scanner: options})
}

// NewWithOptions creates a server and atomically persists its dashboard index when requested.
func NewWithOptions(options Options) (*Server, error) {
	supervisorOptions := options.Supervisor
	if options.IndexPath != "" {
		stateDirectory := filepath.Dir(options.IndexPath)
		if supervisorOptions.LogDirectory == "" {
			supervisorOptions.LogDirectory = filepath.Join(stateDirectory, "logs")
		}
		if supervisorOptions.PortPath == "" {
			supervisorOptions.PortPath = filepath.Join(stateDirectory, "ports.json")
		}
	}
	server := &Server{
		router:     router.New(nil),
		options:    options,
		events:     newEventHub(),
		logClients: make(chan struct{}, 64),
	}
	if options.Publish == nil {
		view, err := ui.NewSiteView(options.IndexPath, server.events.publish)
		if err != nil {
			return nil, err
		}
		server.view = view
	}
	previousSupervisorChange := supervisorOptions.OnChange
	supervisorOptions.OnChange = func() {
		if previousSupervisorChange != nil {
			previousSupervisorChange()
		}
		_ = server.reconcile()
	}
	server.supervisor = supervisor.NewManager(supervisorOptions)
	server.liveConfig.Store(&liveServerConfig{pinToRoot: options.PinToRoot})
	handler := http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		const maximumRequestBody = 64 << 20
		if request.ContentLength > maximumRequestBody {
			http.Error(response, "Beacon accepts request bodies up to 64 MB.", http.StatusRequestEntityTooLarge)
			return
		}
		request.Body = http.MaxBytesReader(response, request.Body, maximumRequestBody)
		if request.URL.Path == "/_beacon/api/events" {
			server.events.serveHTTP(response, request)
			return
		}
		if strings.HasPrefix(request.URL.Path, "/_beacon/api/logs/") {
			server.serveCommandLogs(response, request)
			return
		}
		live := server.liveConfig.Load()
		pinToRoot := ""
		if live != nil {
			pinToRoot = live.pinToRoot
		}
		reservedDashboardPath := request.URL.Path == "/_beacon" || strings.HasPrefix(request.URL.Path, "/_beacon/")
		if pinToRoot != "" && request.URL.Path != "/" && !reservedDashboardPath && server.hasExplicitAppPath(request.URL.Path) {
			server.router.ServeHTTP(response, request)
			return
		}
		if pinToRoot != "" && !reservedDashboardPath {
			proxied := request.Clone(request.Context())
			proxied.URL.Path = "/" + pinToRoot + request.URL.Path
			proxied.URL.RawPath = ""
			if request.URL.Path == "/" {
				server.recordAppUseAsync(pinToRoot)
			}
			server.router.ServeHTTP(response, proxied)
			return
		}
		if request.URL.Path == "/" || strings.HasPrefix(request.URL.Path, "/_beacon/") {
			if server.view == nil {
				http.NotFound(response, request)
				return
			}
			server.view.ServeHTTP(response, request)
			return
		}
		if request.URL.Path == "/_beacon" {
			http.Redirect(response, request, "/_beacon/", http.StatusMovedPermanently)
			return
		}
		if server.view != nil {
			trimmed := strings.Trim(request.URL.Path, "/")
			if trimmed != "" && !strings.Contains(trimmed, "/") {
				server.recordAppUseAsync(trimmed)
			}
		}
		server.router.ServeHTTP(response, request)
	})
	server.http = handler
	if err := server.reconcile(); err != nil {
		return nil, err
	}
	liveWatcher, err := watcher.New(watcher.Options{
		Roots:     options.Scanner.Roots,
		Reconcile: server.reconcile,
	})
	if err != nil {
		return nil, err
	}
	server.watcher = liveWatcher
	return server, nil
}

func (server *Server) hasExplicitAppPath(requestPath string) bool {
	trimmed := strings.TrimPrefix(requestPath, "/")
	slug, _, _ := strings.Cut(trimmed, "/")
	current := server.snapshot.Load()
	if current == nil {
		return false
	}
	for _, application := range current.scan.Apps {
		if application.Slug == slug {
			return true
		}
	}
	return false
}

// Handler returns the live HTTP handler.
func (server *Server) Handler() http.Handler {
	return server.http
}

// Scan returns the discovery snapshot used to build the current mount table.
func (server *Server) Scan() scanner.Result {
	current := server.snapshot.Load()
	if current == nil {
		return scanner.Result{}
	}
	return current.scan
}

// Close stops live filesystem watching.
func (server *Server) Close() error {
	server.activityMu.Lock()
	server.closing = true
	server.activityMu.Unlock()
	server.activityWG.Wait()

	var closeErrors []error
	if server.watcher != nil {
		closeErrors = append(closeErrors, server.watcher.Close())
	}
	if server.supervisor != nil {
		closeErrors = append(closeErrors, server.supervisor.Close())
	}
	return errors.Join(closeErrors...)
}

func (server *Server) recordAppUseAsync(slug string) {
	server.activityMu.Lock()
	if server.closing {
		server.activityMu.Unlock()
		return
	}
	server.activityWG.Add(1)
	server.activityMu.Unlock()
	go func() {
		defer server.activityWG.Done()
		server.recordAppUse(slug)
	}()
}

// RebuildCount returns the number of successfully published mount snapshots.
func (server *Server) RebuildCount() uint64 {
	return server.rebuilds.Load()
}

func (server *Server) recordAppUse(slug string) {
	if server.view != nil {
		_ = server.view.Touch(slug)
	}
}

// Reconcile performs one full read-only scan and publishes the resulting routes.
func (server *Server) Reconcile() error {
	return server.reconcile()
}

// UpdateConfiguration atomically publishes hot-reloadable scan and dashboard
// settings, then rebuilds the immutable routing snapshot.
func (server *Server) UpdateConfiguration(scannerOptions scanner.Options, title, theme, pinToRoot string, firstPort, lastPort int) error {
	server.reconcileMu.Lock()
	defer server.reconcileMu.Unlock()
	previousScanner := server.options.Scanner
	previousTitle := server.options.DashboardTitle
	previousTheme := server.options.DashboardTheme
	previousPin := server.options.PinToRoot
	server.options.Scanner = scannerOptions
	server.options.DashboardTitle = title
	server.options.DashboardTheme = theme
	server.options.PinToRoot = pinToRoot
	server.liveConfig.Store(&liveServerConfig{pinToRoot: pinToRoot})
	previousFirst, previousLast := server.supervisor.SetPortRange(firstPort, lastPort)
	if err := server.reconcileLocked(); err != nil {
		server.options.Scanner = previousScanner
		server.options.DashboardTitle = previousTitle
		server.options.DashboardTheme = previousTheme
		server.options.PinToRoot = previousPin
		server.liveConfig.Store(&liveServerConfig{pinToRoot: previousPin})
		server.supervisor.SetPortRange(previousFirst, previousLast)
		return err
	}
	return nil
}

func (server *Server) reconcile() error {
	server.reconcileMu.Lock()
	defer server.reconcileMu.Unlock()
	return server.reconcileLocked()
}

func (server *Server) reconcileLocked() error {
	result, err := scanner.Scan(server.options.Scanner)
	if err != nil {
		return err
	}
	mounts := make([]router.Mount, 0, len(result.Apps))
	for applicationIndex := range result.Apps {
		application := result.Apps[applicationIndex]
		var applicationHandler http.Handler
		switch application.Kind {
		case app.KindCommand:
			if server.options.AsyncCommandStart {
				applicationHandler, err = server.supervisor.HandlerAsync(application)
			} else {
				applicationHandler, err = server.supervisor.Handler(application)
			}
			if commandState, found := server.supervisor.Snapshot(application.Slug); found {
				application.Status = commandState.Status
				application.Port = commandState.Port
				application.PrefersOwnPort = commandState.PrefersOwnPort
				result.Apps[applicationIndex].Status = commandState.Status
				result.Apps[applicationIndex].Port = commandState.Port
				result.Apps[applicationIndex].PrefersOwnPort = commandState.PrefersOwnPort
			}
		case app.KindPHP:
			if server.options.PHPHandler == nil {
				application.Status = "needs-runtime"
				result.Apps[applicationIndex].Status = application.Status
				applicationHandler = needsPHPRuntimeHandler()
			} else {
				applicationHandler, err = server.options.PHPHandler(application)
				if err == nil && applicationHandler == nil {
					application.Status = "needs-runtime"
					result.Apps[applicationIndex].Status = application.Status
					applicationHandler = needsPHPRuntimeHandler()
				}
			}
		default:
			applicationHandler = staticserver.New(application)
		}
		if err != nil {
			return err
		}
		mounts = append(mounts, router.Mount{
			App:     application,
			Handler: applicationHandler,
		})
	}
	if err := server.supervisor.Reconcile(result.Apps); err != nil {
		return err
	}
	entries := indexer.Build(result.Apps)
	databasePaths := make(map[string]map[string]string)
	for _, application := range result.Apps {
		if len(application.Databases) == 0 {
			continue
		}
		files := make(map[string]string, len(application.Databases))
		for _, relative := range application.Databases {
			files[relative] = filepath.Join(application.Path, filepath.FromSlash(relative))
		}
		databasePaths[application.Slug] = files
	}
	appWarnings := make(map[string][]string)
	for _, warning := range result.Warnings {
		for _, application := range result.Apps {
			relative, relativeErr := filepath.Rel(application.Path, warning.Path)
			if relativeErr == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) && !filepath.IsAbs(relative) {
				appWarnings[application.Slug] = append(appWarnings[application.Slug], warning.Message)
				break
			}
		}
	}
	publication := indexer.Snapshot{
		Entries: entries, Warnings: warningMessages(result.Warnings),
		AppWarnings: appWarnings, Databases: databasePaths,
	}
	if server.view != nil {
		if err := server.view.Publish(publication, server.dashboardOptions()); err != nil {
			return err
		}
	}
	server.router.Swap(mounts)
	server.snapshot.Store(&snapshot{scan: result})
	server.rebuilds.Add(1)
	if server.options.Publish != nil {
		server.options.Publish(publication)
	}
	server.events.publish()
	return nil
}

func (server *Server) dashboardOptions() dashboard.Options {
	return dashboard.Options{
		Title:                server.options.DashboardTitle,
		Theme:                server.options.DashboardTheme,
		Warnings:             server.options.Warnings,
		Discovery:            server.options.Discovery,
		Funnel:               server.options.Funnel,
		SetTailscaleServe:    server.options.SetTailscaleServe,
		LocalHTTPSStatus:     server.options.LocalHTTPSStatus,
		SetLocalHTTPS:        server.options.SetLocalHTTPS,
		SetLocalTrust:        server.options.SetLocalTrust,
		RootCertificate:      server.options.RootCertificate,
		DismissNetworkChange: server.options.DismissNetworkChange,
		Addons:               server.options.Addons,
		ChangeAddon:          server.options.ChangeAddon,
		RestartApp: func(ctx context.Context, slug string) error {
			if server.options.RestartApp != nil {
				return server.options.RestartApp(ctx, slug)
			}
			return server.supervisor.Restart(slug)
		},
		OpenFolder: server.options.OpenFolder,
		Rescan:     server.Reconcile,
		Update:     server.options.Update,
	}
}

func needsPHPRuntimeHandler() http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		response.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = response.Write([]byte(`<!doctype html><html lang="en"><meta charset="utf-8"><title>PHP needed · Beacon</title><body><main><h1>This app needs PHP.</h1><p>Install the optional PHP pack from Beacon Add-ons, then try again.</p></main></body></html>`))
	})
}

func (server *Server) serveCommandLogs(response http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet && request.Method != http.MethodHead {
		response.Header().Set("Allow", "GET, HEAD")
		http.Error(response, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	slug := strings.TrimPrefix(request.URL.Path, "/_beacon/api/logs/")
	if slug == "" || strings.Contains(slug, "/") {
		http.NotFound(response, request)
		return
	}
	commandState, found := server.supervisor.Snapshot(slug)
	if !found {
		http.NotFound(response, request)
		return
	}
	if strings.Contains(request.Header.Get("Accept"), "text/event-stream") {
		server.streamCommandLogs(response, request, slug, commandState)
		return
	}
	response.Header().Set("Content-Type", "application/json; charset=utf-8")
	response.Header().Set("Cache-Control", "no-store")
	if err := json.NewEncoder(response).Encode(commandState); err != nil {
		http.Error(response, "Beacon could not encode these logs.", http.StatusInternalServerError)
	}
}

func (server *Server) streamCommandLogs(response http.ResponseWriter, request *http.Request, slug string, initial supervisor.Snapshot) {
	flusher, ok := response.(http.Flusher)
	if !ok {
		http.Error(response, "Live logs are not available.", http.StatusInternalServerError)
		return
	}
	if err := http.NewResponseController(response).SetWriteDeadline(time.Time{}); err != nil && !errors.Is(err, http.ErrNotSupported) {
		http.Error(response, "Beacon could not prepare the live log stream.", http.StatusInternalServerError)
		return
	}
	if request.Method != http.MethodHead {
		select {
		case server.logClients <- struct{}{}:
			defer func() { <-server.logClients }()
		default:
			http.Error(response, "Beacon has reached its live-log connection limit.", http.StatusServiceUnavailable)
			return
		}
	}
	response.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	response.Header().Set("Cache-Control", "no-store")
	response.Header().Set("X-Accel-Buffering", "no")
	writeSnapshot := func(snapshot supervisor.Snapshot) bool {
		var payload bytes.Buffer
		if err := json.NewEncoder(&payload).Encode(snapshot); err != nil {
			return false
		}
		_, err := fmt.Fprintf(response, "event: logs\ndata: %s\n", payload.Bytes())
		if err == nil {
			flusher.Flush()
		}
		return err == nil
	}
	if request.Method == http.MethodHead || !writeSnapshot(initial) {
		return
	}
	previous := initial
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-request.Context().Done():
			return
		case <-ticker.C:
			current, found := server.supervisor.Snapshot(slug)
			if !found {
				return
			}
			if current == previous {
				continue
			}
			if !writeSnapshot(current) {
				return
			}
			previous = current
		}
	}
}

func warningMessages(warnings []scanner.Warning) []string {
	messages := make([]string, 0, len(warnings))
	for _, warning := range warnings {
		messages = append(messages, warning.Message)
	}
	return messages
}
