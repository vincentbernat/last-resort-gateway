package cmd

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/mattn/go-isatty"
	"github.com/pkg/errors"
	"gopkg.in/urfave/cli.v1"
	"gopkg.in/yaml.v2"

	"lrg/daemon"
	"lrg/gateways"
	"lrg/netlink"
	"lrg/reporter"
)

// The daemon configuration is split in several parts. Each part
// refers to the specified subsystem.
type daemonConfiguration struct {
	Reporting reporter.Configuration
	Gateways  gateways.Configuration
	Netlink   netlink.Configuration
}

// Parse a configuration from YAML.
func (configuration *daemonConfiguration) UnmarshalYAML(unmarshal func(interface{}) error) error {
	// If a section is entirely absent, it won't get its default
	// value. We workaround that by unmarshalling an empty object
	// for each valid key.
	type rawDaemonConfiguration daemonConfiguration
	var raw rawDaemonConfiguration
	err := yaml.Unmarshal([]byte(`
reporting: {}
netlink: {}
`), &raw)
	if err != nil {
		return errors.Wrap(err, "unable to decode default daemon configuration")
	}
	if err := unmarshal(&raw); err != nil {
		return errors.Wrap(err, "unable to decode daemon configuration")
	}
	if len(raw.Gateways) == 0 {
		return errors.New("at least one gateway is needed")
	}
	*configuration = daemonConfiguration(raw)
	return nil
}

var daemonCmd = cli.Command{
	Name:  "daemon",
	Usage: "Start Last-Resort Gateway",
	Description: `Start Last-Resort Gateway as a daemon.

A configuration file must be provided.`,
	ArgsUsage: "FILE",
	Flags: []cli.Flag{
		cli.BoolFlag{
			Name:  "check",
			Usage: "check configuration syntax and exit",
		},
	},
	Action: func(c *cli.Context) error {
		if c.NArg() != 1 {
			return UsageError("a configuration file is required")
		}

		// Unmarshal configuration
		configFileName := c.Args().First()
		configFile, err := ioutil.ReadFile(configFileName)
		if err != nil {
			return errors.Wrap(err,
				fmt.Sprintf("unable to read configuration file %v",
					configFileName))
		}

		var config daemonConfiguration
		if err := yaml.Unmarshal([]byte("{}"), &config); err != nil {
			return errors.Wrap(err, "unable to get default configuration")
		}
		if err := yaml.Unmarshal(configFile, &config); err != nil {
			return errors.Wrap(err,
				fmt.Sprintf("unable to parse configuration file %v",
					configFileName))
		}

		// If we are on a TTY, don't log to syslog, log to console.
		if isatty.IsTerminal(os.Stderr.Fd()) {
			config.Reporting.Logging.Console = true
			config.Reporting.Logging.Syslog = false
		}

		// reporting
		r, err := reporter.New(config.Reporting)
		if err != nil {
			return errors.Wrap(err, "unable to initialize reporting component")
		}
		log.SetOutput(r)

		// daemon
		daemonComponent, err := daemon.New(r)
		if err != nil {
			return errors.Wrap(err, "unable to initialize daemon component")
		}

		// netlink
		netlinkComponent, err := netlink.New(r, config.Netlink)
		if err != nil {
			return errors.Wrap(err, "unable to initialize netlink component")
		}

		// gateways
		gatewayComponent, err := gateways.New(r, config.Gateways, gateways.Dependencies{
			Netlink: netlinkComponent,
		})
		if err != nil {
			return errors.Wrap(err, "unable to initialize gateway component")
		}

		// If we only asked for a check, stop here.
		if c.Bool("check") {
			return nil
		}

		// Start all the components.
		components := []interface{}{
			r,
			daemonComponent,
			netlinkComponent,
			gatewayComponent,
		}
		startedComponents := []interface{}{}
		defer func() {
			for _, cmp := range startedComponents {
				if stopperC, ok := cmp.(stopper); ok {
					if err := stopperC.Stop(); err != nil {
						r.Error(err, "unable to stop component, ignoring")
					}
				}
			}
		}()
		for _, cmp := range components {
			if starterC, ok := cmp.(starter); ok {
				if err := starterC.Start(); err != nil {
					return errors.Wrap(err, "unable to start component")
				}
			}
			startedComponents = append([]interface{}{cmp}, startedComponents...)
		}

		r.Info("Last-Resort Gateway has started...",
			"version", Version,
			"build-date", BuildDate)

		select {
		case <-daemonComponent.Terminated():
			r.Info("stopping all components...")
		}

		return nil
	},
}

type starter interface {
	Start() error
}
type stopper interface {
	Stop() error
}
