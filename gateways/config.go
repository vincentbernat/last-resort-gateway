package gateways

import (
	"net"

	"github.com/pkg/errors"
	"github.com/vishvananda/netlink"

	"lrg/config"
	"lrg/helpers"
)

// Configuration contains the configuration for the last resort
// gateways. This is a slice of last resort gateways.
type Configuration []LRGConfiguration

// LRGConfiguration represents the configuration for one last resort
// gateway.
type LRGConfiguration struct {
	From LRGFromConfiguration
	To   LRGToConfiguration
}

// LRGFromConfiguration is the first half of a last-resort gateway.
type LRGFromConfiguration struct {
	Prefix   config.Prefix
	Protocol *config.Protocol
	Metric   *config.Metric
	Table    config.Table
}

// LRGToConfiguration is the second half of a last-resort gateway.
type LRGToConfiguration struct {
	Prefix    config.Prefix
	Protocol  config.Protocol
	Metric    config.Metric
	Table     config.Table
	Blackhole bool
}

// UnmarshalYAML parses the configuration of the gateway component
// from YAML.
func (c *Configuration) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type rawConfiguration Configuration
	raw := rawConfiguration{}
	if err := unmarshal(&raw); err != nil {
		return errors.Wrap(err, "unable to decode gateways configuration")
	}
	if len(raw) == 0 {
		return errors.New("at least one gateway is needed")
	}
	*c = Configuration(raw)
	return nil
}

var (
	// DefaultToMetric is the default metric for copied route
	DefaultToMetric = config.Metric(4294967295)
	// DefaultToProtocol is the default protocol for copied route
	DefaultToProtocol = config.Protocol{ID: 254, Name: "lrg"}
	// DefaultTable is the default table
	DefaultTable = config.Table{ID: 254, Name: "main"}
)

// UnmarshalYAML parses the configuration of one gateway
// from YAML.
func (c *LRGConfiguration) UnmarshalYAML(unmarshal func(interface{}) error) error {
	ipPlaceholder := config.Prefix{
		IP:   net.ParseIP("ffff:ffff:ffff:ffff:ffff:ffff:ffff:ffff"),
		Mask: net.CIDRMask(128, 128),
	}
	type rawConfiguration LRGConfiguration
	raw := rawConfiguration{
		From: LRGFromConfiguration{
			Prefix: ipPlaceholder,
			Table:  DefaultTable,
		},
		To: LRGToConfiguration{
			// Prefix: copied over
			Protocol: DefaultToProtocol,
			Metric:   DefaultToMetric,
			// Table: copied over
			Blackhole: false,
		},
	}
	if err := unmarshal(&raw); err != nil {
		return errors.Wrap(err, "unable to decode gateway configuration")
	}

	// Copy values from From to To and decode again
	raw.To.Prefix = raw.From.Prefix
	raw.To.Table = raw.From.Table
	if err := unmarshal(&raw); err != nil {
		return errors.Wrap(err, "unable to decode gateway configuration")
	}

	// Check compatibility errors
	switch {
	case raw.From.Prefix.IP.Equal(ipPlaceholder.IP):
		return errors.New("source prefix missing from configuration")
	case raw.From.Prefix.IP.To4() == nil && raw.To.Prefix.IP.To4() != nil:
		return errors.Errorf("incompatible families for from/to prefixes (%s/%s)",
			raw.From.Prefix, raw.To.Prefix)
	case raw.From.Prefix.IP.To4() != nil && raw.To.Prefix.IP.To4() == nil:
		return errors.Errorf("incompatible families for from/to prefixes (%s/%s)",
			raw.From.Prefix, raw.To.Prefix)
	}
	*c = LRGConfiguration(raw)
	return nil
}

// Match will tell if a "from" configuration matches the given route.
func (c *LRGFromConfiguration) Match(route *netlink.Route) bool {
	return route.Dst != nil &&
		helpers.IPNetEqual(net.IPNet(c.Prefix), *route.Dst) &&
		(c.Protocol == nil || c.Protocol.ID == uint(route.Protocol)) &&
		(c.Metric == nil || uint(*c.Metric) == uint(route.Priority)) &&
		c.Table.ID == uint(route.Table)
}

// Match will tel if a "to" configuration matches the given route.
func (c *LRGToConfiguration) Match(route *netlink.Route) bool {
	return route.Dst != nil &&
		helpers.IPNetEqual(net.IPNet(c.Prefix), *route.Dst) &&
		c.Protocol.ID == uint(route.Protocol) &&
		uint(c.Metric) == uint(route.Priority) &&
		c.Table.ID == uint(route.Table)
}
