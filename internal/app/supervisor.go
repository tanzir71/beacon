// Package app owns the lifecycle of Beacon's four independent services.
package app

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

// ErrNotRunning rejects lifecycle commands outside the application's lifetime.
var ErrNotRunning = errors.New("the application supervisor is not running")

// Service owns its resources until Run returns. It must honor cancellation and
// join its workers before returning; worker failures must be returned to Run,
// not raised as panics in unsupervised goroutines.
type Service interface {
	Run(context.Context) error
}

// ServiceFunc adapts a context-owned service entry point to Service.
type ServiceFunc func(context.Context) error

// Run executes the service entry point in its supervisor-owned goroutine.
func (run ServiceFunc) Run(ctx context.Context) error { return run(ctx) }

// Options bounds exponential retry delays; zero values select the defaults.
type Options struct {
	MinBackoff time.Duration
	MaxBackoff time.Duration
}

// Status describes lifecycle facts, not network reachability or readiness.
type Status struct {
	Enabled   bool
	State     string
	Starts    uint64
	Failures  uint64
	LastError string
}

type worker struct {
	service Service
	status  Status
	cancel  context.CancelFunc
	wake    chan struct{}
}

// Supervisor manages four peers with independent cancellation and retry state.
type Supervisor struct {
	mu      sync.Mutex
	workers map[string]*worker
	options Options
	ctx     context.Context
}

// New requires exactly the four service areas and takes its own service map.
func New(services map[string]Service, options Options) (*Supervisor, error) {
	if options.MinBackoff < 0 || options.MaxBackoff < 0 {
		return nil, errors.New("service backoff cannot be negative")
	}
	if options.MinBackoff == 0 {
		options.MinBackoff = time.Second
	}
	if options.MaxBackoff == 0 {
		options.MaxBackoff = 30 * time.Second
	}
	if options.MaxBackoff < options.MinBackoff {
		return nil, errors.New("maximum service backoff is below its minimum")
	}
	if len(services) != 4 {
		return nil, errors.New("exactly sites, links, events and ui services are required")
	}
	supervisor := &Supervisor{workers: make(map[string]*worker, 4), options: options}
	for _, name := range []string{"sites", "links", "events", "ui"} {
		service, found := services[name]
		if !found || service == nil {
			return nil, fmt.Errorf("the %s service is required", name)
		}
		supervisor.workers[name] = &worker{
			service: service,
			status:  Status{Enabled: true, State: "stopped"},
			wake:    make(chan struct{}, 1),
		}
	}
	return supervisor, nil
}

// Run starts all enabled services and joins them when the parent is canceled.
// A service failure never cancels a sibling's context.
func (supervisor *Supervisor) Run(ctx context.Context) error {
	supervisor.mu.Lock()
	if supervisor.ctx != nil {
		supervisor.mu.Unlock()
		return errors.New("the application supervisor is already running")
	}
	supervisor.ctx = ctx
	var workers sync.WaitGroup
	for _, worker := range supervisor.workers {
		workers.Go(func() { supervisor.runWorker(ctx, worker) })
	}
	supervisor.mu.Unlock()
	workers.Wait()
	supervisor.mu.Lock()
	supervisor.ctx = nil
	supervisor.mu.Unlock()
	return nil
}

// Start enables a stopped service and is idempotent while it is enabled.
func (supervisor *Supervisor) Start(name string) error {
	return supervisor.control(name, true, false)
}

// Stop requests cancellation without stopping sibling services.
func (supervisor *Supervisor) Stop(name string) error {
	return supervisor.control(name, false, false)
}

// Restart requests a replacement after the old invocation returns.
func (supervisor *Supervisor) Restart(name string) error {
	return supervisor.control(name, true, true)
}

func (supervisor *Supervisor) control(name string, enabled, restart bool) error {
	supervisor.mu.Lock()
	defer supervisor.mu.Unlock()
	if supervisor.ctx == nil || supervisor.ctx.Err() != nil {
		return ErrNotRunning
	}
	worker, found := supervisor.workers[name]
	if !found {
		return fmt.Errorf("unknown service %q", name)
	}
	if worker.status.Enabled == enabled && !restart {
		return nil
	}
	worker.status.Enabled = enabled
	if worker.cancel != nil {
		worker.status.State = "stopping"
		worker.cancel()
	}
	select {
	case worker.wake <- struct{}{}:
	default:
	}
	return nil
}

// Snapshot returns a copy of every service's lifecycle facts.
func (supervisor *Supervisor) Snapshot() map[string]Status {
	supervisor.mu.Lock()
	defer supervisor.mu.Unlock()
	statuses := make(map[string]Status, len(supervisor.workers))
	for name, worker := range supervisor.workers {
		statuses[name] = worker.status
	}
	return statuses
}

func (supervisor *Supervisor) runWorker(ctx context.Context, worker *worker) {
	defer func() {
		supervisor.mu.Lock()
		worker.status.State = "stopped"
		worker.cancel = nil
		supervisor.mu.Unlock()
	}()
	delay := supervisor.options.MinBackoff
	for ctx.Err() == nil {
		supervisor.mu.Lock()
		if !worker.status.Enabled {
			worker.status.State = "stopped"
			supervisor.mu.Unlock()
			select {
			case <-ctx.Done():
			case <-worker.wake:
			}
			continue
		}
		// Commands already reflected in Enabled must not bypass a future backoff.
		select {
		case <-worker.wake:
		default:
		}
		runContext, cancel := context.WithCancel(ctx)
		worker.cancel = cancel
		worker.status.State = "running"
		worker.status.Starts++
		supervisor.mu.Unlock()

		started := time.Now()
		err := runService(runContext, worker.service)
		interrupted := runContext.Err() != nil
		cancel()
		supervisor.mu.Lock()
		worker.cancel = nil
		if interrupted {
			worker.status.State = "stopped"
			supervisor.mu.Unlock()
			delay = supervisor.options.MinBackoff
			continue
		}
		worker.status.Failures++
		worker.status.LastError = err.Error()
		worker.status.State = "backoff"
		supervisor.mu.Unlock()

		pause, next := backoffAfterRun(delay, time.Since(started), supervisor.options)
		delay = next
		timer := time.NewTimer(pause)
		select {
		case <-ctx.Done():
		case <-worker.wake:
		case <-timer.C:
		}
		timer.Stop()
	}
}

func backoffAfterRun(delay, runFor time.Duration, options Options) (time.Duration, time.Duration) {
	if runFor >= options.MaxBackoff {
		delay = options.MinBackoff
	}
	next := options.MaxBackoff
	if delay <= options.MaxBackoff/2 {
		next = delay * 2
	}
	return delay, next
}

func runService(ctx context.Context, service Service) (err error) {
	defer func() {
		if failure := recover(); failure != nil {
			err = fmt.Errorf("service panicked: %v", failure)
		}
	}()
	if err = service.Run(ctx); err == nil {
		return errors.New("service exited before shutdown")
	}
	return err
}
