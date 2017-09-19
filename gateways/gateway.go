package gateways

import (
	"fmt"
	"net"
	"syscall"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/pkg/errors"
	knetlink "github.com/vishvananda/netlink"

	"lrg/helpers"
	"lrg/netlink"
)

const (
	// LRGStateMissing when gateway is missing
	LRGStateMissing int64 = iota
	// LRGStateInstalling when gateway is installing
	LRGStateInstalling
	// LRGStateInstalled when gateway is installed
	LRGStateInstalled
)

// gateway is the combination of a last-resort gateway configuration
// with the current state of this gateway.
type gateway struct {
	index  uint
	config *LRGConfiguration
	state  *gatewayState
}

type gatewayState struct {
	notification    chan netlink.Notification
	currentRoute    *knetlink.Route
	candidateRoutes []*knetlink.Route

	// Timer to install and retry installing a route
	installationBackoff *backoff.ExponentialBackOff
	installationTicker  *backoff.Ticker
	installationTick    <-chan time.Time
}

const (
	installFailureInfoDelay    = 5 * time.Second
	installFailureWarningDelay = 30 * time.Second
	installFailureErrorDelay   = 1 * time.Minute
)

// newGateway initializes a last-resort gateway from its
// configuration.
func newGateway(index uint, config *LRGConfiguration) gateway {
	gw := gateway{
		index:  index,
		config: config,
		state: &gatewayState{
			notification: make(chan netlink.Notification, 100),
		},
	}
	return gw
}

// String will turn a gateway to a readable string. Only index, prefix
// and table are used.
func (g gateway) String() string {
	return fmt.Sprintf("gw%d<%s-%s>", g.index, g.config.From.Prefix, g.config.From.Table)
}

// runGateway manages the last-resort gateway given as argument. It
// should be run in a goroutine.
func (c *Component) runGateway(gateway gateway) error {
	c.r.Info(fmt.Sprintf("starting handler for gateway %s", gateway))
	c.r.Counter("count").Inc(1)
	defer c.r.Info(fmt.Sprintf("stopping handler for gateway %s", gateway))
	defer c.r.Counter("count").Dec(1)
	for {
		select {
		case <-c.t.Dying():
			// Component should stop
			if gateway.state.installationTick != nil {
				gateway.state.installationTicker.Stop()
				gateway.state.installationTick = nil
			}
			return nil

		case notification := <-gateway.state.notification:
			// Process an incoming
			// notification. Eventually, this will trigger
			// the next event.
			c.processNotification(&gateway, notification)

		case <-gateway.state.installationTick:
			// We should try to install the current route
			c.r.Debug(fmt.Sprintf("installing route %s", gateway.state.currentRoute),
				"gateway", gateway)
			if err := c.d.Netlink.AddRoute(*gateway.state.currentRoute); err != nil {
				elapsed := gateway.state.installationBackoff.GetElapsedTime()
				if elapsed > installFailureErrorDelay {
					c.r.Error(err, "unable to install route",
						"route", gateway.state.currentRoute,
						"elapsed", elapsed)
				} else {
					alert := c.r.Debug
					if elapsed > installFailureWarningDelay {
						alert = c.r.Warn
					} else if elapsed > installFailureInfoDelay {
						alert = c.r.Info
					}
					alert("unable to install route",
						"route", gateway.state.currentRoute,
						"err", err,
						"elapsed", elapsed)
				}
				c.r.Counter(fmt.Sprintf("gw%d.install.errors", gateway.index)).Inc(1)
				c.r.Counter("install.errors").Inc(1)
				continue
			}
			gateway.state.installationTicker.Stop()
			gateway.state.installationTick = nil
			c.r.Gauge(fmt.Sprintf("gw%d.state", gateway.index)).Update(LRGStateInstalled)
		}
	}
}

// pushNotification forwards a given notification to a gateway to be
// processed.
func (c *Component) pushNotification(gateway gateway, notification netlink.Notification) {
	gateway.state.notification <- notification
}

// processNotification will handle a notification for the given
// gateway.
func (c *Component) processNotification(gateway *gateway, notification netlink.Notification) {
	switch {
	case notification.StartOfRIB:
		c.r.Debug("received start of RIB event", "gateway", gateway)
		gateway.state.candidateRoutes = []*knetlink.Route{}
	case notification.EndOfRIB:
		c.r.Debug("received end of RIB event", "gateway", gateway)
		c.installCandidateRoute(gateway)
	case notification.RouteUpdate != nil:
		c.r.Counter(fmt.Sprintf("gw%d.updates.total", gateway.index)).Inc(1)
		config := gateway.config
		route := &notification.RouteUpdate.Route
		switch {
		case config.To.Match(route):
			c.r.Counter(fmt.Sprintf("gw%d.updates.target", gateway.index)).Inc(1)
			switch notification.RouteUpdate.Type {
			case syscall.RTM_DELROUTE:
				c.r.Debug(fmt.Sprintf("update %s removes current gateway target",
					route), "gateway", gateway)
				gateway.state.currentRoute = nil
				c.installCandidateRoute(gateway)
			case syscall.RTM_NEWROUTE:
				c.r.Debug(fmt.Sprintf("update %s matches current gateway target",
					route), "gateway", gateway)
				gateway.state.currentRoute = route
				c.installCandidateRoute(gateway)
			default:
				c.r.Error(errors.New("unknown route update type received"),
					"",
					"notification", notification,
					"gateway", gateway)
			}
		case config.From.Match(route):
			c.r.Counter(fmt.Sprintf("gw%d.updates.source", gateway.index)).Inc(1)
			// Update the candidates. As we may have to
			// handle the odd IPv6 ECMP routes, we don't
			// try to be smart. To delete a route, we need
			// an exact match. We add a route whatever
			// happens and we apply the appropriate sort
			// algorithm (tos first, then priority).
			switch notification.RouteUpdate.Type {
			case syscall.RTM_DELROUTE:
				c.r.Debug(fmt.Sprintf("update %s deletes a candidate to gateway",
					route), "gateway", gateway)
				c.removeCandidateRoute(gateway, route)
				c.installCandidateRoute(gateway)
			case syscall.RTM_NEWROUTE:
				c.r.Debug(fmt.Sprintf("update %s adds a candidate to gateway",
					route), "gateway", gateway)
				c.addCandidateRoute(gateway, route)
				c.installCandidateRoute(gateway)
			default:
				c.r.Error(errors.New("unknwon route update type received"),
					"",
					"notification", notification,
					"gateway", gateway)
			}
		default:
			c.r.Counter(fmt.Sprintf("gw%d.updates.alien", gateway.index)).Inc(1)
			// Ignore the update
		}
	}
}

// removeCandidateRoute will remove a candidate route from the list of
// candidate routes. The route may not exist. We don't error in this
// case.
func (c *Component) removeCandidateRoute(gateway *gateway, route *knetlink.Route) {
	new := make([]*knetlink.Route, 0, len(gateway.state.candidateRoutes))
	for _, current := range gateway.state.candidateRoutes {
		if !current.Equal(*route) {
			new = append(new, current)
		}
	}
	gateway.state.candidateRoutes = new
	c.r.Debug(fmt.Sprintf("current list of candidates is %v", gateway.state.candidateRoutes),
		"gateway", gateway)
}

// addCandidateRoute will add a candidate route to the list of
// candidate routes. If the route exists, nothing is done. If a route
// has the same table, prefix, tos and priority, it is replaced. When
// using IPv6 ECMP routes, this may be problematic. This needs to be
// carefully tested during integration tests.
func (c *Component) addCandidateRoute(gateway *gateway, route *knetlink.Route) {
	new := make([]*knetlink.Route, 0, len(gateway.state.candidateRoutes))
	for _, current := range gateway.state.candidateRoutes {
		if current.Equal(*route) {
			return
		}
		if helpers.IPNetEqual(*current.Dst, *route.Dst) &&
			current.Table == route.Table &&
			current.Tos == route.Tos &&
			current.Priority == route.Priority {
			c.r.Debug(fmt.Sprintf("replace route %s by %s", current, route),
				"gateway", gateway)
			continue
		}
		new = append(new, current)
	}
	gateway.state.candidateRoutes = append(new, route)
	c.r.Debug(fmt.Sprintf("current list of candidates is %v", gateway.state.candidateRoutes),
		"gateway", gateway)
}

// installCandidateRoute will select the best candidate route (sorting by
// tos, then priority) and will install it.
func (c *Component) installCandidateRoute(gateway *gateway) {
	target := targetRoute(gateway.state.candidateRoutes, &gateway.config.To)
	if target == nil {
		c.r.Debug("no candidates for gateway",
			"gateway", gateway)
		if gateway.state.currentRoute == nil {
			c.r.Gauge(fmt.Sprintf("gw%d.state", gateway.index)).Update(LRGStateMissing)
		}
		return
	}
	if gateway.state.currentRoute != nil && target.Equal(*gateway.state.currentRoute) {
		c.r.Debug("no change for gateway",
			"gateway", gateway)
		return
	}
	c.r.Counter(fmt.Sprintf("gw%d.changes", gateway.index)).Inc(1)
	c.r.Info("last-resort gateway change",
		"from", gateway.state.currentRoute,
		"to", target,
		"gateway", gateway)
	gateway.state.currentRoute = target
	c.installRoute(gateway)
}

// installRoute will trigger route installation for the provided
// gateway. Installation will be retried until it succeeds. It just
// sets a ticker to be used in gateway loop.
func (c *Component) installRoute(gateway *gateway) {
	c.r.Gauge(fmt.Sprintf("gw%d.state", gateway.index)).Update(LRGStateInstalling)
	if gateway.state.installationTick != nil {
		gateway.state.installationTicker.Stop()
	}
	b := backoff.NewExponentialBackOff()
	b.InitialInterval = time.Duration(100 * time.Millisecond)
	b.MaxInterval = 1 * time.Minute
	b.MaxElapsedTime = 0 // Never stops
	b.Multiplier = 2
	gateway.state.installationBackoff = b
	gateway.state.installationTicker = backoff.NewTicker(b)
	gateway.state.installationTick = gateway.state.installationTicker.C
}

// bestCandidateRoute will return the best candidate route (sorting by
// tos, then priority, using the older one in case of equality).
func bestCandidateRoute(candidates []*knetlink.Route) (best *knetlink.Route) {
	for _, current := range candidates {
		if best == nil ||
			best.Tos > current.Tos ||
			(best.Tos == current.Tos && best.Priority > current.Priority) {
			best = current
		}
	}
	return
}

// targetRoute will build the target routes from the configuration and
// the list of candidates. It may return nil if there is no candidate
// and no blackhole route was requested.
func targetRoute(candidates []*knetlink.Route, config *LRGToConfiguration) (target *knetlink.Route) {
	best := bestCandidateRoute(candidates)
	if best == nil {
		if !config.Blackhole {
			return
		}
		target = &knetlink.Route{
			Type: syscall.RTN_BLACKHOLE,
		}
	} else {
		bestCopy := *best
		target = &bestCopy
	}

	// Modify some fields to match configuration
	dst := net.IPNet(config.Prefix)
	target.Dst = &dst
	target.Protocol = int(config.Protocol.ID)
	target.Priority = int(config.Metric)
	target.Table = int(config.Table.ID)

	return
}
