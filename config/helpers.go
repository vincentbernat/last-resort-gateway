// +build !release

package config

import "net"

// MustParseCIDR will parse a CIDR and panic if there is a problem.
func MustParseCIDR(s string) *net.IPNet {
	_, parsed, err := net.ParseCIDR(s)
	if err != nil {
		panic(err)
	}
	return parsed
}

// MustParsePrefix will parse a CIDR and panic if there is a problem.
func MustParsePrefix(s string) Prefix {
	return Prefix(*MustParseCIDR(s))
}
