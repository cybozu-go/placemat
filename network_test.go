package placemat

import (
	"net"
	"testing"
)

func TestIptables(t *testing.T) {
	ip := net.ParseIP("172.16.0.1")
	sut := iptables(ip)
	if sut != "iptables" {
		t.Fatal("expected is 'iptables', but actual is ", sut)
	}

	ip6 := net.ParseIP("2001:db8:85a3:0:0:8a2e:370:7334")
	sut6 := iptables(ip6)
	if sut6 != "ip6tables" {
		t.Fatal("expected is 'ip6tables', but actual is ", sut6)
	}
}
