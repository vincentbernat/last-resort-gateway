package netlink

import (
	"syscall"
	"testing"

	"github.com/vishvananda/netlink"

	"lrg/config"
	"lrg/helpers"
)

func TestMockComponent(t *testing.T) {
	c, inject := NewMock()
	count1 := 0
	count2 := 0
	var last1 Notification
	var last2 Notification
	c.Subscribe("first", func(n Notification) {
		count1++
		last1 = n
	})
	c.Subscribe("second", func(n Notification) {
		count2++
		last2 = n
	})

	expected := Notification{RouteUpdate: &netlink.RouteUpdate{
		Type: syscall.RTM_NEWROUTE,
		Route: netlink.Route{
			LinkIndex: 2,
			Dst:       config.MustParseCIDR("192.168.0.0/16"),
		},
	}}
	inject(expected)
	if count1 != 1 {
		t.Fatalf("Unexpected value for count1 after first route inject (%d, expected 1)",
			count1)
	}
	if count2 != 1 {
		t.Fatalf("Unexpected value for count2 after first route inject (%d, expected 1)",
			count2)
	}
	if diff := helpers.Diff(last1, expected); diff != "" {
		t.Fatalf("Unexpected route update after first route inject (-got, +want):\n%s",
			diff)
	}

	c.Unsubscribe("second")
	expected = Notification{RouteUpdate: &netlink.RouteUpdate{
		Type: syscall.RTM_NEWROUTE,
		Route: netlink.Route{
			LinkIndex: 2,
			Dst:       config.MustParseCIDR("192.168.10.0/24"),
		},
	}}
	inject(expected)
	if count1 != 2 {
		t.Fatalf("Unexpected value for count1 after second route inject (%d, expected 2)",
			count1)
	}
	if count2 != 1 {
		t.Fatalf("Unexpected value for count2 after second route inject (%d, expected 1)",
			count2)
	}
	if diff := helpers.Diff(last1, expected); diff != "" {
		t.Fatalf("Unexpected route update after second route inject (-got, +want):\n%s",
			diff)
	}
}
