package netlink

import (
	"testing"
	"time"

	"gopkg.in/yaml.v2"

	"lrg/config"
	"lrg/helpers"
)

func TestUnmarshalConfiguration(t *testing.T) {
	cases := []struct {
		in   string
		want Configuration
	}{
		{
			in:   "{}",
			want: DefaultConfiguration,
		}, {
			in: `
socketsize: 1000000
channelsize: 50
backoffinterval: 1s
backoffmaxinterval: 1m
`,
			want: Configuration{
				SocketSize:         1000000,
				ChannelSize:        50,
				BackoffInterval:    config.Duration(time.Second),
				BackoffMaxInterval: config.Duration(time.Minute),
				CureInterval:       DefaultConfiguration.CureInterval,
			},
		},
	}

	for _, c := range cases {
		var got Configuration
		err := yaml.Unmarshal([]byte(c.in), &got)
		switch {
		case err != nil:
			t.Errorf("Unmarshal(%q) error:\n%+v",
				c.in, err)
		default:
			if diff := helpers.Diff(got, c.want); diff != "" {
				t.Errorf("Unmarshal(%q) (-got +want):\n%s", c.in, diff)
			}
		}
	}
}
