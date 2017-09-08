// Package gateways contains the core logic to maintain a set of last
// resort gateways.
package gateways

import (
	"lrg/netlink"
	"lrg/reporter"
)

// Component represents the configuration and state of a set of
// last-resort gateways.
type Component struct {
	r      *reporter.Reporter
	d      *Dependencies
	config Configuration
}

// Dependencies are the dependencies for the gateway component.
type Dependencies struct {
	Netlink netlink.Component
}

// New creates a new gateway component.
func New(reporter *reporter.Reporter, configuration Configuration, dependencies Dependencies) (*Component, error) {
	c := Component{
		r:      reporter,
		d:      &dependencies,
		config: configuration,
	}
	return &c, nil
}
