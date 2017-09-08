package config

import (
	"fmt"
	"testing"

	"gopkg.in/yaml.v2"
)

func TestUnmarshalAddr(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{":80", ":80"},
		{"0.0.0.0:80", "0.0.0.0:80"},
		{"127.0.0.1:80", "127.0.0.1:80"},
		{"localhost:80", "127.0.0.1:80"},
		{"ip6-localhost:80", "[::1]:80"},
		{"[2001:db8:cafe::1]:80", "[2001:db8:cafe::1]:80"},
		{"[::]:80", "[::]:80"},
		{"127.0.0.1", ""},
		{"127.0.0.1:65536", ""},
		{"127.0.0.1:-1", ""},
		{"127.0.0.1.14.5:80", ""},
		{"~~hello!!:80", ""},
		{"::15::16:80", ""},
		{"i.shoud.get.nxdomain.invalid:80", ""},
	}
	for _, tc := range cases {
		var got Addr
		input := fmt.Sprintf("%q", tc.in)
		err := yaml.Unmarshal([]byte(input), &got)
		switch {
		case err != nil && tc.want != "":
			t.Errorf("Unmarshal(%q) error\n%+v", tc.in, err)
		case err == nil && tc.want == "":
			t.Errorf("Unmarshal(%q) == %q but expected error", tc.in, got.String())
		case err == nil && tc.want != got.String():
			t.Errorf("Unmarshal(%q) == %q but expected %q", tc.in, got.String(), tc.want)
		}
	}
}

func TestUnmarshalPrefix(t *testing.T) {
	cases := []string{
		"10.0.0.0/8",
		"192.168.1.2/32",
		"0.0.0.0/0",
		"fe80::/16",
		"fe80::1/128",
		"2001:db8::/64",
	}
	for _, c := range cases {
		var got Prefix
		err := yaml.Unmarshal([]byte(c), &got)
		if err != nil {
			t.Errorf("Unmarshal(%q) error\n%+v", c, err)
			continue
		}
		if c != got.String() {
			t.Errorf("Unmarshal(%q) == %v but expected %v", c, got.String(), c)
		}
	}
}

func TestUnmarshalInvalidPrefix(t *testing.T) {
	cases := []string{
		"10.0.0.1/8",
		"fe80::/8",
		"fe80::1/16",
		"10.0.0.0/33",
		"fe80::/129",
		"totally invalid",
	}
	for _, c := range cases {
		var got Prefix
		err := yaml.Unmarshal([]byte(c), &got)
		if err == nil {
			t.Errorf("Unmarshal(%q) == %v but expected error", c, got)
		}
	}
}

func TestUnmarshalMetric(t *testing.T) {
	cases := []struct {
		input string
		want  Metric
		err   bool
	}{
		{"0", Metric(0), false},
		{"1", Metric(1), false},
		{"100", Metric(100), false},
		{"4294967295", Metric(4294967295), false},
		{"4294967296", Metric(0), true},
		{"-1", Metric(0), true},
		{"nope", Metric(0), true},
	}
	for _, tc := range cases {
		var got Metric
		err := yaml.Unmarshal([]byte(tc.input), &got)
		switch {
		case err != nil && !tc.err:
			t.Errorf("Unmarshal(%q) error:\n%+v", tc.input, err)
		case err == nil && tc.err:
			t.Errorf("Unmarshal(%q) == %d but expected error", tc.input, uint(got))
		case uint(got) != uint(tc.want):
			t.Errorf("Unmarshal(%q) == %d but expected %d", tc.input, uint(got), uint(tc.want))
		}
	}
}
