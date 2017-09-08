// Package metrics handles metrics for JURA.
//
// This is a wrapper around Code Hale's Metrics library. It also
// setups the supported exporters.
package metrics

import (
	"sync"
	"time"

	"github.com/rcrowley/go-metrics"
)

const runtimeMetricsInterval = 5 * time.Second

// Metrics represents the internal state of the metric subsystem.
type Metrics struct {
	config   Configuration
	prefix   string
	Registry metrics.Registry

	done chan struct{}  // Channel to close to signal exporters to stop
	wg   sync.WaitGroup // Wait group used to know that exporters have stopped
}

// New creates a new metric registry and setup the appropriate
// exporters. The provided prefix is used for system-wide metrics.
func New(configuration Configuration, prefix string) (*Metrics, error) {
	reg := metrics.NewRegistry()
	m := Metrics{
		config:   configuration,
		prefix:   prefix,
		Registry: reg,
		done:     make(chan struct{}),
	}

	return &m, nil
}

// Start starts the metric collection and the exporters.
func (m *Metrics) Start() error {
	// Register runtime metrics
	runtimeRegistry := metrics.NewPrefixedChildRegistry(m.Registry, "go.")
	metrics.RegisterRuntimeMemStats(runtimeRegistry)
	go func() {
		for {
			timeout := time.After(runtimeMetricsInterval)
			select {
			case <-m.done:
				return
			case <-timeout:
				break
			}
			metrics.CaptureRuntimeMemStatsOnce(runtimeRegistry)
		}
	}()

	// Register exporters
	for _, c := range m.config {
		if err := c.initExporter(m); err != nil {
			m.Stop()
			return err
		}
	}
	return nil
}

// MustStart starts the metric collection and panic if there is an
// error.
func (m *Metrics) MustStart() {
	if err := m.Start(); err != nil {
		panic(err)
	}
}

// Stop stops all exporters and wait for them to terminate.
func (m *Metrics) Stop() error {
	close(m.done)
	m.wg.Wait()
	return nil
}
