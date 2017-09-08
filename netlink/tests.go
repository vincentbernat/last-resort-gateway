// +build !release

package netlink

import (
	"github.com/vishvananda/netlink"
)

type mockComponent struct {
	observerSubComponent
}

// NewMock creates a new mock component for netlink component. This
// component does nothing on its own. It also provides a function to
// inject route updates and will just broadcast them to all
// subscribers.
func NewMock() (Component, func(netlink.RouteUpdate)) {
	c := &mockComponent{
		observerSubComponent: newObserver(),
	}
	return c, c.inject
}

// Start does nothing.
func (c *mockComponent) Start() error {
	return nil
}

// Stop does nothing.
func (c *mockComponent) Stop() error {
	return nil
}

// inject will inject a route update into the component. It will just
// be broadcasted to all subscribers.
func (c *mockComponent) inject(u netlink.RouteUpdate) {
	c.notify(u)
}
