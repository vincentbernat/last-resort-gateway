package gateways

import (
	"net"
	"syscall"
	"testing"

	"github.com/vishvananda/netlink"

	"lrg/config"
	"lrg/helpers"
)

func TestBestCandidateRoute(t *testing.T) {
	cases := []netlink.Route{
		{Tos: 0, Priority: 0},
		{Tos: 0, Priority: 1},
		{Tos: 0, Priority: 2},
		{Tos: 1, Priority: 0},
		{Tos: 1, Priority: 1},
		{Tos: 1, Priority: 2},
		{Tos: 2, Priority: 0},
		{Tos: 2, Priority: 1},
		{Tos: 2, Priority: 2},
		{Tos: 2, Priority: 3},
	}
	linkIndex := 1
	for i1, r1 := range cases {
		r1.LinkIndex = linkIndex
		linkIndex++
		for i2, r2 := range cases {
			r2.LinkIndex = linkIndex
			linkIndex++
			for i3, r3 := range cases {
				r3.LinkIndex = linkIndex
				linkIndex++
				candidates := []*netlink.Route{
					&r1, &r2, &r3,
				}
				got := bestCandidateRoute(candidates)
				var expected netlink.Route
				switch {
				case i1 <= i2 && i2 <= i3:
					expected = r1
				case i1 <= i3 && i3 <= i2:
					expected = r1
				case i2 <= i1 && i1 <= i3:
					expected = r2
				case i2 <= i3 && i3 <= i1:
					expected = r2
				case i3 <= i1 && i1 <= i2:
					expected = r3
				case i3 <= i2 && i2 <= i1:
					expected = r3
				default:
					panic("unexpected")
				}
				if diff := helpers.Diff(got, expected); diff != "" {
					t.Errorf("bestCandidateRoute(%s) (-got +want):\n%s",
						candidates, diff)
				}
			}
		}
	}

	// Special case with no candidates
	got := bestCandidateRoute([]*netlink.Route{})
	if got != nil {
		t.Errorf("bestCandidateRoute([]) == %q but expected nothing",
			got)
	}
}

func TestTargetRoute(t *testing.T) {
	cases := []struct {
		candidate *netlink.Route
		config    LRGToConfiguration
		expected  *netlink.Route
	}{
		{
			// No candidate, no blackhole route
			candidate: nil,
			config: LRGToConfiguration{
				Prefix:    config.MustParsePrefix("10.0.0.0/8"),
				Table:     config.Table{ID: 254},
				Protocol:  config.Protocol{ID: 5},
				Metric:    1000,
				Blackhole: false,
			},
			expected: nil,
		}, {
			candidate: nil,
			config: LRGToConfiguration{
				Prefix:    config.MustParsePrefix("10.0.0.0/8"),
				Table:     config.Table{ID: 254},
				Protocol:  config.Protocol{ID: 5},
				Metric:    1000,
				Blackhole: true,
			},
			expected: &netlink.Route{
				Dst:      config.MustParseCIDR("10.0.0.0/8"),
				Table:    254,
				Protocol: 5,
				Priority: 1000,
				Type:     syscall.RTN_BLACKHOLE,
			},
		}, {
			candidate: &netlink.Route{
				Dst:      config.MustParseCIDR("0.0.0.0/8"),
				Table:    200,
				Protocol: 2,
				Priority: 10,
				Type:     syscall.RTN_UNICAST,
				Gw:       net.IPv4(1, 1, 1, 1),
			},
			config: LRGToConfiguration{
				Prefix:    config.MustParsePrefix("10.0.0.0/8"),
				Table:     config.Table{ID: 254},
				Protocol:  config.Protocol{ID: 5},
				Metric:    1000,
				Blackhole: true,
			},
			expected: &netlink.Route{
				Dst:      config.MustParseCIDR("10.0.0.0/8"),
				Table:    254,
				Protocol: 5,
				Priority: 1000,
				Type:     syscall.RTN_UNICAST,
				Gw:       net.IPv4(1, 1, 1, 1),
			},
		}, {
			candidate: &netlink.Route{
				Dst:      config.MustParseCIDR("::/0"),
				Table:    200,
				Protocol: 2,
				Priority: 10,
				Type:     syscall.RTN_UNICAST,
				Gw:       net.ParseIP("2001:db8:15::1"),
			},
			config: LRGToConfiguration{
				Prefix:    config.MustParsePrefix("::/0"),
				Table:     config.Table{ID: 254},
				Protocol:  config.Protocol{ID: 5},
				Metric:    1000,
				Blackhole: true,
			},
			expected: &netlink.Route{
				Dst:      config.MustParseCIDR("::/0"),
				Table:    254,
				Protocol: 5,
				Priority: 1000,
				Type:     syscall.RTN_UNICAST,
				Gw:       net.ParseIP("2001:db8:15::1"),
			},
		},
	}
	for _, tc := range cases {
		candidates := []*netlink.Route{}
		if tc.candidate != nil {
			candidate := *tc.candidate
			candidates = append(candidates, &candidate)
		}
		got := targetRoute(candidates, &tc.config)
		switch {
		case got == nil && tc.expected == nil:
		case got == nil:
			t.Errorf("targetRoute(%q,%q) not found",
				candidates, tc.config)
		case tc.expected == nil:
			t.Errorf("targetRoute(%q,%q) == %q but expected nothing",
				candidates, tc.config, got)
		default:
			if diff := helpers.Diff(got, tc.expected); diff != "" {
				t.Errorf("targetRoute(%q,%q) (-got +want):\n%s",
					candidates, tc.config, diff)
			}
		}
	}
}
