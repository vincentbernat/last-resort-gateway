// Package netlink handles communication with the kernel (get routes
// and write routes). It doesn't store any data but publishes it with
// callbacks. It is a very thin layer and not meant a complete
// abstraction of the underlying netlink library.
package netlink

import (
	"fmt"
	"strings"
	"syscall"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/pkg/errors"
	"github.com/vishvananda/netlink"
	"gopkg.in/tomb.v2"

	"lrg/reporter"
)

// Component is the interface for the netlink component.
type Component interface {
	Start() error
	Stop() error
	Subscribe(func(Notification))
}

// fsmState represents the current state of the FSM for the netlink component.
type fsmState int

const (
	idle fsmState = iota
	ipv4Routes
	ipv6Routes
	updateRoutes
)

// realComponent is an implementation of the netlink component really
// using Netlink.
type realComponent struct {
	r      *reporter.Reporter
	t      tomb.Tomb
	config Configuration

	// When state == updateRoutes, then updates == liveUpdates
	updates        chan netlink.RouteUpdate
	liveUpdates    chan netlink.RouteUpdate
	state          fsmState
	subscribeError error

	observerSubComponent
}

// New creates a new gateway component.
func New(reporter *reporter.Reporter, configuration Configuration) (Component, error) {
	c := realComponent{
		r:                    reporter,
		config:               configuration,
		observerSubComponent: newObserver(),
	}
	return &c, nil
}

// Start the netlink component.
func (c *realComponent) Start() error {
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
func (c *realComponent) injectRoutes(family int) error {
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
			case c.updates <- update:
				c.r.Counter(fmt.Sprintf("route.initial.%s",
					strings.ToLower(familyStr))).Inc(1)
			}
		}
		c.r.Debug(fmt.Sprintf("all initial routes for family %s sent", familyStr))
		close(c.updates)
		return nil
	})
	return nil
}

// transition change the current state to the next one and execute the
// appropriate actions.
func (c *realComponent) transition() error {
	c.r.Debug(fmt.Sprintf("transition away from state %d", c.state))
	switch c.state {
	case idle, updateRoutes:
		// Start listening to updates right now. Otherwise, we
		// may lose some updates.
		c.liveUpdates = make(chan netlink.RouteUpdate, c.config.ChannelSize)
		c.subscribeError = nil
		if err := netlink.RouteSubscribeWithOptions(c.liveUpdates, c.t.Dying(),
			netlink.RouteSubscribeOptions{
				ErrorCallback: func(err error) {
					c.subscribeError = err
				}}); err != nil {
			return errors.Wrapf(err, "cannot subscribe to route changes")
		}

		c.updates = make(chan netlink.RouteUpdate, c.config.ChannelSize)
		c.notify(Notification{StartOfRIB: true})
		if err := c.injectRoutes(netlink.FAMILY_V4); err != nil {
			return errors.Wrapf(err, "cannot transition from idle state")
		}
		c.state = ipv4Routes
	case ipv4Routes:
		c.updates = make(chan netlink.RouteUpdate, c.config.ChannelSize)
		if err := c.injectRoutes(netlink.FAMILY_V6); err != nil {
			return errors.Wrapf(err, "cannot transition from IPv4 route state")
		}
		c.state = ipv6Routes
	case ipv6Routes:
		c.notify(Notification{EndOfRIB: true})
		c.updates = c.liveUpdates
		c.state = updateRoutes
	default:
		panic("unknown current state")
	}
	return nil
}

func (c *realComponent) run() error {
	c.state = idle

	// Sleep a bit between each difficult transition
	transitionBackoff := backoff.NewExponentialBackOff()
	transitionBackoff.InitialInterval = time.Duration(c.config.BackoffInterval)
	transitionBackoff.Multiplier = 2
	transitionBackoff.MaxInterval = time.Duration(c.config.BackoffMaxInterval)
	transitionBackoff.MaxElapsedTime = 0
	transitionTicker := backoff.NewTicker(transitionBackoff)
	defer transitionTicker.Stop()
	var transitionTick <-chan time.Time

	// Once we didn't get an error since quite some time, we need
	// to reset the above ticker
	var cureTick <-chan time.Time

	for {
		select {
		// Manage delayed transitions
		case <-transitionTick:
			if err := c.transition(); err != nil {
				c.r.Error(err, "cannot change state")
				continue
			}
			transitionTick = nil
		case <-cureTick:
			c.r.Debug("no error since a long time, reset transition ticker")
			cureTick = nil

		// Start the FSM once there is a subscriber
		case <-c.subscribed:
			if err := c.transition(); err != nil {
				c.r.Error(err, "cannot change state")
			} else {
				c.subscribed = nil
			}

		case routeUpdate := <-c.updates:
			if routeUpdate.Table == syscall.RT_TABLE_UNSPEC {
				// Channel has been closed. We need to
				// transition to the next state to
				// recover from this.
				c.updates = nil

				switch c.state {
				case idle, ipv4Routes, ipv6Routes:
					// OK, just transition to next state.
					if err := c.transition(); err != nil {
						c.r.Error(err, "cannot change state")
					} else {
						// Transition now.
						continue
					}

				case updateRoutes:
					// Not totally OK, is it important?
					switch err := c.subscribeError.(type) {
					case syscall.Errno:
						if err == syscall.ENOBUFS {
							// Not important, just log something
							c.r.Info("netlink receive buffer too small",
								"err", err)
							c.r.Counter("error.overflow").Inc(1)
						} else {
							// Important, send an alert, but try to recover
							err := errors.Wrapf(c.subscribeError,
								"fatal error while receiving route updates")
							c.r.Error(err, "")
							c.r.Counter("error.unknown1").Inc(1)
						}
					default:
						// Important too, send an alert, try to recover
						err = errors.Wrapf(err,
							"fatal error while receiving route updates")
						c.r.Error(err, "")
						c.r.Counter("error.unknown2").Inc(1)
					}
				}

				// We still need to trigger a transition, but we'll sleep a bit.
				if cureTick == nil {
					transitionBackoff.Reset()
					cureTick = time.After(time.Duration(c.config.CureInterval))
				}
				c.r.Debug("sleep before next transition",
					"elapsed", transitionBackoff.GetElapsedTime())
				transitionTick = transitionTicker.C
				continue
			}

			c.notify(Notification{RouteUpdate: &routeUpdate})
			c.r.Counter("route.updates").Inc(1)
			c.r.Counter("callback.calls").Inc(1)

		case <-c.t.Dying():
			return nil
		}
	}
}
