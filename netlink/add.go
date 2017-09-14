package netlink

import (
	"github.com/pkg/errors"
	"github.com/vishvananda/netlink"
)

// AddRoute will install the specified route. It will replace an
// existing route with the same characteristics. No retry logic is
// attempted, so error must be handled in upper layers.
func (c *realComponent) AddRoute(route netlink.Route) error {
	if err := netlink.RouteReplace(&route); err != nil {
		return errors.Wrapf(err, "cannot install route %s", route)
	}
	return nil
}
