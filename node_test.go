package placemat

import (
	"net"
	"testing"
)

func TestGenerateRandomMacForKVM(t *testing.T) {
	sut := generateMACForKVM("test")
	if len(sut) != 17 {
		t.Fatal("length of MAC address string is not 17")
	}
	if sut == generateMACForKVM("hoge") {
		t.Fatal("it should generate unique address")
	}
	_, err := net.ParseMAC(sut)
	if err != nil {
		t.Fatal("invalid MAC address", err)
	}
}
