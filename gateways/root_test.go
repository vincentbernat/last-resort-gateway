package gateways

import (
	"net"
	"sync"
	"syscall"
	"testing"
	"time"

	knetlink "github.com/vishvananda/netlink"

	"lrg/config"
	"lrg/helpers"
	"lrg/netlink"
	"lrg/reporter"
)

func TestGateways(t *testing.T) {
	defaultIPv4 := config.MustParsePrefix("0.0.0.0/0")
	simpleConfiguration := Configuration{
		LRGConfiguration{
			From: LRGFromConfiguration{
				Prefix: defaultIPv4,
				Table:  DefaultTable,
			},
			To: LRGToConfiguration{
				Prefix:    defaultIPv4,
				Protocol:  DefaultToProtocol,
				Metric:    DefaultToMetric,
				Table:     DefaultTable,
				Blackhole: false,
			},
		},
	}
	r := reporter.NewMock()
	cases := []struct {
		description   string
		config        Configuration
		notifications []netlink.Notification
		expected      knetlink.Route
	}{
		{
			description: "empty configuration",
			config:      Configuration{},
			notifications: []netlink.Notification{
				netlink.Notification{StartOfRIB: true},
				netlink.Notification{EndOfRIB: true},
			},
			expected: knetlink.Route{},
		}, {
			description: "empty RIB",
			config:      simpleConfiguration,
			notifications: []netlink.Notification{
				netlink.Notification{StartOfRIB: true},
				netlink.Notification{EndOfRIB: true},
			},
			expected: knetlink.Route{},
		}, {
			description: "empty RIB with blackhole enabled",
			config: Configuration{
				LRGConfiguration{
					From: LRGFromConfiguration{
						Prefix: defaultIPv4,
						Table:  DefaultTable,
					},
					To: LRGToConfiguration{
						Prefix:    defaultIPv4,
						Protocol:  DefaultToProtocol,
						Metric:    DefaultToMetric,
						Table:     DefaultTable,
						Blackhole: true,
					},
				},
			},
			notifications: []netlink.Notification{
				netlink.Notification{StartOfRIB: true},
				netlink.Notification{EndOfRIB: true},
			},
			expected: knetlink.Route{
				Dst:      config.MustParseCIDR("0.0.0.0/0"),
				Table:    int(DefaultTable.ID),
				Protocol: int(DefaultToProtocol.ID),
				Priority: int(DefaultToMetric),
				Type:     syscall.RTN_BLACKHOLE,
			},
		}, {
			description: "non-default target table",
			config: Configuration{
				LRGConfiguration{
					From: LRGFromConfiguration{
						Prefix: defaultIPv4,
						Table:  DefaultTable,
					},
					To: LRGToConfiguration{
						Prefix:    defaultIPv4,
						Protocol:  DefaultToProtocol,
						Metric:    DefaultToMetric,
						Table:     config.Table{ID: 200},
						Blackhole: true,
					},
				},
			},
			notifications: []netlink.Notification{
				netlink.Notification{StartOfRIB: true},
				netlink.Notification{EndOfRIB: true},
			},
			expected: knetlink.Route{
				Dst:      config.MustParseCIDR("0.0.0.0/0"),
				Table:    200,
				Protocol: int(DefaultToProtocol.ID),
				Priority: int(DefaultToMetric),
				Type:     syscall.RTN_BLACKHOLE,
			},
		}, {
			description: "non-default target metric",
			config: Configuration{
				LRGConfiguration{
					From: LRGFromConfiguration{
						Prefix: defaultIPv4,
						Table:  DefaultTable,
					},
					To: LRGToConfiguration{
						Prefix:    defaultIPv4,
						Protocol:  DefaultToProtocol,
						Metric:    100,
						Table:     DefaultTable,
						Blackhole: true,
					},
				},
			},
			notifications: []netlink.Notification{
				netlink.Notification{StartOfRIB: true},
				netlink.Notification{EndOfRIB: true},
			},
			expected: knetlink.Route{
				Dst:      config.MustParseCIDR("0.0.0.0/0"),
				Table:    int(DefaultTable.ID),
				Protocol: int(DefaultToProtocol.ID),
				Priority: 100,
				Type:     syscall.RTN_BLACKHOLE,
			},
		}, {
			description: "non-default target protocol",
			config: Configuration{
				LRGConfiguration{
					From: LRGFromConfiguration{
						Prefix: defaultIPv4,
						Table:  DefaultTable,
					},
					To: LRGToConfiguration{
						Prefix:    defaultIPv4,
						Protocol:  config.Protocol{ID: 5},
						Metric:    DefaultToMetric,
						Table:     DefaultTable,
						Blackhole: true,
					},
				},
			},
			notifications: []netlink.Notification{
				netlink.Notification{StartOfRIB: true},
				netlink.Notification{EndOfRIB: true},
			},
			expected: knetlink.Route{
				Dst:      config.MustParseCIDR("0.0.0.0/0"),
				Table:    int(DefaultTable.ID),
				Protocol: 5,
				Priority: int(DefaultToMetric),
				Type:     syscall.RTN_BLACKHOLE,
			},
		}, {
			description: "different target prefix",
			config: Configuration{
				LRGConfiguration{
					From: LRGFromConfiguration{
						Prefix: defaultIPv4,
						Table:  DefaultTable,
					},
					To: LRGToConfiguration{
						Prefix:    config.MustParsePrefix("10.0.0.0/8"),
						Protocol:  DefaultToProtocol,
						Metric:    DefaultToMetric,
						Table:     DefaultTable,
						Blackhole: true,
					},
				},
			},
			notifications: []netlink.Notification{
				netlink.Notification{StartOfRIB: true},
				netlink.Notification{EndOfRIB: true},
			},
			expected: knetlink.Route{
				Dst:      config.MustParseCIDR("10.0.0.0/8"),
				Table:    int(DefaultTable.ID),
				Protocol: int(DefaultToProtocol.ID),
				Priority: int(DefaultToMetric),
				Type:     syscall.RTN_BLACKHOLE,
			},
		}, {
			description: "matching route in initial RIB",
			config:      simpleConfiguration,
			notifications: []netlink.Notification{
				netlink.Notification{StartOfRIB: true},
				netlink.Notification{
					RouteUpdate: &knetlink.RouteUpdate{
						Type: syscall.RTM_NEWROUTE,
						Route: knetlink.Route{
							Dst:   config.MustParseCIDR("0.0.0.0/0"),
							Table: int(DefaultTable.ID),
						},
					},
				},
				netlink.Notification{EndOfRIB: true},
			},
			expected: knetlink.Route{
				Dst:      config.MustParseCIDR("0.0.0.0/0"),
				Table:    int(DefaultTable.ID),
				Protocol: int(DefaultToProtocol.ID),
				Priority: int(DefaultToMetric),
			},
		}, {
			description: "non-matching route in initial RIB",
			config:      simpleConfiguration,
			notifications: []netlink.Notification{
				netlink.Notification{StartOfRIB: true},
				netlink.Notification{
					RouteUpdate: &knetlink.RouteUpdate{
						Type: syscall.RTM_NEWROUTE,
						Route: knetlink.Route{
							Dst:   config.MustParseCIDR("10.0.0.0/8"),
							Table: int(DefaultTable.ID),
						},
					},
				},
				netlink.Notification{EndOfRIB: true},
			},
			expected: knetlink.Route{},
		}, {
			description: "candidate route disappears",
			config:      simpleConfiguration,
			notifications: []netlink.Notification{
				netlink.Notification{StartOfRIB: true},
				netlink.Notification{
					RouteUpdate: &knetlink.RouteUpdate{
						Type: syscall.RTM_NEWROUTE,
						Route: knetlink.Route{
							Dst:   config.MustParseCIDR("0.0.0.0/0"),
							Table: int(DefaultTable.ID),
						},
					},
				},
				netlink.Notification{
					RouteUpdate: &knetlink.RouteUpdate{
						Type: syscall.RTM_DELROUTE,
						Route: knetlink.Route{
							Dst:   config.MustParseCIDR("0.0.0.0/0"),
							Table: int(DefaultTable.ID),
						},
					},
				},
				netlink.Notification{EndOfRIB: true},
			},
			expected: knetlink.Route{
				Dst:      config.MustParseCIDR("0.0.0.0/0"),
				Table:    int(DefaultTable.ID),
				Protocol: int(DefaultToProtocol.ID),
				Priority: int(DefaultToMetric),
			},
		}, {
			description: "candidate route updated",
			config:      simpleConfiguration,
			notifications: []netlink.Notification{
				netlink.Notification{StartOfRIB: true},
				netlink.Notification{
					RouteUpdate: &knetlink.RouteUpdate{
						Type: syscall.RTM_NEWROUTE,
						Route: knetlink.Route{
							Dst:   config.MustParseCIDR("0.0.0.0/0"),
							Table: int(DefaultTable.ID),
						},
					},
				},
				netlink.Notification{
					RouteUpdate: &knetlink.RouteUpdate{
						Type: syscall.RTM_NEWROUTE,
						Route: knetlink.Route{
							Dst:   config.MustParseCIDR("0.0.0.0/0"),
							Table: int(DefaultTable.ID),
							Gw:    net.ParseIP("1.1.1.1"),
						},
					},
				},
				netlink.Notification{EndOfRIB: true},
			},
			expected: knetlink.Route{
				Dst:      config.MustParseCIDR("0.0.0.0/0"),
				Table:    int(DefaultTable.ID),
				Protocol: int(DefaultToProtocol.ID),
				Priority: int(DefaultToMetric),
				Gw:       net.ParseIP("1.1.1.1"),
			},
		}, {
			description: "additional candidate route",
			config:      simpleConfiguration,
			notifications: []netlink.Notification{
				netlink.Notification{StartOfRIB: true},
				netlink.Notification{
					RouteUpdate: &knetlink.RouteUpdate{
						Type: syscall.RTM_NEWROUTE,
						Route: knetlink.Route{
							Dst:   config.MustParseCIDR("0.0.0.0/0"),
							Table: int(DefaultTable.ID),
						},
					},
				},
				netlink.Notification{
					RouteUpdate: &knetlink.RouteUpdate{
						Type: syscall.RTM_NEWROUTE,
						Route: knetlink.Route{
							Dst:      config.MustParseCIDR("0.0.0.0/0"),
							Table:    int(DefaultTable.ID),
							Gw:       net.ParseIP("1.1.1.1"),
							Priority: 200,
						},
					},
				},
				netlink.Notification{EndOfRIB: true},
			},
			expected: knetlink.Route{
				Dst:      config.MustParseCIDR("0.0.0.0/0"),
				Table:    int(DefaultTable.ID),
				Protocol: int(DefaultToProtocol.ID),
				Priority: int(DefaultToMetric),
			},
		}, {
			description: "additional candidate route and original candidate disappears",
			config:      simpleConfiguration,
			notifications: []netlink.Notification{
				netlink.Notification{StartOfRIB: true},
				netlink.Notification{
					RouteUpdate: &knetlink.RouteUpdate{
						Type: syscall.RTM_NEWROUTE,
						Route: knetlink.Route{
							Dst:   config.MustParseCIDR("0.0.0.0/0"),
							Table: int(DefaultTable.ID),
						},
					},
				},
				netlink.Notification{EndOfRIB: true},
				netlink.Notification{
					RouteUpdate: &knetlink.RouteUpdate{
						Type: syscall.RTM_NEWROUTE,
						Route: knetlink.Route{
							Dst:      config.MustParseCIDR("0.0.0.0/0"),
							Table:    int(DefaultTable.ID),
							Gw:       net.ParseIP("1.1.1.1"),
							Priority: 200,
						},
					},
				},
				netlink.Notification{
					RouteUpdate: &knetlink.RouteUpdate{
						Type: syscall.RTM_DELROUTE,
						Route: knetlink.Route{
							Dst:   config.MustParseCIDR("0.0.0.0/0"),
							Table: int(DefaultTable.ID),
						},
					},
				},
			},
			expected: knetlink.Route{
				Dst:      config.MustParseCIDR("0.0.0.0/0"),
				Table:    int(DefaultTable.ID),
				Protocol: int(DefaultToProtocol.ID),
				Priority: int(DefaultToMetric),
				Gw:       net.ParseIP("1.1.1.1"),
			},
		}, {
			description: "target route disappears",
			config:      simpleConfiguration,
			notifications: []netlink.Notification{
				netlink.Notification{StartOfRIB: true},
				netlink.Notification{
					RouteUpdate: &knetlink.RouteUpdate{
						Type: syscall.RTM_NEWROUTE,
						Route: knetlink.Route{
							Dst:   config.MustParseCIDR("0.0.0.0/0"),
							Table: int(DefaultTable.ID),
						},
					},
				},
				netlink.Notification{EndOfRIB: true},
				netlink.Notification{}, // don't remember last installed route
				netlink.Notification{
					RouteUpdate: &knetlink.RouteUpdate{
						Type: syscall.RTM_DELROUTE,
						Route: knetlink.Route{
							Dst:      config.MustParseCIDR("0.0.0.0/0"),
							Table:    int(DefaultTable.ID),
							Protocol: int(DefaultToProtocol.ID),
							Priority: int(DefaultToMetric),
						},
					},
				},
			},
			expected: knetlink.Route{
				Dst:      config.MustParseCIDR("0.0.0.0/0"),
				Table:    int(DefaultTable.ID),
				Protocol: int(DefaultToProtocol.ID),
				Priority: int(DefaultToMetric),
			},
		}, {
			description: "target route changes and get reinstalled",
			config:      simpleConfiguration,
			notifications: []netlink.Notification{
				netlink.Notification{StartOfRIB: true},
				netlink.Notification{
					RouteUpdate: &knetlink.RouteUpdate{
						Type: syscall.RTM_NEWROUTE,
						Route: knetlink.Route{
							Dst:   config.MustParseCIDR("0.0.0.0/0"),
							Table: int(DefaultTable.ID),
						},
					},
				},
				netlink.Notification{EndOfRIB: true},
				netlink.Notification{}, // don't remember last installed route
				netlink.Notification{
					RouteUpdate: &knetlink.RouteUpdate{
						Type: syscall.RTM_NEWROUTE,
						Route: knetlink.Route{
							Dst:      config.MustParseCIDR("0.0.0.0/0"),
							Table:    int(DefaultTable.ID),
							Protocol: int(DefaultToProtocol.ID),
							Priority: int(DefaultToMetric),
							Gw:       net.ParseIP("1.1.1.1"),
						},
					},
				},
			},
			expected: knetlink.Route{
				Dst:      config.MustParseCIDR("0.0.0.0/0"),
				Table:    int(DefaultTable.ID),
				Protocol: int(DefaultToProtocol.ID),
				Priority: int(DefaultToMetric),
			},
		},
	}
	for _, tc := range cases {
		var lock sync.Mutex
		last := knetlink.Route{}
		nl, inject := netlink.NewMock(func(r knetlink.Route) error {
			lock.Lock()
			defer lock.Unlock()
			last = r
			return nil
		})
		c, err := New(r, tc.config, Dependencies{Netlink: nl})
		if err != nil {
			t.Errorf("New(%s) error:\n%+v", tc.config, err)
		}
		if err := c.Start(); err != nil {
			t.Errorf("Start() error:\n%+v", err)
			continue
		}
		empty := netlink.Notification{}
		for _, r := range tc.notifications {
			if r == empty {
				time.Sleep(20 * time.Millisecond)
				lock.Lock()
				last = knetlink.Route{}
				lock.Unlock()
			} else {
				inject(r)
			}
		}
		time.Sleep(20 * time.Millisecond)
		lock.Lock()
		lastCopy := last
		lock.Unlock()
		if diff := helpers.Diff(lastCopy, tc.expected); diff != "" {
			t.Errorf("Unexpected last resort gateway [%s]\n** Config:\n%+v\n** Notifications:\n%+v\n** Result (-got +want):\n%s",
				tc.description, tc.config, tc.notifications, diff)
		}
		if err := c.Stop(); err != nil {
			t.Errorf("Stop() error:\n%+v", err)
		}
	}
}
