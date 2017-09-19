// Package gateways contains the core logic to maintain a set of last
// resort gateways.
package gateways

import (
	"gopkg.in/tomb.v2"

	"lrg/netlink"
	"lrg/reporter"
)

// Component represents the configuration and state of a set of
// last-resort gateways.
type Component struct {
	r      *reporter.Reporter
	d      *Dependencies
	t      tomb.Tomb
	config Configuration

	gateways []gateway
}

// Dependencies are the dependencies for the gateway component.
type Dependencies struct {
	Netlink netlink.Component
}

// New creates a new gateway component.
func New(reporter *reporter.Reporter, configuration Configuration, dependencies Dependencies) (*Component, error) {
	c := Component{
		r:        reporter,
		d:        &dependencies,
		config:   configuration,
		gateways: []gateway{},
	}
	return &c, nil
}

// Start will activate the gateway component. For each last-resort
// gateway, a goroutine will be spawned to handle it.
func (c *Component) Start() error {
	for index, gwConfig := range c.config {
		gw := newGateway(uint(index+1), &gwConfig)
		c.gateways = append(c.gateways, gw)
	}
	c.d.Netlink.Subscribe(func(n netlink.Notification) {
		if c.t.Alive() {
			for _, gw := range c.gateways {
				c.r.Counter("notification.count").Inc(1)
				c.pushNotification(gw, n)
			}
		}
	})
	c.t.Go(func() error {
		for _, gw := range c.gateways {
			c.t.Go(func() error { return c.runGateway(gw) })
		}
		return nil
	})
	return nil
}

// Stop will disable the gateway component by stopping all goroutines.
func (c *Component) Stop() error {
	c.r.Info("shutting down gateway component")
	defer c.r.Info("gateway component stopped")
	c.t.Kill(nil)
	return c.t.Wait()
}
