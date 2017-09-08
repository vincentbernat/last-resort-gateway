// Package daemon will handle daemon-related operations: readiness,
// watchdog, exit, reexec...
package daemon

import (
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"lrg/reporter"
)

// Component is the interface the daemon component provides.
type Component interface {
	Start() error
	Stop() error

	// Lifecycle
	Terminated() <-chan struct{}
	Terminate()

	// Watchdog
	Ready()
	Watchdog() <-chan time.Time
	TickWatchdog()
}

// realComponent is a non-mock implementation of the Component
// interface.
type realComponent struct {
	r *reporter.Reporter

	lifecycleComponent

	watchdogTicker *time.Ticker
	watchdogLock   sync.Mutex
}

// New will create a new daemon component.
func New(r *reporter.Reporter) (Component, error) {
	return &realComponent{
		r: r,
		lifecycleComponent: lifecycleComponent{
			terminateChannel: make(chan struct{}),
		},
	}, nil
}

// Start will make the daemon component active.
func (c *realComponent) Start() error {
	// On signal, terminate
	go func() {
		signals := make(chan os.Signal, 1)
		signal.Notify(signals,
			syscall.SIGINT, syscall.SIGTERM,
			syscall.SIGHUP)
		select {
		case s := <-signals:
			c.r.Debug("signal received", "signal", s)
			switch s {
			case syscall.SIGINT, syscall.SIGTERM:
				c.r.Info("quitting...")
				c.Terminate()
				signal.Stop(signals)
			}
		case <-c.Terminated():
			// Do nothing.
		}
	}()

	// Watchdog
	c.initializeWatchdog()
	return nil
}

// Stop will stop the component.
func (c *realComponent) Stop() error {
	c.stopWatchdog()
	c.Terminate()
	return nil
}
