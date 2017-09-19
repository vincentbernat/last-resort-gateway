package helpers

import (
	"net"
)

// IPNetEqual returns true iff both IPNet are equals
func IPNetEqual(ipn1 net.IPNet, ipn2 net.IPNet) bool {
	m1, _ := ipn1.Mask.Size()
	m2, _ := ipn2.Mask.Size()
	return m1 == m2 && ipn1.IP.Equal(ipn2.IP)
}
