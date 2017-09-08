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
	c, err := New(r)
	if err != nil {
		t.Fatalf("New() error:\n%+v", err)
	}

	// Setup observer
	done := make(chan struct{})
	c.Subscribe("slow", func(u netlink.RouteUpdate) {
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
	c, err := New(r)
	if err != nil {
		t.Fatalf("New() error:\n%+v", err)
	}

	// Setup observer
	got := []netlink.RouteUpdate{}
	done := make(chan struct{})
	c.Subscribe("store", func(u netlink.RouteUpdate) {
		if u.Table == syscall.RT_TABLE_UNSPEC {
			close(done)
			return
		}
		if u.Table == syscall.RT_TABLE_LOCAL {
			return
		}
		got = append(got, u)
	})
	defer c.Unsubscribe("store")

	// Add some initial routes
	setup := `
[ -d /sys/class/net/dummy0 ] || ip link add name dummy0 type dummy || true
ip link set up dev dummy0
sysctl -w net.ipv6.conf.dummy0.optimistic_dad=1
sysctl -w net.ipv6.conf.dummy0.disable_ipv6=0
ip route flush dev dummy0
ip -6 route flush dev dummy0
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
		got := []netlink.RouteUpdate{}
		c.Subscribe("store", func(u netlink.RouteUpdate) {
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
	r := reporter.NewMock()
	c, err := New(r)
	if err != nil {
		t.Fatalf("New() error:\n%+v", err)
	}

	count := 0
	done := make(chan struct{})
	c.Subscribe("discard", func(u netlink.RouteUpdate) {
		if u.Table == syscall.RT_TABLE_UNSPEC {
			close(done)
			return
		}
		if u.Table != syscall.RT_TABLE_MAIN {
			return
		}
		count++
	})

	setup := `
[ -d /sys/class/net/dummy0 ] || ip link add name dummy0 type dummy || true
ip link set up dev dummy0
sysctl -w net.ipv6.conf.dummy0.disable_ipv6=1
ip route flush dev dummy0
ip route add 192.168.0.0/24 dev dummy0
seq 1 10000 | while read n; do
  j=$((n%65536/256))
  k=$((n%256))
  ip route add 10.2.$j.$k/32 via 192.168.0.1
done
`
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

	<-done
	if count != 10001 {
		t.Fatalf("unexpected number of initial routes (%d, expected 10001)", count)
	}

	prevCount := 0
	count = 0
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
		time.Sleep(200 * time.Millisecond)
		if prevCount == count {
			break
		}
	}
	if count != 10001 {
		t.Fatalf("unexpected number of removed routes (%d, expected 10001)", count)
	}
}
