package netlink

import (
	"bytes"
	"fmt"
	"net"
	"os/exec"
	"syscall"
	"testing"
	"time"

	"github.com/vishvananda/netlink"

	"lrg/config"
	"lrg/helpers"
	"lrg/reporter"
)

func TestFastStartStop(t *testing.T) {
	r := reporter.NewMock()
	c, err := New(r, DefaultConfiguration)
	if err != nil {
		t.Fatalf("New() error:\n%+v", err)
	}

	// Setup observer
	done := make(chan struct{})
	c.Subscribe(func(_ Notification) {
		<-done
	})

	// Start/stop component
	if err := c.Start(); err != nil {
		t.Fatalf("Start() error:\n%+v", err)
	}
	go func() {
		<-c.(*realComponent).t.Dying()
		close(done)
	}()
	if err := c.Stop(); err != nil {
		t.Fatalf("Stop() error:\n%+v", err)
	}
	if counter := r.Counter("callback.calls").Snapshot().Count(); counter > 2 {
		t.Errorf("callback was called too many times (%d)", counter)
	}
}

func TestObserveRoutes(t *testing.T) {
	r := reporter.NewMock()
	c, err := New(r, DefaultConfiguration)
	if err != nil {
		t.Fatalf("New() error:\n%+v", err)
	}

	// Setup observer
	var got []*netlink.RouteUpdate
	done := make(chan struct{})
	c.Subscribe(func(notification Notification) {
		if notification.StartOfRIB {
			got = []*netlink.RouteUpdate{}
			return
		}
		if notification.EndOfRIB {
			close(done)
			return
		}
		u := notification.RouteUpdate
		if u.Table == syscall.RT_TABLE_LOCAL {
			return
		}
		got = append(got, u)
	})

	// Add some initial routes
	setup := `
[ -d /sys/class/net/dummy0 ] || ip link add name dummy0 type dummy || true
ip route flush table main
ip -6 route flush table main
ip link set up dev dummy0
sysctl -w net.ipv6.conf.dummy0.optimistic_dad=1
sysctl -w net.ipv6.conf.dummy0.disable_ipv6=0
ip route add 192.168.24.0/24 dev dummy0
ip route add 192.168.25.0/24 via 192.168.24.1
ip route add 2001:db8:24::/64 dev dummy0
ip route add 2001:db8:25::/64 via 2001:db8:24::1
ip route add 192.168.26.0/24 dev dummy0 table 100
`
	var outbuf, errbuf bytes.Buffer
	cmd := exec.Command("sh", "-exc", setup)
	cmd.Stdout = &outbuf
	cmd.Stderr = &errbuf
	if err := cmd.Run(); err != nil {
		t.Fatalf("Unable to setup routes\n** Setup:\n%s\n** Stdout:\n%s\n** Stderr:\n%s\n** Error:\n%+v",
			setup, outbuf.String(), errbuf.String(), err)
	}

	// Start component
	if err := c.Start(); err != nil {
		t.Fatalf("Start() error:\n%+v", err)
	}
	defer func() {
		if err := c.Stop(); err != nil {
			t.Fatalf("Stop() error:\n%+v", err)
		}
	}()

	// Check we get initial routes. We rely heavily on default
	// kernel order (order by table, then by prefix). The
	// component will also order IPv4 routes before IPv6 routes.
	<-done
	expected := []netlink.RouteUpdate{
		{
			Type: syscall.RTM_NEWROUTE,
			Route: netlink.Route{
				LinkIndex: 2,
				Dst:       config.MustParseCIDR("192.168.26.0/24"),
				Table:     100,
			},
		}, {
			Type: syscall.RTM_NEWROUTE,
			Route: netlink.Route{
				LinkIndex: 2,
				Dst:       config.MustParseCIDR("192.168.24.0/24"),
				Table:     syscall.RT_TABLE_MAIN,
			},
		}, {
			Type: syscall.RTM_NEWROUTE,
			Route: netlink.Route{
				LinkIndex: 2,
				Dst:       config.MustParseCIDR("192.168.25.0/24"),
				Gw:        net.ParseIP("192.168.24.1"),
				Table:     syscall.RT_TABLE_MAIN,
			},
		}, {
			Type: syscall.RTM_NEWROUTE,
			Route: netlink.Route{
				LinkIndex: 2,
				Dst:       config.MustParseCIDR("2001:db8:24::/64"),
				Table:     syscall.RT_TABLE_MAIN,
			},
		}, {
			Type: syscall.RTM_NEWROUTE,
			Route: netlink.Route{
				LinkIndex: 2,
				Dst:       config.MustParseCIDR("2001:db8:25::/64"),
				Gw:        net.ParseIP("2001:db8:24::1"),
				Table:     syscall.RT_TABLE_MAIN,
			},
		}, {
			Type: syscall.RTM_NEWROUTE,
			Route: netlink.Route{
				LinkIndex: 2,
				Dst:       config.MustParseCIDR("fe80::/64"),
				Table:     syscall.RT_TABLE_MAIN,
			},
		},
	}

	if diff := helpers.Diff(got, expected); diff != "" {
		t.Fatalf("initial routes received (-got, +want):\n%s", diff)
	}

	// Send some additional routes
	cases := []struct {
		setup    string
		expected []netlink.RouteUpdate
	}{
		{
			setup: "add 192.168.27.0/24 dev dummy0",
			expected: []netlink.RouteUpdate{
				{
					Type: syscall.RTM_NEWROUTE,
					Route: netlink.Route{
						LinkIndex: 2,
						Dst:       config.MustParseCIDR("192.168.27.0/24"),
						Table:     syscall.RT_TABLE_MAIN,
					},
				},
			},
		}, {
			setup: "add 192.168.28.0/24 dev dummy0 table 90",
			expected: []netlink.RouteUpdate{
				{
					Type: syscall.RTM_NEWROUTE,
					Route: netlink.Route{
						LinkIndex: 2,
						Dst:       config.MustParseCIDR("192.168.28.0/24"),
						Table:     90,
					},
				},
			},
		}, {
			setup: "del 192.168.27.0/24 dev dummy0",
			expected: []netlink.RouteUpdate{
				{
					Type: syscall.RTM_DELROUTE,
					Route: netlink.Route{
						LinkIndex: 2,
						Dst:       config.MustParseCIDR("192.168.27.0/24"),
						Table:     syscall.RT_TABLE_MAIN,
					},
				},
			},
		}, {
			setup: "del 192.168.28.0/24 dev dummy0 table 90",
			expected: []netlink.RouteUpdate{
				{
					Type: syscall.RTM_DELROUTE,
					Route: netlink.Route{
						LinkIndex: 2,
						Dst:       config.MustParseCIDR("192.168.28.0/24"),
						Table:     90,
					},
				},
			},
		}, {
			setup: "add 2001:db8:27::/64 dev dummy0",
			expected: []netlink.RouteUpdate{
				{
					Type: syscall.RTM_NEWROUTE,
					Route: netlink.Route{
						LinkIndex: 2,
						Dst:       config.MustParseCIDR("2001:db8:27::/64"),
						Table:     syscall.RT_TABLE_MAIN,
					},
				},
			},
		}, {
			setup: "add 2001:db8:28::/64 dev dummy0 table 90",
			expected: []netlink.RouteUpdate{
				{
					Type: syscall.RTM_NEWROUTE,
					Route: netlink.Route{
						LinkIndex: 2,
						Dst:       config.MustParseCIDR("2001:db8:28::/64"),
						Table:     90,
					},
				},
			},
		}, {
			setup: "del 2001:db8:27::/64 dev dummy0",
			expected: []netlink.RouteUpdate{
				{
					Type: syscall.RTM_DELROUTE,
					Route: netlink.Route{
						LinkIndex: 2,
						Dst:       config.MustParseCIDR("2001:db8:27::/64"),
						Table:     syscall.RT_TABLE_MAIN,
					},
				},
			},
		}, {
			setup: "del 2001:db8:28::/64 dev dummy0 table 90",
			expected: []netlink.RouteUpdate{
				{
					Type: syscall.RTM_DELROUTE,
					Route: netlink.Route{
						LinkIndex: 2,
						Dst:       config.MustParseCIDR("2001:db8:28::/64"),
						Table:     90,
					},
				},
			},
		}, {
			setup: "add 192.168.30.0/24 via 192.168.24.1",
			expected: []netlink.RouteUpdate{
				{
					Type: syscall.RTM_NEWROUTE,
					Route: netlink.Route{
						LinkIndex: 2,
						Dst:       config.MustParseCIDR("192.168.30.0/24"),
						Gw:        net.ParseIP("192.168.24.1"),
						Table:     syscall.RT_TABLE_MAIN,
					},
				},
			},
		}, {
			setup: "add 2001:db8:30::/64 via 2001:db8:24::1",
			expected: []netlink.RouteUpdate{
				{
					Type: syscall.RTM_NEWROUTE,
					Route: netlink.Route{
						LinkIndex: 2,
						Dst:       config.MustParseCIDR("2001:db8:30::/64"),
						Gw:        net.ParseIP("2001:db8:24::1"),
						Table:     syscall.RT_TABLE_MAIN,
					},
				},
			},
		}, {
			setup: "add 192.168.31.0/24 via 192.168.24.1 proto kernel",
			expected: []netlink.RouteUpdate{
				{
					Type: syscall.RTM_NEWROUTE,
					Route: netlink.Route{
						LinkIndex: 2,
						Dst:       config.MustParseCIDR("192.168.31.0/24"),
						Gw:        net.ParseIP("192.168.24.1"),
						Table:     syscall.RT_TABLE_MAIN,
						Protocol:  2,
					},
				},
			},
		}, {
			setup: "add 192.168.31.0/24 via 192.168.24.1 metric 200",
			expected: []netlink.RouteUpdate{
				{
					Type: syscall.RTM_NEWROUTE,
					Route: netlink.Route{
						LinkIndex: 2,
						Dst:       config.MustParseCIDR("192.168.31.0/24"),
						Gw:        net.ParseIP("192.168.24.1"),
						Table:     syscall.RT_TABLE_MAIN,
						Priority:  200,
					},
				},
			},
		}, {
			setup: "add 2001:db8:31::/64 via 2001:db8:24::1 metric 200",
			expected: []netlink.RouteUpdate{
				{
					Type: syscall.RTM_NEWROUTE,
					Route: netlink.Route{
						LinkIndex: 2,
						Dst:       config.MustParseCIDR("2001:db8:31::/64"),
						Gw:        net.ParseIP("2001:db8:24::1"),
						Table:     syscall.RT_TABLE_MAIN,
						Priority:  200,
					},
				},
			},
		}, {
			setup: "add 192.168.32.0/24 nexthop via 192.168.24.1 nexthop via 192.168.24.2",
			expected: []netlink.RouteUpdate{
				{
					Type: syscall.RTM_NEWROUTE,
					Route: netlink.Route{
						LinkIndex: 2,
						Dst:       config.MustParseCIDR("192.168.32.0/24"),
						MultiPath: []*netlink.NexthopInfo{
							&netlink.NexthopInfo{
								LinkIndex: 2,
								Gw:        net.ParseIP("192.168.24.1"),
							},
							&netlink.NexthopInfo{
								LinkIndex: 2,
								Gw:        net.ParseIP("192.168.24.2"),
							},
						},
						Table: syscall.RT_TABLE_MAIN,
					},
				},
			},
		}, {
			// May be dependent on kernel version
			setup: "add 2001:db8:32::/64 nexthop via 2001:db8:24::1 nexthop via 2001:db8:24::2",
			expected: []netlink.RouteUpdate{
				{
					Type: syscall.RTM_NEWROUTE,
					Route: netlink.Route{
						LinkIndex: 2,
						Dst:       config.MustParseCIDR("2001:db8:32::/64"),
						MultiPath: []*netlink.NexthopInfo{
							&netlink.NexthopInfo{
								LinkIndex: 2,
								Gw:        net.ParseIP("2001:db8:24::1"),
							},
							&netlink.NexthopInfo{
								LinkIndex: 2,
								Gw:        net.ParseIP("2001:db8:24::2"),
							},
						},
						Table: syscall.RT_TABLE_MAIN,
					},
				},
			},
		},
	}

	for _, tc := range cases {
		ready := make(chan struct{})
		got := []*netlink.RouteUpdate{}
		c.Subscribe(func(notification Notification) {
			u := notification.RouteUpdate
			if u == nil {
				t.Fatalf("Non-route update received: %v", notification)
			}
			got = append(got, u)
			close(ready)
		})
		var outbuf, errbuf bytes.Buffer
		cmd := exec.Command("sh", "-exc", fmt.Sprintf("ip route %s", tc.setup))
		cmd.Stdout = &outbuf
		cmd.Stderr = &errbuf
		if err := cmd.Run(); err != nil {
			t.Fatalf("Unable to setup route\n** Setup:\n%s\n** Stdout:\n%s\n** Stderr:\n%s\n** Error:\n%+v",
				tc.setup, outbuf.String(), errbuf.String(), err)
		}
		timeout := time.After(1 * time.Second)
		select {
		case <-ready:
		case <-timeout:
		}
		if diff := helpers.Diff(got, tc.expected); diff != "" {
			t.Fatalf("route %q (-got, +want):\n%s", tc.setup, diff)
		}
	}

	// Check counters
	if counter := r.Counter("callback.calls").Snapshot().Count(); counter < 20 {
		t.Errorf("callbacks were called not called enough times (%d)", counter)
	}
	if counter := r.Counter("route.updates").Snapshot().Count(); counter < 20 {
		t.Errorf("route counter too low (%d)", counter)
	}
	if counter := r.Counter("route.initial.ipv4").Snapshot().Count(); counter < 3 {
		t.Errorf("initial IPv4 route counter too low (%d, expected 3)", counter)
	}
	if counter := r.Counter("route.initial.ipv6").Snapshot().Count(); counter < 3 {
		t.Errorf("initial IPv6 route counter too low (%d, expected 3)", counter)
	}
}

func TestManyManyRoutes(t *testing.T) {
	if testing.Short() {
		t.Skip("Skip many many routes test in short mode")
	}
	routes := 5000
	r := reporter.NewMock()
	c, err := New(r, DefaultConfiguration)
	if err != nil {
		t.Fatalf("New() error:\n%+v", err)
	}

	prefixes := map[string]bool{}
	events := []string{}
	eor := make(chan struct{})
	c.Subscribe(func(notification Notification) {
		if notification.StartOfRIB {
			events = append(events, "start")
			t.Logf("receive a start event with count = %d", len(prefixes))
			prefixes = map[string]bool{}
			return
		}
		if notification.EndOfRIB {
			events = append(events, "end")
			t.Logf("receive a end event with count = %d", len(prefixes))
			if eor != nil {
				close(eor)
			}
			return
		}
		u := notification.RouteUpdate
		if u.Table != syscall.RT_TABLE_MAIN {
			return
		}
		if u.Dst.IP.To4() == nil {
			return
		}
		var evt string
		switch u.Type {
		case syscall.RTM_NEWROUTE:
			evt = "route+"
			prefixes[u.Dst.String()] = true
		case syscall.RTM_DELROUTE:
			evt = "route-"
			delete(prefixes, u.Dst.String())
		}

		if len(events) == 0 {
			events = append(events, evt)
			return
		}
		last := events[len(events)-1]
		if last == evt {
			return
		}
		events = append(events, evt)
	})

	setup := fmt.Sprintf(`
[ -d /sys/class/net/dummy0 ] || ip link add name dummy0 type dummy || true
ip link set up dev dummy0
sysctl -w net.ipv6.conf.dummy0.disable_ipv6=1
ip route flush table main
ip -6 route flush table main
ip route add 192.168.0.0/24 dev dummy0
seq 1 %d | while read n; do
  j=$((n%%65536/256))
  k=$((n%%256))
  ip route add 10.2.$j.$k/32 via 192.168.0.1
done`, routes)
	var outbuf, errbuf bytes.Buffer
	cmd := exec.Command("sh", "-ec", setup)
	cmd.Stdout = &outbuf
	cmd.Stderr = &errbuf
	if err := cmd.Run(); err != nil {
		t.Fatalf("Unable setup many routes\n** Setup:\n%s\n** Stdout:\n%s\n** Stderr:\n%s\n** Error:\n%+v",
			setup, outbuf.String(), errbuf.String(), err)
	}

	// Start component
	if err := c.Start(); err != nil {
		t.Fatalf("Start() error:\n%+v", err)
	}
	defer func() {
		if err := c.Stop(); err != nil {
			t.Fatalf("Stop() error:\n%+v", err)
		}
	}()

	<-eor
	if diff := helpers.Diff(events, []string{"start", "route+", "end"}); diff != "" {
		t.Fatalf("unexpected log of events for initial routes (-got, +want):\n%s", diff)
	}
	got := len(prefixes)
	if got != routes+1 {
		t.Fatalf("unexpected number of initial routes (%d, expected %d)",
			got, routes+1)
	}

	prevCount := len(prefixes)
	events = []string{}
	eor = nil
	outbuf.Reset()
	errbuf.Reset()
	cmd = exec.Command("sh", "-ec", "ip route flush dev dummy0")
	cmd.Stdout = &outbuf
	cmd.Stderr = &errbuf
	if err := cmd.Run(); err != nil {
		t.Fatalf("Unable to flush routes\n** Setup:\n%s\n** Stdout:\n%s\n** Stderr:\n%s\n** Error:\n%+v",
			setup, outbuf.String(), errbuf.String(), err)
	}
	for {
		time.Sleep(500 * time.Millisecond)
		count := len(prefixes)
		if prevCount == count {
			break
		}
		prevCount = count
	}
	// The netlink socket should be too small to get all route
	// notifications at once. Therefore, it is expected we have to
	// scan again the routes to get the result. We may even have
	// to scan several times. If this doesn't work, we'll need a
	// method to enforce socket receive buffer to something like
	// 106496. This test is quite racy.
	events = append(events, "", "", "", "")
	if diff := helpers.Diff(events[:4], []string{
		"route-", // Routes are being removed
		"start",  // Overflow, we start from scratch
		"route+", // Routes not removed yet
		"end",
		// We could have either "route-", "start" or even nothing.
	}); diff != "" {
		t.Fatalf("unexpected log of events for removed routes (-got, +want):\n%s", diff)
	}
	got = len(prefixes)
	if got != 0 {
		t.Fatalf("unexpected remaining routes (%d, expected 0)", got)
	}
}
