// +build !release

package daemon

import (
	"time"
)

// MockComponent is a daemon component that does nothing. It doesn't
// need to be started to work.
type MockComponent struct {
	lifecycleComponent
}

// NewMock will create a daemon component that does nothing.
func NewMock() Component {
	return &MockComponent{
		lifecycleComponent: lifecycleComponent{
			terminateChannel: make(chan struct{}),
		},
	}
}

// Start does nothing.
func (c *MockComponent) Start() error {
	return nil
}

// Stop does nothing.
func (c *MockComponent) Stop() error {
	c.Terminate()
	return nil
}

// Ready does nothing.
func (c *MockComponent) Ready() {}

// Watchdog returns a watchdog that never ticks.
func (c *MockComponent) Watchdog() <-chan time.Time {
	return nil
}

// TickWatchdog does nothing
func (c *MockComponent) TickWatchdog() {}
