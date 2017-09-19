package gateways

import (
	"net"
	"strconv"
	"testing"

	"github.com/vishvananda/netlink"
	"gopkg.in/yaml.v2"

	"lrg/config"
	"lrg/helpers"
)

func TestUnmarshalGateways(t *testing.T) {
	defaultIPv4 := config.MustParsePrefix("0.0.0.0/0")
	defaultIPv6 := config.MustParsePrefix("::/0")
	randomPrefix := config.MustParsePrefix("10.16.0.0/16")
	metric0 := config.Metric(0)
	metric1000 := config.Metric(1000)
	cases := []struct {
		input string
		want  Configuration
		err   bool
	}{
		{
			input: "",
			want:  Configuration{},
		}, {
			input: " - from: {}",
			err:   true,
		}, {
			input: " - to: {}",
			err:   true,
		}, {
			input: `
- from:
    prefix: 0.0.0.0/0`,
			want: Configuration{
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
			},
		}, {
			input: `
- from:
    prefix: ::/0`,
			want: Configuration{
				LRGConfiguration{
					From: LRGFromConfiguration{
						Prefix: defaultIPv6,
						Table:  DefaultTable,
					},
					To: LRGToConfiguration{
						Prefix:    defaultIPv6,
						Protocol:  DefaultToProtocol,
						Metric:    DefaultToMetric,
						Table:     DefaultTable,
						Blackhole: false,
					},
				},
			},
		}, {
			input: `
- from:
    prefix: 0.0.0.0/0
- from:
    prefix: ::/0`,
			want: Configuration{
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
				LRGConfiguration{
					From: LRGFromConfiguration{
						Prefix: defaultIPv6,
						Table:  DefaultTable,
					},
					To: LRGToConfiguration{
						Prefix:    defaultIPv6,
						Protocol:  DefaultToProtocol,
						Metric:    DefaultToMetric,
						Table:     DefaultTable,
						Blackhole: false,
					},
				},
			},
		}, {
			input: `
- from:
    prefix: 0.0.0.0/0
  to:
    prefix: 10.16.0.0/16`,
			want: Configuration{
				LRGConfiguration{
					From: LRGFromConfiguration{
						Prefix: defaultIPv4,
						Table:  DefaultTable,
					},
					To: LRGToConfiguration{
						Prefix:    randomPrefix,
						Protocol:  DefaultToProtocol,
						Metric:    DefaultToMetric,
						Table:     DefaultTable,
						Blackhole: false,
					},
				},
			},
		}, {
			input: `
- from:
    prefix: 0.0.0.0/0
  to:
    prefix: ::/0`,
			want: Configuration{},
			err:  true,
		}, {
			input: `
- from:
    prefix: 0.0.0.0/0
    protocol: kernel
    metric: 0
    table: 254
  to:
    protocol: 254
    metric: 1000
    table: public
    blackhole: true`,
			want: Configuration{
				LRGConfiguration{
					From: LRGFromConfiguration{
						Prefix:   defaultIPv4,
						Protocol: &config.Protocol{ID: 2, Name: "kernel"},
						Metric:   &metric0,
						Table:    config.Table{ID: 254},
					},
					To: LRGToConfiguration{
						Prefix:    defaultIPv4,
						Protocol:  config.Protocol{ID: 254},
						Metric:    metric1000,
						Table:     config.Table{ID: 90, Name: "public"},
						Blackhole: true,
					},
				},
			},
		},
	}
	for _, tc := range cases {
		var got Configuration
		err := yaml.Unmarshal([]byte(tc.input), &got)
		switch {
		case err != nil && !tc.err:
			t.Errorf("Unmarshal(%q) error:\n%+v", tc.input, err)
		case err == nil && tc.err:
			t.Errorf("Unmarshal(%q) == %v but expected error", tc.input, got)
		default:
			if diff := helpers.Diff(got, tc.want); diff != "" {
				t.Errorf("Unmarshal(%q) (-got, +want):\n%s", tc.input, diff)
			}
		}
	}
}

func TestFromMatch(t *testing.T) {
	defaultIPv4 := net.IPNet(config.MustParsePrefix("0.0.0.0/0"))
	defaultIPv6 := net.IPNet(config.MustParsePrefix("::/0"))
	randomPrefix := net.IPNet(config.MustParsePrefix("10.16.0.0/16"))
	metric1000 := config.Metric(1000)
	cases := []struct {
		config   LRGFromConfiguration
		route    netlink.Route
		expected bool
	}{
		{
			config: LRGFromConfiguration{
				Prefix: config.Prefix(defaultIPv4),
				Table:  config.Table{ID: 254},
			},
			route: netlink.Route{
				Dst:   &defaultIPv4,
				Table: 254,
			},
			expected: true,
		}, {
			config: LRGFromConfiguration{
				Prefix: config.Prefix(defaultIPv4),
				Table:  config.Table{ID: 254},
			},
			route: netlink.Route{
				Dst:   &randomPrefix,
				Table: 254,
			},
			expected: false,
		}, {
			config: LRGFromConfiguration{
				Prefix: config.Prefix(defaultIPv6),
				Table:  config.Table{ID: 254},
			},
			route: netlink.Route{
				Dst:   &defaultIPv6,
				Table: 254,
			},
			expected: true,
		}, {
			config: LRGFromConfiguration{
				Prefix: config.Prefix(defaultIPv4),
				Table:  config.Table{ID: 254},
			},
			route: netlink.Route{
				Dst:   &defaultIPv6,
				Table: 254,
			},
			expected: false,
		}, {
			config: LRGFromConfiguration{
				Prefix: config.Prefix(defaultIPv6),
				Table:  config.Table{ID: 254},
			},
			route: netlink.Route{
				Dst:   &defaultIPv4,
				Table: 254,
			},
			expected: false,
		}, {
			config: LRGFromConfiguration{
				Prefix: config.Prefix(defaultIPv4),
				Table:  config.Table{ID: 254},
			},
			route: netlink.Route{
				Dst:      &defaultIPv4,
				Table:    254,
				Priority: 100,
			},
			expected: true,
		}, {
			config: LRGFromConfiguration{
				Prefix: config.Prefix(defaultIPv6),
				Table:  config.Table{ID: 254},
			},
			route: netlink.Route{
				Dst:      &defaultIPv6,
				Table:    254,
				Priority: 100,
			},
			expected: true,
		}, {
			config: LRGFromConfiguration{
				Prefix: config.Prefix(defaultIPv4),
				Table:  config.Table{ID: 254},
			},
			route: netlink.Route{
				Dst:      &defaultIPv4,
				Table:    254,
				Protocol: 5,
			},
			expected: true,
		}, {
			config: LRGFromConfiguration{
				Prefix: config.Prefix(defaultIPv6),
				Table:  config.Table{ID: 254},
			},
			route: netlink.Route{
				Dst:      &defaultIPv6,
				Table:    254,
				Protocol: 5,
			},
			expected: true,
		}, {
			config: LRGFromConfiguration{
				Prefix: config.Prefix(defaultIPv4),
				Metric: &metric1000,
				Table:  config.Table{ID: 254},
			},
			route: netlink.Route{
				Dst:   &defaultIPv4,
				Table: 254,
			},
			expected: false,
		}, {
			config: LRGFromConfiguration{
				Prefix: config.Prefix(defaultIPv6),
				Metric: &metric1000,
				Table:  config.Table{ID: 254},
			},
			route: netlink.Route{
				Dst:   &defaultIPv6,
				Table: 254,
			},
			expected: false,
		}, {
			config: LRGFromConfiguration{
				Prefix: config.Prefix(defaultIPv4),
				Metric: &metric1000,
				Table:  config.Table{ID: 254},
			},
			route: netlink.Route{
				Dst:      &defaultIPv4,
				Priority: 1000,
				Table:    254,
			},
			expected: true,
		}, {
			config: LRGFromConfiguration{
				Prefix: config.Prefix(defaultIPv6),
				Metric: &metric1000,
				Table:  config.Table{ID: 254},
			},
			route: netlink.Route{
				Dst:      &defaultIPv6,
				Priority: 1000,
				Table:    254,
			},
			expected: true,
		}, {
			config: LRGFromConfiguration{
				Prefix:   config.Prefix(defaultIPv4),
				Protocol: &config.Protocol{ID: 5},
				Table:    config.Table{ID: 254},
			},
			route: netlink.Route{
				Dst:      &defaultIPv4,
				Protocol: 5,
				Table:    254,
			},
			expected: true,
		}, {
			config: LRGFromConfiguration{
				Prefix:   config.Prefix(defaultIPv6),
				Protocol: &config.Protocol{ID: 5},
				Table:    config.Table{ID: 254},
			},
			route: netlink.Route{
				Dst:      &defaultIPv6,
				Protocol: 5,
				Table:    254,
			},
			expected: true,
		}, {
			config: LRGFromConfiguration{
				Prefix:   config.Prefix(defaultIPv4),
				Protocol: &config.Protocol{ID: 5},
				Table:    config.Table{ID: 254},
			},
			route: netlink.Route{
				Dst:      &defaultIPv4,
				Protocol: 6,
				Table:    254,
			},
			expected: false,
		}, {
			config: LRGFromConfiguration{
				Prefix:   config.Prefix(defaultIPv6),
				Protocol: &config.Protocol{ID: 5},
				Table:    config.Table{ID: 254},
			},
			route: netlink.Route{
				Dst:      &defaultIPv6,
				Protocol: 6,
				Table:    254,
			},
			expected: false,
		},
	}
	for _, tc := range cases {
		got := tc.config.Match(&tc.route)
		if tc.expected != got {
			t.Errorf("LRGFromConfiguration.Match(%s,%s) == %s but expected %s",
				tc.config, tc.route,
				strconv.FormatBool(got), strconv.FormatBool(tc.expected))
		}
	}
}

func TestToMatch(t *testing.T) {
	defaultIPv4 := net.IPNet(config.MustParsePrefix("0.0.0.0/0"))
	defaultIPv6 := net.IPNet(config.MustParsePrefix("::/0"))
	randomPrefix := net.IPNet(config.MustParsePrefix("10.16.0.0/16"))
	cases := []struct {
		config   LRGToConfiguration
		route    netlink.Route
		expected bool
	}{
		{
			config: LRGToConfiguration{
				Prefix:   config.Prefix(defaultIPv4),
				Metric:   10,
				Protocol: config.Protocol{ID: 17},
				Table:    config.Table{ID: 254},
			},
			route: netlink.Route{
				Dst:      &defaultIPv4,
				Table:    254,
				Priority: 10,
				Protocol: 17,
			},
			expected: true,
		}, {
			config: LRGToConfiguration{
				Prefix:   config.Prefix(defaultIPv4),
				Metric:   10,
				Protocol: config.Protocol{ID: 17},
				Table:    config.Table{ID: 254},
			},
			route: netlink.Route{
				Dst:      &randomPrefix,
				Table:    254,
				Priority: 10,
				Protocol: 17,
			},
			expected: false,
		}, {
			config: LRGToConfiguration{
				Prefix:   config.Prefix(defaultIPv6),
				Metric:   10,
				Protocol: config.Protocol{ID: 17},
				Table:    config.Table{ID: 254},
			},
			route: netlink.Route{
				Dst:      &defaultIPv6,
				Table:    254,
				Priority: 10,
				Protocol: 17,
			},
			expected: true,
		}, {
			config: LRGToConfiguration{
				Prefix:   config.Prefix(defaultIPv4),
				Metric:   10,
				Protocol: config.Protocol{ID: 17},
				Table:    config.Table{ID: 254},
			},
			route: netlink.Route{
				Dst:      &defaultIPv6,
				Table:    254,
				Priority: 10,
				Protocol: 17,
			},
			expected: false,
		}, {
			config: LRGToConfiguration{
				Prefix:   config.Prefix(defaultIPv6),
				Metric:   10,
				Protocol: config.Protocol{ID: 17},
				Table:    config.Table{ID: 254},
			},
			route: netlink.Route{
				Dst:      &defaultIPv4,
				Table:    254,
				Priority: 10,
				Protocol: 17,
			},
			expected: false,
		}, {
			config: LRGToConfiguration{
				Prefix:   config.Prefix(defaultIPv4),
				Metric:   1000,
				Protocol: config.Protocol{ID: 17},
				Table:    config.Table{ID: 254},
			},
			route: netlink.Route{
				Dst:      &defaultIPv4,
				Table:    254,
				Priority: 10,
				Protocol: 17,
			},
			expected: false,
		}, {
			config: LRGToConfiguration{
				Prefix:   config.Prefix(defaultIPv6),
				Metric:   1000,
				Protocol: config.Protocol{ID: 17},
				Table:    config.Table{ID: 254},
			},
			route: netlink.Route{
				Dst:      &defaultIPv6,
				Table:    254,
				Priority: 10,
				Protocol: 17,
			},
			expected: false,
		}, {
			config: LRGToConfiguration{
				Prefix:   config.Prefix(defaultIPv4),
				Metric:   10,
				Protocol: config.Protocol{ID: 5},
				Table:    config.Table{ID: 254},
			},
			route: netlink.Route{
				Dst:      &defaultIPv4,
				Protocol: 6,
				Priority: 10,
				Table:    254,
			},
			expected: false,
		}, {
			config: LRGToConfiguration{
				Prefix:   config.Prefix(defaultIPv6),
				Metric:   10,
				Protocol: config.Protocol{ID: 5},
				Table:    config.Table{ID: 254},
			},
			route: netlink.Route{
				Dst:      &defaultIPv6,
				Priority: 10,
				Protocol: 6,
				Table:    254,
			},
			expected: false,
		},
	}
	for _, tc := range cases {
		got := tc.config.Match(&tc.route)
		if tc.expected != got {
			t.Errorf("LRGToConfiguration.Match(%s,%s) == %s but expected %s",
				tc.config, tc.route,
				strconv.FormatBool(got), strconv.FormatBool(tc.expected))
		}
	}
}
