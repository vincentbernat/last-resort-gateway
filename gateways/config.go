package gateways

import (
	"github.com/pkg/errors"

	"lrg/config"
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
	Prefix   *config.Prefix
	Protocol *config.Protocol
	Metric   *config.Metric
	Table    *config.Table
}

// LRGToConfiguration is the second half of a last-resort gateway.
type LRGToConfiguration struct {
	Prefix    *config.Prefix
	Protocol  *config.Protocol
	Metric    *config.Metric
	Table     *config.Table
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

// UnmarshalYAML parses the configuration of one gateway
// from YAML.
func (c *LRGConfiguration) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type rawConfiguration LRGConfiguration
	raw := rawConfiguration{}
	if err := unmarshal(&raw); err != nil {
		return errors.Wrap(err, "unable to decode gateway configuration")
	}
	if raw.From.Prefix == nil {
		return errors.New("source prefix missing from configuration")
	}
	switch {
	case raw.To.Prefix == nil:
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
