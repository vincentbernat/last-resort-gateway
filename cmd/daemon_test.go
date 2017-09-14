package cmd

import (
	"testing"

	"gopkg.in/yaml.v2"

	"lrg/config"
	"lrg/gateways"
	"lrg/helpers"
	"lrg/netlink"
	"lrg/reporter"
	"lrg/reporter/logger"
	"lrg/reporter/metrics"
	"lrg/reporter/sentry"
)

func TestDaemonConfiguration(t *testing.T) {
	defaultIPv4 := config.MustParsePrefix("0.0.0.0/0")
	defaultIPv6 := config.MustParsePrefix("::/0")
	cases := []struct {
		in   string
		want daemonConfiguration
		err  bool
	}{
		{
			in:  "{}",
			err: true,
		}, {
			// Minimal configuration
			in: `
gateways:
  - from:
      prefix: ::/0`,
			want: daemonConfiguration{
				Reporting: reporter.Configuration{
					Logging: logger.DefaultConfiguration,
				},
				Gateways: gateways.Configuration{
					gateways.LRGConfiguration{
						From: gateways.LRGFromConfiguration{
							Prefix: defaultIPv6,
							Table:  gateways.DefaultTable,
						},
						To: gateways.LRGToConfiguration{
							Prefix:    defaultIPv6,
							Table:     gateways.DefaultTable,
							Protocol:  gateways.DefaultToProtocol,
							Metric:    gateways.DefaultToMetric,
							Blackhole: false,
						},
					},
				},
				Netlink: netlink.DefaultConfiguration,
			},
		}, {
			in: `
reporting:
  logging:
    console: true
    syslog: false
    level: debug
  metrics:
    - expvar:
        listen: :8123
  sentry:
    dsn: "http://public:secret@errors"
netlink:
  socketsize: 1000000
gateways:
  - from:
      prefix: 0.0.0.0/0
  - from:
      prefix: ::/0`,
			want: daemonConfiguration{
				Reporting: reporter.Configuration{
					Logging: logger.Configuration{
						Console: true,
						Syslog:  false,
						Level:   4,
					},
					Metrics: metrics.Configuration([]metrics.ExporterConfiguration{
						&metrics.ExpvarConfiguration{
							Listen: config.Addr(":8123"),
						},
					}),
					Sentry: sentry.Configuration{
						DSN: "http://public:secret@errors",
					},
				},
				Netlink: netlink.Configuration{
					SocketSize:         1000000,
					ChannelSize:        netlink.DefaultConfiguration.ChannelSize,
					BackoffInterval:    netlink.DefaultConfiguration.BackoffInterval,
					BackoffMaxInterval: netlink.DefaultConfiguration.BackoffMaxInterval,
					CureInterval:       netlink.DefaultConfiguration.CureInterval,
				},
				Gateways: gateways.Configuration{
					gateways.LRGConfiguration{
						From: gateways.LRGFromConfiguration{
							Prefix: defaultIPv4,
							Table:  gateways.DefaultTable,
						},
						To: gateways.LRGToConfiguration{
							Prefix:    defaultIPv4,
							Table:     gateways.DefaultTable,
							Protocol:  gateways.DefaultToProtocol,
							Metric:    gateways.DefaultToMetric,
							Blackhole: false,
						},
					},
					gateways.LRGConfiguration{
						From: gateways.LRGFromConfiguration{
							Prefix: defaultIPv6,
							Table:  gateways.DefaultTable,
						},
						To: gateways.LRGToConfiguration{
							Prefix:    defaultIPv6,
							Table:     gateways.DefaultTable,
							Protocol:  gateways.DefaultToProtocol,
							Metric:    gateways.DefaultToMetric,
							Blackhole: false,
						},
					},
				},
			},
		},
	}
	for _, tc := range cases {
		var got daemonConfiguration
		err := yaml.Unmarshal([]byte(tc.in), &got)
		switch {
		case err != nil && !tc.err:
			t.Errorf("Unmarshal(%q) error:\n%+v", tc.in, err)
		case err == nil && tc.err:
			t.Errorf("Unmarshal(%q) = %+v but expected error", tc.in, got)
		default:
			if diff := helpers.Diff(got, tc.want); diff != "" {
				t.Errorf("Unmarshal(%q) (-got +want):\n%s", tc.in, diff)
			}
		}
	}
}
