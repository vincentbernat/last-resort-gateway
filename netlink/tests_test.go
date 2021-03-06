package netlink

import (
	"syscall"
	"testing"

	"github.com/vishvananda/netlink"

	"lrg/config"
	"lrg/helpers"
)

func TestMockComponent(t *testing.T) {
	routes := []netlink.Route{}
	c, inject := NewMock(func(route netlink.Route) error {
		routes = append(routes, route)
		return nil
	})
	count := 0
	var last Notification
	c.Subscribe(func(n Notification) {
		count++
		last = n
	})

	expected := Notification{RouteUpdate: &netlink.RouteUpdate{
		Type: syscall.RTM_NEWROUTE,
		Route: netlink.Route{
			LinkIndex: 2,
			Dst:       config.MustParseCIDR("192.168.0.0/16"),
		},
	}}
	inject(expected)
	if count != 1 {
		t.Fatalf("Unexpected value for count after first route inject (%d, expected 1)",
			count)
	}
	if diff := helpers.Diff(last, expected); diff != "" {
		t.Fatalf("Unexpected route update after first route inject (-got, +want):\n%s",
			diff)
	}

	expected = Notification{RouteUpdate: &netlink.RouteUpdate{
		Type: syscall.RTM_NEWROUTE,
		Route: netlink.Route{
			LinkIndex: 2,
			Dst:       config.MustParseCIDR("192.168.10.0/24"),
		},
	}}
	inject(expected)
	if count != 2 {
		t.Fatalf("Unexpected value for count after second route inject (%d, expected 2)",
			count)
	}
	if diff := helpers.Diff(last, expected); diff != "" {
		t.Fatalf("Unexpected route update after second route inject (-got, +want):\n%s",
			diff)
	}

	if err := c.AddRoute(netlink.Route{
		LinkIndex: 2,
		Dst:       config.MustParseCIDR("192.168.0.0/16"),
	}); err != nil {
		t.Fatalf("AddRoute() error:\n%+v", err)
	}
	if diff := helpers.Diff(routes, []netlink.Route{
		netlink.Route{
			LinkIndex: 2,
			Dst:       config.MustParseCIDR("192.168.0.0/16"),
		}}); diff != "" {
		t.Fatalf("AddRoute() (-got, +want):\n%s", diff)
	}

}
