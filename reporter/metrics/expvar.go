package metrics

import (
	"net"
	"net/http"
	"time"

	"github.com/pkg/errors"
	"github.com/rcrowley/go-metrics/exp"
	"gopkg.in/tylerb/graceful.v1"

	"lrg/config"
)

// ExpvarConfiguration is the configuration for exporting metrics to
// expvar.
type ExpvarConfiguration struct {
	Listen config.Addr
}

// UnmarshalYAML parses a configuration from YAML.
func (c *ExpvarConfiguration) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type rawExpvarConfiguration ExpvarConfiguration
	var raw rawExpvarConfiguration
	if err := unmarshal(&raw); err != nil {
		return errors.Wrap(err, "unable to decode expvar configuration")
	}
	if raw.Listen == "" {
		return errors.Errorf("missing listen value")
	}
	*c = ExpvarConfiguration(raw)
	return nil
}

// Initialization
func (c *ExpvarConfiguration) initExporter(metrics *Metrics) error {
	// Setup the muxer
	mux := http.NewServeMux()
	handler := exp.ExpHandler(metrics.Registry)
	mux.Handle("/", handler)

	// Run the HTTP server
	address := c.Listen
	server := &graceful.Server{
		Timeout:          10 * time.Second,
		NoSignalHandling: true,
		Server: &http.Server{
			Addr:    address.String(),
			Handler: mux,
		},
	}
	listener, err := net.Listen("tcp", address.String())
	if err != nil {
		return errors.Wrapf(err, "unable to listen to %v", address)
	}
	go server.Serve(listener)

	// Handle stop correctly
	metrics.wg.Add(1)
	go func() {
		<-metrics.done
		server.Stop(1 * time.Second)
		<-server.StopChan()
		metrics.wg.Done()
	}()

	return nil
}
