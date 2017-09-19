package helpers

import (
	"net"
	"strconv"
	"testing"
)

func TestIPNetEqual(t *testing.T) {
	cases := []string{
		"1.1.1.1/24", "1.1.1.0/24", "1.1.1.1/32",
		"0.0.0.0/0", "0.0.0.0/14",
		"2001:db8::/32", "2001:db8::/128",
		"2001:db8::caff/32", "2001:db8::caff/128",
	}
	for _, c1 := range cases {
		for _, c2 := range cases {
			i1, n1, err1 := net.ParseCIDR(c1)
			i2, n2, err2 := net.ParseCIDR(c2)
			if err1 != nil {
				panic(err1)
			}
			if err2 != nil {
				panic(err2)
			}
			n1.IP = i1
			n2.IP = i2
			got := IPNetEqual(*n1, *n2)
			expected := c1 == c2
			if got != expected {
				t.Errorf("IPNetEqual(%q,%q) == %s but expected %s",
					c1, c2,
					strconv.FormatBool(got),
					strconv.FormatBool(expected))
			}
		}
	}
}
