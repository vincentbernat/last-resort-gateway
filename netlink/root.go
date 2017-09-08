// Package netlink handles communication with the kernel (get routes
// and write routes). It doesn't store any data but publishes it with
// callbacks. It is a very thin layer and not meant a complete
// abstraction of the underlying netlink library. Subscription should
// be done before starting the component (otherwise, initial routes
// won't be sent).
package netlink

import (
	"fmt"
	"strings"
	"syscall"

	"github.com/pkg/errors"
	"github.com/vishvananda/netlink"
	"gopkg.in/tomb.v2"

	"lrg/reporter"
)

// Component is the interface for the netlink component.
type Component interface {
	Start() error
	Stop() error
	Subscribe(string, func(netlink.RouteUpdate))
	Unsubscribe(string)
}

// realComponent is an implementation of the netlink component really
// using Netlink.
type realComponent struct {
	r *reporter.Reporter
	t tomb.Tomb

	// Channels for route updates (initial routes, then updates)
	v4routes    chan netlink.RouteUpdate
	v6routes    chan netlink.RouteUpdate
	liveUpdates chan netlink.RouteUpdate

	observerSubComponent
}

// New creates a new gateway component.
func New(reporter *reporter.Reporter) (Component, error) {
	c := realComponent{
		r:                    reporter,
		v4routes:             make(chan netlink.RouteUpdate),
		v6routes:             make(chan netlink.RouteUpdate),
		liveUpdates:          make(chan netlink.RouteUpdate),
		observerSubComponent: newObserver(),
	}
	return &c, nil
}

// Start the netlink component.
func (c *realComponent) Start() error {
	// Get existing routes
	if err := c.injectRoutes(netlink.FAMILY_V4, c.v4routes); err != nil {
		err = errors.Wrapf(err, "cannot get IPv4 routes")
		c.t.Kill(err)
		return err
	}
	if err := c.injectRoutes(netlink.FAMILY_V6, c.v6routes); err != nil {
		err = errors.Wrapf(err, "cannot get IPv6 routes")
		c.t.Kill(err)
		return err
	}
	// Subscribe to new routes
	if err := netlink.RouteSubscribe(c.liveUpdates, c.t.Dying()); err != nil {
		err = errors.Wrapf(err, "cannot subscribe to route changes")
		c.t.Kill(err)
		return err
	}
	c.t.Go(c.run)
	return nil
}

// Stop the netlink component.
func (c *realComponent) Stop() error {
	c.r.Info("shutting down netlink component")
	defer c.r.Info("netlink component stopped")
	c.t.Kill(nil)
	return c.t.Wait()
}

// injectRoutes will inject existing routes into the provided route
// update channel. The channel is closed once routes have been sent.
func (c *realComponent) injectRoutes(family int, updates chan<- netlink.RouteUpdate) error {
	// Get routes from all tables
	routes, err := netlink.RouteListFiltered(family, &netlink.Route{
		Table: syscall.RT_TABLE_UNSPEC,
	}, netlink.RT_FILTER_TABLE)
	if err != nil {
		return err
	}

	// Send the routes into the route update channel. We have to
	// handle the case where we need to stop the component before
	// we were able to send all routes.
	var familyStr string
	switch family {
	case netlink.FAMILY_V4:
		familyStr = "IPv4"
	case netlink.FAMILY_V6:
		familyStr = "IPv6"
	default:
		panic("unknown family for injectRoutes")
	}
	c.t.Go(func() error {
		for _, route := range routes {
			update := netlink.RouteUpdate{
				Type:  syscall.RTM_NEWROUTE,
				Route: route,
			}
			select {
			case <-c.t.Dying():
				c.r.Debug(fmt.Sprintf("component stopped during %s route injection",
					familyStr))
				return nil
			case updates <- update:
				c.r.Counter(fmt.Sprintf("route.initial.%s",
					strings.ToLower(familyStr))).Inc(1)
			}
		}
		c.r.Debug(fmt.Sprintf("all initial routes for family %s sent", familyStr))
		close(updates)
		return nil
	})
	return nil
}

func (c *realComponent) run() error {
	updates := c.v4routes
	for {
		select {
		case routeUpdate := <-updates:
			if routeUpdate.Table == syscall.RT_TABLE_UNSPEC {
				// Channel has been closed, hop to the next one.
				switch updates {
				case c.v4routes:
					c.r.Debug("switch to v6 route updates")
					updates = c.v6routes
					continue
				case c.v6routes:
					c.r.Debug("switch to live route updates")
					updates = c.liveUpdates
					// Keep sending the empty route
				default:
					panic("route updates channel unexpectedly closed")
				}
			}
			count := c.notify(routeUpdate)
			c.r.Counter("route.updates").Inc(1)
			c.r.Counter("callback.calls").Inc(int64(count))
			c.r.Debug("route update received",
				"update", routeUpdate,
				"count", count)

		case <-c.t.Dying():
			return nil
		}
	}
}
