// Package reporter is a façade for reporting duties in JURA.
//
// Such a façade includes logging, error handling and metrics.
package reporter

import (
	"github.com/getsentry/raven-go"
	log "gopkg.in/inconshreveable/log15.v2"

	"lrg/reporter/logger"
	"lrg/reporter/metrics"
	"lrg/reporter/sentry"
)

// Reporter contains the state for a reporter.
type Reporter struct {
	logger  log.Logger
	sentry  *raven.Client
	metrics *metrics.Metrics
	prefix  string
}

// New creates a new reporter from a configuration.
func New(config Configuration) (*Reporter, error) {
	// Initialize sentry
	s, err := sentry.New(config.Sentry)
	if err != nil {
		return nil, err
	}

	// Initialize logger
	l, err := logger.New(config.Logging, sentryHandler(s), "lrg")
	if err != nil {
		return nil, err
	}

	// Initialize metrics
	m, err := metrics.New(config.Metrics, "lrg")
	if err != nil {
		return nil, err
	}

	return &Reporter{
		logger:  l,
		sentry:  s,
		metrics: m,
		prefix:  "lrg",
	}, nil
}

// Start will start the reporter component
func (r *Reporter) Start() error {
	if r.metrics != nil {
		return r.metrics.Start()
	}
	return nil
}

// Stop will stop reporting and clean the associated resources.
func (r *Reporter) Stop() error {
	if r.sentry != nil {
		r.Info("shutting down Sentry subsystem")
		r.sentry.Wait()
		r.sentry.Close()
	}
	if r.metrics != nil {
		r.Info("shutting down metric subsystem")
		r.metrics.Stop()
	}
	r.Info("stop reporting")
	return nil
}
