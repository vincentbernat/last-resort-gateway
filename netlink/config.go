package netlink

import (
	"time"

	"github.com/pkg/errors"

	"lrg/config"
)

// Configuration contains the configuration for netlink component.
type Configuration struct {
	SocketSize         uint
	ChannelSize        uint
	BackoffInterval    config.Duration
	BackoffMaxInterval config.Duration
	CureInterval       config.Duration
}

// DefaultConfiguration is the default configuration of the netlink component.
var DefaultConfiguration = Configuration{
	SocketSize:         0,
	ChannelSize:        100,
	BackoffInterval:    config.Duration(10 * time.Millisecond),
	BackoffMaxInterval: config.Duration(10 * time.Second),
	CureInterval:       config.Duration(30 * time.Second),
}

// UnmarshalYAML parses the configuration for the netlink component from
// YAML.
func (c *Configuration) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type rawConfiguration Configuration
	raw := rawConfiguration(DefaultConfiguration)
	if err := unmarshal(&raw); err != nil {
		return errors.Wrap(err, "unable to decode netlink component configuration")
	}

	*c = Configuration(raw)
	return nil
}
