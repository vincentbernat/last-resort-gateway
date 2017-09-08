package gateways

import (
	"testing"

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
					From: LRGFromConfiguration{Prefix: &defaultIPv4},
					To:   LRGToConfiguration{},
				},
			},
		}, {
			input: `
- from:
    prefix: ::/0`,
			want: Configuration{
				LRGConfiguration{
					From: LRGFromConfiguration{Prefix: &defaultIPv6},
					To:   LRGToConfiguration{},
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
					From: LRGFromConfiguration{Prefix: &defaultIPv4},
					To:   LRGToConfiguration{},
				},
				LRGConfiguration{
					From: LRGFromConfiguration{Prefix: &defaultIPv6},
					To:   LRGToConfiguration{},
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
					From: LRGFromConfiguration{Prefix: &defaultIPv4},
					To:   LRGToConfiguration{Prefix: &randomPrefix},
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
						Prefix:   &defaultIPv4,
						Protocol: &config.Protocol{ID: 2, Name: "kernel"},
						Metric:   &metric0,
						Table:    &config.Table{ID: 254},
					},
					To: LRGToConfiguration{
						Protocol:  &config.Protocol{ID: 254},
						Metric:    &metric1000,
						Table:     &config.Table{ID: 90, Name: "public"},
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
