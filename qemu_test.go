package placemat

import (
	"net"
	"strings"
	"testing"
)

func TestGenerateRandomMacForKVM(t *testing.T) {
	sut := generateRandomMACForKVM()
	if len(sut) != 17 {
		t.Fatal("length of MAC address string is not 17")
	}
	if sut == generateRandomMACForKVM() {
		t.Fatal("it should generate unique address")
	}
	_, err := net.ParseMAC(sut)
	if err != nil {
		t.Fatal("invalid MAC address", err)
	}

}

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

func TestStartNodeCmdParams(t *testing.T) {
	systemVol := NewImageVolume("system", RecreateIfNotPresent, "ubuntu-image", false)
	dataVol := NewRawVolume("data", RecreateAlways, "10GB")

	cases := []struct {
		n    Node
		opts [][]string
	}{
		{
			Node{
				Name: "boot",
				Spec: NodeSpec{
					Interfaces: []string{"net1"},
					SMBIOS: SMBIOSSpec{
						Manufacturer: "cybozu",
						Product:      "mk2",
						Serial:       "1234abcd",
					},
					Volumes: []Volume{
						systemVol,
						dataVol,
					},
					Resources: ResourceSpec{
						CPU:    "2",
						Memory: "2G",
					},
					BIOS: UEFI,
				},
			},
			[][]string{
				{"-smbios", "type=1,manufacturer=cybozu,product=mk2,serial=1234abcd"},
				{"-smp", "2"},
				{"-m", "2G"},
				{"-drive", "if=pflash,file=" + defaultOVMFCodePath + ",format=raw,readonly"},
				{"-drive", "if=pflash,file=/tmp/nvram/boot.fd,format=raw"},
			},
		},
		{
			Node{
				Name: "worker",
			},
			[][]string{
				{"-smbios", "type=1,serial=" + nodeSerial("worker")},
			},
		},
	}

	vhostNetSupported = true
	q := QemuProvider{NoGraphic: false, dataDir: "/tmp"}

	for _, c := range cases {
		params := q.qemuParams(&c.n)
		paramsZero := strings.Join(params, "\x00")
		for _, o := range c.opts {
			optZero := strings.Join(o, "\x00")
			if !strings.Contains(paramsZero, optZero) {
				t.Fatalf("%v does not contains %v", params, o)
			}
		}
	}

}
