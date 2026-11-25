package app

import (
	"context"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestServiceFailureIsContained(t *testing.T) {
	for _, victim := range []string{"sites", "links", "events", "ui"} {
		t.Run(victim, func(t *testing.T) {
			t.Parallel()
			crash := make(chan struct{})
			calls := make(map[string]*atomic.Int64)
			services := make(map[string]Service)
			for _, name := range []string{"sites", "links", "events", "ui"} {
				calls[name] = new(atomic.Int64)
				services[name] = ServiceFunc(func(ctx context.Context) error {
					attempt := calls[name].Add(1)
					if name == victim && attempt == 1 {
						select {
						case <-crash:
							panic("injected service failure")
						case <-ctx.Done():
							return ctx.Err()
						}
					}
					<-ctx.Done()
					return ctx.Err()
				})
			}
			supervisor := testSupervisor(t, services, Options{MinBackoff: 10 * time.Millisecond, MaxBackoff: time.Second})
			startSupervisor(t, supervisor)
			waitFor(t, func() bool {
				for _, count := range calls {
					if count.Load() != 1 {
						return false
					}
				}
				return true
			})
			close(crash)
			waitFor(t, func() bool { return calls[victim].Load() == 2 })
			for name, status := range supervisor.Snapshot() {
				if name == victim {
					if status.Failures != 1 || !strings.Contains(status.LastError, "injected service failure") {
						t.Errorf("panic was not recorded: %#v", status)
					}
				} else if calls[name].Load() != 1 || status.Failures != 0 || status.State != "running" {
					t.Errorf("%s affected by %s panic: calls=%d status=%#v", name, victim, calls[name].Load(), status)
				}
			}
		})
	}
}

func TestServicesCanBeControlledIndependently(t *testing.T) {
	supervisor := testSupervisor(t, healthyServices(), Options{})
	startSupervisor(t, supervisor)
	waitFor(t, func() bool {
		for _, status := range supervisor.Snapshot() {
			if status.Starts != 1 {
				return false
			}
		}
		return true
	})
	if err := supervisor.Stop("sites"); err != nil {
		t.Fatal(err)
	}
	waitFor(t, func() bool { return supervisor.Snapshot()["sites"].State == "stopped" })
	if err := supervisor.Start("sites"); err != nil {
		t.Fatal(err)
	}
	waitFor(t, func() bool { return supervisor.Snapshot()["sites"].Starts == 2 })
	if err := supervisor.Restart("links"); err != nil {
		t.Fatal(err)
	}
	waitFor(t, func() bool { return supervisor.Snapshot()["links"].Starts == 2 })
	if err := supervisor.Start("links"); err != nil {
		t.Fatal(err)
	}
	for name, status := range supervisor.Snapshot() {
		want := uint64(1)
		if name == "sites" || name == "links" {
			want = 2
		}
		if status.Starts != want || status.Failures != 0 {
			t.Errorf("%s lifecycle = %#v, want %d starts and no failures", name, status, want)
		}
	}
	if err := supervisor.Restart("unknown"); err == nil {
		t.Fatal("unknown service accepted")
	}
}

func TestFailuresBackOff(t *testing.T) {
	services := healthyServices()
	attempts := make(chan time.Time, 4)
	services["events"] = ServiceFunc(func(ctx context.Context) error {
		select {
		case attempts <- time.Now():
		case <-ctx.Done():
			return ctx.Err()
		}
		return errors.New("event store unavailable")
	})
	const delay = 40 * time.Millisecond
	supervisor := testSupervisor(t, services, Options{MinBackoff: delay, MaxBackoff: delay})
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- supervisor.Run(ctx) }()
	t.Cleanup(func() {
		cancel()
		select {
		case err := <-done:
			if err != nil {
				t.Error(err)
			}
		case <-time.After(5 * time.Second):
			t.Error("supervisor did not join its services")
		}
	})
	var previous time.Time
	for range 3 {
		select {
		case attempt := <-attempts:
			if !previous.IsZero() && attempt.Sub(previous) < delay {
				t.Errorf("failure retried without backoff: %s", attempt.Sub(previous))
			}
			previous = attempt
		case <-time.After(5 * time.Second):
			t.Fatal("service did not retry")
		}
	}
	waitFor(t, func() bool { return supervisor.Snapshot()["events"].Failures >= 3 })
}

func TestShutdownInterruptsLongBackoff(t *testing.T) {
	services := healthyServices()
	services["links"] = ServiceFunc(func(context.Context) error { return errors.New("link failed") })
	supervisor := testSupervisor(t, services, Options{MinBackoff: time.Hour, MaxBackoff: time.Hour})
	cancel := startSupervisor(t, supervisor)
	waitFor(t, func() bool { return supervisor.Snapshot()["links"].State == "backoff" })
	cancel()
	waitFor(t, func() bool { return supervisor.Snapshot()["links"].State == "stopped" })
	if supervisor.Snapshot()["links"].Starts != 1 {
		t.Fatal("shutdown retried the failed service")
	}
}

func TestSupervisorValidatesServicesAndOptions(t *testing.T) {
	for _, name := range []string{"sites", "links", "events", "ui"} {
		services := healthyServices()
		delete(services, name)
		if _, err := New(services, Options{}); err == nil {
			t.Errorf("accepted missing %s service", name)
		}
		services[name] = nil
		if _, err := New(services, Options{}); err == nil {
			t.Errorf("accepted nil %s service", name)
		}
	}
	for _, options := range []Options{
		{MinBackoff: -time.Second}, {MaxBackoff: -time.Second},
		{MinBackoff: time.Second, MaxBackoff: time.Millisecond},
	} {
		if _, err := New(healthyServices(), options); err == nil {
			t.Errorf("accepted invalid backoff: %#v", options)
		}
	}
	services := healthyServices()
	services["extra"] = services["sites"]
	if _, err := New(services, Options{}); err == nil {
		t.Fatal("accepted an unknown service")
	}
	supervisor := testSupervisor(t, healthyServices(), Options{})
	if err := supervisor.Start("sites"); !errors.Is(err, ErrNotRunning) {
		t.Errorf("control before Run = %v, want ErrNotRunning", err)
	}
	startSupervisor(t, supervisor)
	waitFor(t, func() bool { return supervisor.Snapshot()["ui"].Starts == 1 })
	if err := supervisor.Run(context.Background()); err == nil {
		t.Fatal("accepted a concurrent Run")
	}
}

func TestUnexpectedCleanExitIsRestarted(t *testing.T) {
	services := healthyServices()
	var calls atomic.Int64
	services["ui"] = ServiceFunc(func(ctx context.Context) error {
		if calls.Add(1) == 1 {
			return nil
		}
		<-ctx.Done()
		return ctx.Err()
	})
	supervisor := testSupervisor(t, services, Options{MinBackoff: time.Millisecond})
	services["ui"] = nil
	startSupervisor(t, supervisor)
	waitFor(t, func() bool { return calls.Load() == 2 })
	status := supervisor.Snapshot()["ui"]
	if status.Failures != 1 || status.LastError != "service exited before shutdown" {
		t.Fatalf("clean early exit was not treated as a failure: %#v", status)
	}
	snapshot := supervisor.Snapshot()
	delete(snapshot, "ui")
	if len(supervisor.Snapshot()) != 4 {
		t.Fatal("caller could mutate supervisor state through a snapshot")
	}
}

func TestCanceledRunStartsNothing(t *testing.T) {
	services := healthyServices()
	var calls atomic.Int64
	for name := range services {
		services[name] = ServiceFunc(func(context.Context) error {
			calls.Add(1)
			return nil
		})
	}
	supervisor := testSupervisor(t, services, Options{})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := supervisor.Run(ctx); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 0 {
		t.Fatal("started a service after application shutdown")
	}
}

func TestRestartAndShutdownWaitForServiceCleanup(t *testing.T) {
	services := healthyServices()
	var calls atomic.Int64
	canceled := make(chan int64, 2)
	firstCleanup, secondCleanup := make(chan struct{}), make(chan struct{})
	releaseFirst := sync.OnceFunc(func() { close(firstCleanup) })
	releaseSecond := sync.OnceFunc(func() { close(secondCleanup) })
	services["sites"] = ServiceFunc(func(ctx context.Context) error {
		attempt := calls.Add(1)
		<-ctx.Done()
		canceled <- attempt
		if attempt == 1 {
			<-firstCleanup
		} else {
			<-secondCleanup
		}
		return ctx.Err()
	})
	supervisor := testSupervisor(t, services, Options{})
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		if err := supervisor.Run(ctx); err != nil {
			t.Errorf("run supervisor: %v", err)
		}
	}()
	t.Cleanup(func() {
		releaseFirst()
		releaseSecond()
		cancel()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Error("supervisor did not join cleanup")
		}
	})
	waitFor(t, func() bool { return calls.Load() == 1 })
	if err := supervisor.Restart("sites"); err != nil {
		t.Fatal(err)
	}
	select {
	case attempt := <-canceled:
		if attempt != 1 {
			t.Fatalf("canceled invocation %d, want 1", attempt)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("restart did not cancel the first invocation")
	}
	var controls sync.WaitGroup
	for range 16 {
		controls.Go(func() {
			if err := supervisor.Stop("sites"); err != nil {
				t.Error(err)
			}
			if err := supervisor.Start("sites"); err != nil {
				t.Error(err)
			}
		})
	}
	controls.Wait()
	if calls.Load() != 1 || supervisor.Snapshot()["sites"].Starts != 1 {
		t.Fatal("replacement overlapped the old invocation's cleanup")
	}
	releaseFirst()
	waitFor(t, func() bool { return calls.Load() == 2 })
	cancel()
	select {
	case attempt := <-canceled:
		if attempt != 2 {
			t.Fatalf("canceled invocation %d, want 2", attempt)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("shutdown did not cancel the second invocation")
	}
	select {
	case <-done:
		t.Fatal("application returned before service cleanup finished")
	case <-time.After(20 * time.Millisecond):
	}
	releaseSecond()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("application did not join completed cleanup")
	}
}

func TestBackoffGrowsSaturatesAndResets(t *testing.T) {
	options := Options{MinBackoff: time.Second, MaxBackoff: 8 * time.Second}
	delay := options.MinBackoff
	for _, want := range []time.Duration{time.Second, 2 * time.Second, 4 * time.Second, 8 * time.Second, 8 * time.Second} {
		pause, next := backoffAfterRun(delay, 0, options)
		if pause != want {
			t.Fatalf("retry delay = %s, want %s", pause, want)
		}
		delay = next
	}
	pause, next := backoffAfterRun(delay, options.MaxBackoff, options)
	if pause != time.Second || next != 2*time.Second {
		t.Fatalf("healthy run did not reset backoff: pause=%s next=%s", pause, next)
	}
	options.MaxBackoff = time.Duration(1<<63 - 1)
	pause, next = backoffAfterRun(options.MaxBackoff, 0, options)
	if pause != options.MaxBackoff || next != options.MaxBackoff {
		t.Fatalf("large retry delay overflowed: pause=%s next=%s", pause, next)
	}
}

func healthyServices() map[string]Service {
	services := make(map[string]Service)
	for _, name := range []string{"sites", "links", "events", "ui"} {
		services[name] = ServiceFunc(func(ctx context.Context) error {
			<-ctx.Done()
			return ctx.Err()
		})
	}
	return services
}

func testSupervisor(t *testing.T, services map[string]Service, options Options) *Supervisor {
	t.Helper()
	supervisor, err := New(services, options)
	if err != nil {
		t.Fatal(err)
	}
	return supervisor
}

func startSupervisor(t *testing.T, supervisor *Supervisor) context.CancelFunc {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- supervisor.Run(ctx) }()
	t.Cleanup(func() {
		cancel()
		select {
		case err := <-done:
			if err != nil {
				t.Errorf("run supervisor: %v", err)
			}
		case <-time.After(5 * time.Second):
			t.Error("supervisor did not join its services")
		}
	})
	return cancel
}

func waitFor(t *testing.T, condition func() bool) {
	t.Helper()
	deadline := time.NewTimer(5 * time.Second)
	defer deadline.Stop()
	ticker := time.NewTicker(time.Millisecond)
	defer ticker.Stop()
	for !condition() {
		select {
		case <-ticker.C:
		case <-deadline.C:
			t.Fatal("service state did not settle")
		}
	}
}
