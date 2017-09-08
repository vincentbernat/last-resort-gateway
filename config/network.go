package config

import (
	"fmt"
	"net"

	"github.com/pkg/errors"
)

// Addr represents an endpoint (eg. 127.0.0.1:1654).
type Addr string

func (a Addr) String() string {
	return string(a)
}

// UnmarshalText parses text as an address of the form "host:port" or
// "[ipv6-host%zone]:port" and resolves a pair of domain name and port
// name. A literal address or host name for IPv6 must be enclosed in
// square brackets, as in "[::1]:80", "[ipv6-host]:http" or
// "[ipv6-host%zone]:80". Resolution of names is biased to IPv4.
func (a *Addr) UnmarshalText(text []byte) error {
	rawAddr := string(text)
	addr, err := net.ResolveTCPAddr("tcp", rawAddr)
	if err != nil {
		return errors.Wrapf(err, "unable to solve %q", rawAddr)
	}
	*a = Addr(addr.String())
	return nil
}

// Prefix represents an IP subnet.
type Prefix net.IPNet

// UnmarshalText parses and validates a Prefix. Non-pure subnets are
// not accepted (like 192.168.1.1/24).
func (s *Prefix) UnmarshalText(text []byte) error {
	rawPrefix := string(text)
	ip, ipnet, err := net.ParseCIDR(rawPrefix)
	if err != nil {
		return errors.Wrap(err, "not an IP subnet")
	}
	if !ip.Equal(ipnet.IP) {
		return errors.New("not an IP subnet")
	}
	*s = Prefix(*ipnet)
	return nil
}

func (s Prefix) String() string {
	ipnet := net.IPNet(s)
	return ipnet.String()
}

// Metric represents a route metric
type Metric uint

// UnmarshalYAML parses and validates a metric. It's a 32bit unsigned int.
func (m *Metric) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var rawUint uint32
	if err := unmarshal(&rawUint); err != nil {
		return errors.Wrap(err, "not a valid metric")
	}
	*m = Metric(rawUint)
	return nil
}

func (m Metric) String() string {
	return fmt.Sprintf("%d", uint(m))
}
