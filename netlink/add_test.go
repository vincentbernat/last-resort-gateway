package netlink

import (
	"bytes"
	"net"
	"os/exec"
	"strings"
	"syscall"
	"testing"

	"github.com/vishvananda/netlink"

	"lrg/config"
	"lrg/helpers"
	"lrg/reporter"
)

func TestAddRoute(t *testing.T) {
	r := reporter.NewMock()
	c, err := New(r, DefaultConfiguration)
	if err != nil {
		t.Fatalf("New() error:\n%+v", err)
	}

	if err := c.Start(); err != nil {
		t.Fatalf("Start() error:\n%+v", err)
	}
	defer func() {
		if err := c.Stop(); err != nil {
			t.Fatalf("Stop() error:\n%+v", err)
		}
	}()

	cases := []struct {
		setup    string
		route    netlink.Route
		expected string
	}{
		{
			setup: "",
			route: netlink.Route{
				LinkIndex: 2,
				Dst:       config.MustParseCIDR("192.168.26.0/24"),
				Table:     syscall.RT_TABLE_MAIN,
			},
			expected: "192.168.26.0/24 dev dummy0",
		}, {
			setup: "",
			route: netlink.Route{
				LinkIndex: 2,
				Dst:       config.MustParseCIDR("2001:db8:16::/64"),
				Table:     syscall.RT_TABLE_MAIN,
			},
			expected: "2001:db8:16::/64 dev dummy0 metric 1024 pref medium",
		}, {
			setup: "",
			route: netlink.Route{
				LinkIndex: 2,
				Dst:       config.MustParseCIDR("192.168.26.0/24"),
				Table:     syscall.RT_TABLE_MAIN,
				Priority:  10,
			},
			expected: "192.168.26.0/24 dev dummy0 metric 10",
		}, {
			setup: "",
			route: netlink.Route{
				LinkIndex: 2,
				Dst:       config.MustParseCIDR("2001:db8:16::/64"),
				Table:     syscall.RT_TABLE_MAIN,
				Priority:  10,
			},
			expected: "2001:db8:16::/64 dev dummy0 metric 10 pref medium",
		}, {
			setup: "",
			route: netlink.Route{
				LinkIndex: 2,
				Dst:       config.MustParseCIDR("192.168.26.0/24"),
				Table:     syscall.RT_TABLE_MAIN,
				Protocol:  2,
			},
			expected: "192.168.26.0/24 dev dummy0 proto kernel",
		}, {
			setup: "",
			route: netlink.Route{
				LinkIndex: 2,
				Dst:       config.MustParseCIDR("2001:db8:54::/64"),
				Table:     syscall.RT_TABLE_MAIN,
				Protocol:  2,
			},
			expected: "2001:db8:54::/64 dev dummy0 proto kernel metric 1024 pref medium",
		}, {
			setup: "",
			route: netlink.Route{
				LinkIndex: 2,
				Dst:       config.MustParseCIDR("192.168.26.0/24"),
				Table:     100,
			},
			expected: "192.168.26.0/24 dev dummy0 table 100",
		}, {
			setup: "",
			route: netlink.Route{
				LinkIndex: 2,
				Dst:       config.MustParseCIDR("2001:db8:54::/64"),
				Table:     100,
			},
			expected: "2001:db8:54::/64 dev dummy0 table 100 metric 1024 pref medium",
		}, {
			setup: "ip route add 192.168.26.0/24 dev dummy0",
			route: netlink.Route{
				LinkIndex: 2,
				Dst:       config.MustParseCIDR("192.168.27.0/24"),
				Gw:        net.ParseIP("192.168.26.1"),
				Table:     syscall.RT_TABLE_MAIN,
			},
			expected: `
192.168.26.0/24 dev dummy0 scope link
192.168.27.0/24 via 192.168.26.1 dev dummy0
`,
		}, {
			setup: "ip route add 192.168.26.0/24 dev dummy0",
			route: netlink.Route{
				Dst: config.MustParseCIDR("192.168.27.0/24"),
				MultiPath: []*netlink.NexthopInfo{
					&netlink.NexthopInfo{
						LinkIndex: 2,
						Gw:        net.ParseIP("192.168.26.1"),
					},
					&netlink.NexthopInfo{
						LinkIndex: 2,
						Gw:        net.ParseIP("192.168.26.2"),
					},
				},
				Table: syscall.RT_TABLE_MAIN,
			},
			expected: `
192.168.26.0/24 dev dummy0 scope link
192.168.27.0/24
nexthop via 192.168.26.1 dev dummy0 weight 1
nexthop via 192.168.26.2 dev dummy0 weight 1
`,
		}, {
			setup: `
ip route add 192.168.26.0/24 dev dummy0
ip route add 192.168.27.0/24 via 192.168.26.1
`,
			route: netlink.Route{
				Dst:   config.MustParseCIDR("192.168.27.0/24"),
				Gw:    net.ParseIP("192.168.26.2"),
				Table: syscall.RT_TABLE_MAIN,
			},
			expected: `
192.168.26.0/24 dev dummy0 scope link
192.168.27.0/24 via 192.168.26.2 dev dummy0
`,
		}, {
			setup: `
ip route add 2001:db8:26::/64 dev dummy0
ip route add 2001:db8:27::/64 via 2001:db8:26::1
`,
			route: netlink.Route{
				Dst:   config.MustParseCIDR("2001:db8:27::/64"),
				Gw:    net.ParseIP("2001:db8:26::2"),
				Table: syscall.RT_TABLE_MAIN,
			},
			expected: `
2001:db8:26::/64 dev dummy0 metric 1024 pref medium
2001:db8:27::/64 via 2001:db8:26::2 dev dummy0 metric 1024 pref medium
`,
		}, {
			setup: `
ip route add 192.168.26.0/24 dev dummy0
ip route add 192.168.27.0/24 via 192.168.26.1
`,
			route: netlink.Route{
				Dst: config.MustParseCIDR("192.168.27.0/24"),
				MultiPath: []*netlink.NexthopInfo{
					&netlink.NexthopInfo{
						LinkIndex: 2,
						Gw:        net.ParseIP("192.168.26.1"),
					},
					&netlink.NexthopInfo{
						LinkIndex: 2,
						Gw:        net.ParseIP("192.168.26.2"),
					},
				},
				Table: syscall.RT_TABLE_MAIN,
			},
			expected: `
192.168.26.0/24 dev dummy0 scope link
192.168.27.0/24
nexthop via 192.168.26.1 dev dummy0 weight 1
nexthop via 192.168.26.2 dev dummy0 weight 1
`,
		}, {
			setup: `
ip route add 192.168.26.0/24 dev dummy0
ip route add 192.168.27.0/24 via 192.168.26.1
`,
			route: netlink.Route{
				Dst:      config.MustParseCIDR("192.168.27.0/24"),
				Gw:       net.ParseIP("192.168.26.2"),
				Priority: 100,
				Table:    syscall.RT_TABLE_MAIN,
			},
			expected: `
192.168.26.0/24 dev dummy0 scope link
192.168.27.0/24 via 192.168.26.1 dev dummy0
192.168.27.0/24 via 192.168.26.2 dev dummy0 metric 100
`,
		}, {
			setup: `
ip route add 2001:db8:26::/64 dev dummy0
ip route add 2001:db8:27::/64 via 2001:db8:26::1
`,
			route: netlink.Route{
				Dst:      config.MustParseCIDR("2001:db8:27::/64"),
				Gw:       net.ParseIP("2001:db8:26::2"),
				Priority: 100,
				Table:    syscall.RT_TABLE_MAIN,
			},
			expected: `
2001:db8:26::/64 dev dummy0 metric 1024 pref medium
2001:db8:27::/64 via 2001:db8:26::2 dev dummy0 metric 100 pref medium
2001:db8:27::/64 via 2001:db8:26::1 dev dummy0 metric 1024 pref medium
`,
		},
	}

	for idx, tc := range cases {
		resetNamespace(t)
		var outbuf, errbuf bytes.Buffer
		cmd := exec.Command("sh", "-exc", tc.setup)
		cmd.Stdout = &outbuf
		cmd.Stderr = &errbuf
		if err := cmd.Run(); err != nil {
			t.Errorf("Unable to setup routes\n** Setup:\n%s\n** Stdout:\n%s\n** Stderr:\n%s\n** Error:\n%+v",
				tc.setup, outbuf.String(), errbuf.String(), err)
			continue
		}

		err := c.AddRoute(tc.route)
		if tc.expected != "" && err != nil {
			t.Errorf("AddRoute(%d: %s) error:\n%+v", idx, tc.route, err)
			continue
		}

		outbuf.Reset()
		errbuf.Reset()
		cmd = exec.Command("sh", "-c", `
ip route show table 0 \
  | grep -v table.local \
  | grep -v '^fe80::/64 dev dummy0 '
`)
		cmd.Stdout = &outbuf
		cmd.Stderr = &errbuf
		if err := cmd.Run(); err != nil {
			t.Errorf("Unable to get routes\n** Stdout:\n%s\n** Stderr:\n%s\n** Error:\n%+v",
				outbuf.String(), errbuf.String(), err)
			continue
		}
		expected := helpers.TrimSpaces(tc.expected)
		got := helpers.TrimSpaces(outbuf.String())
		if diff := helpers.Diff(strings.Split(got, "\n"),
			strings.Split(expected, "\n")); diff != "" {
			t.Errorf("AddRoute(%d: %s) (-got +want):\n%s", idx, tc.route, diff)
		}
	}
}
