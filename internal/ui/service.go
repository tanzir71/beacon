package ui

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"

	"github.com/tanzir71/beacon/internal/sites/indexer"
	"github.com/tanzir71/beacon/internal/ui/dashboard"
)

// ServiceOptions supplies UI-owned state and current application operations.
// Dashboard and OnChange callbacks must be bounded; OnChange must also be safe
// for concurrent calls from the renderer and preference requests.
type ServiceOptions struct {
	IndexPath string
	Dashboard func() dashboard.Options
	OnChange  func()
}

type runningUI struct {
	ctx      context.Context
	view     *SiteView
	mu       sync.Mutex
	requests sync.WaitGroup
	closing  bool
}

// Service is an independently supervised renderer with a stable HTTP handler.
// Only the latest site snapshot is retained; publication never waits on UI I/O.
type Service struct {
	options ServiceOptions
	runMu   sync.Mutex
	latest  atomic.Pointer[indexer.Snapshot]
	changed chan struct{}
	current atomic.Pointer[runningUI]
}

// NewService allocates a service without opening files, listeners or workers.
func NewService(options ServiceOptions) *Service {
	return &Service{options: options, changed: make(chan struct{}, 1)}
}

// Publish implements the Sites snapshot receiver, including while UI is down.
func (service *Service) Publish(snapshot indexer.Snapshot) {
	owned := snapshot.Clone()
	service.latest.Store(&owned)
	service.Refresh()
}

// Refresh requests a render using current options, coalescing pending work.
func (service *Service) Refresh() {
	select {
	case service.changed <- struct{}{}:
	default:
	}
}

// Run owns preference loading, rendering, and all requests of this invocation.
// Renderer errors and panics leave through Run for the app supervisor. Request
// panics retain normal net/http per-request isolation. Shutdown cancels and joins
// admitted requests before another invocation can start.
func (service *Service) Run(ctx context.Context) error {
	if !service.runMu.TryLock() {
		return errors.New("the interface service is already running")
	}
	defer service.runMu.Unlock()
	if ctx.Err() != nil {
		return nil
	}
	view, err := NewSiteView(service.options.IndexPath, service.options.OnChange)
	if err != nil {
		return fmt.Errorf("start interface: %w", err)
	}
	runContext, cancel := context.WithCancel(ctx)
	active := &runningUI{ctx: runContext, view: view}
	defer func() {
		service.current.CompareAndSwap(active, nil)
		active.mu.Lock()
		active.closing = true
		cancel()
		active.mu.Unlock()
		active.requests.Wait()
	}()
	for runContext.Err() == nil {
		select {
		case <-service.changed:
		default:
		}
		var data indexer.Snapshot
		if latest := service.latest.Load(); latest != nil {
			data = *latest
		}
		var options dashboard.Options
		if service.options.Dashboard != nil {
			options = service.options.Dashboard()
		}
		if err := view.Publish(data, options); err != nil {
			return fmt.Errorf("publish interface: %w", err)
		}
		if runContext.Err() != nil {
			return nil
		}
		service.current.Store(active)
		if service.options.OnChange != nil {
			service.options.OnChange()
		}
		select {
		case <-runContext.Done():
			return nil
		case <-service.changed:
		}
	}
	return nil
}

// Ready reports a successfully published view in the current live invocation.
// It is not evidence of network reachability.
func (service *Service) Ready() bool {
	active := service.current.Load()
	return active != nil && active.ctx.Err() == nil
}

// ServeHTTP serves the current invocation without owning an HTTP listener.
func (service *Service) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	active := service.acquire()
	if active == nil {
		http.Error(response, "Beacon's interface is unavailable.", http.StatusServiceUnavailable)
		return
	}
	defer active.requests.Done()
	ctx, cancel := context.WithCancel(request.Context())
	stopServiceCancel := context.AfterFunc(active.ctx, cancel)
	defer stopServiceCancel()
	defer cancel()
	active.view.ServeHTTP(response, request.WithContext(ctx))
}

// Touch records site use in UI-owned state and participates in shutdown joining.
func (service *Service) Touch(slug string) error {
	active := service.acquire()
	if active == nil {
		return errors.New("the interface is unavailable")
	}
	defer active.requests.Done()
	return active.view.Touch(slug)
}

func (service *Service) acquire() *runningUI {
	active := service.current.Load()
	if active == nil {
		return nil
	}
	active.mu.Lock()
	defer active.mu.Unlock()
	if active.closing || active.ctx.Err() != nil {
		return nil
	}
	active.requests.Add(1)
	return active
}
