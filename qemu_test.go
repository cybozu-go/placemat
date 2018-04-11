package placemat

import (
	"context"
	"io/ioutil"
	"net"
	"os"
	"strings"
	"testing"
)

func touch(path string) error {
	f, err := os.OpenFile(path, os.O_RDONLY|os.O_CREATE, 0666)
	if err == nil {
		return err
	}
	return f.Close()
}

func TestVolumeExists(t *testing.T) {
	dir, err := ioutil.TempDir("", t.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	qemu := QemuProvider{}
	err = qemu.SetupDataDir(dir)
	if err != nil {
		t.Fatal(err)
	}

	touch(qemu.volumePath("host1", "volume1"))
	exists, err := qemu.VolumeExists(context.Background(), "host1", "volume1")
	if err != nil {
		t.Fatal("expected err != nil, ", err)
	}
	if !exists {
		t.Fatal("expected exists")
	}

	exists, err = qemu.VolumeExists(context.Background(), "host1", "volume2")
	if err != nil {
		t.Fatal("expected err != nil, ", err)
	}
	if exists {
		t.Fatal("expected not exists")
	}
}

func TestCreateVolume(t *testing.T) {
	dir, err := ioutil.TempDir("", t.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	qemu := QemuProvider{}
	err = qemu.SetupDataDir(dir)
	if err != nil {
		t.Fatal(err)
	}

	err = qemu.CreateVolume(context.Background(), "host1", NewRawVolume("volume1", RecreateNever, "10G"))
	if err != nil {
		t.Fatal("expected err != nil", err)
	}

	_, err = os.Stat(qemu.volumePath("host1", "volume1"))
	if os.IsNotExist(err) {
		t.Fatal("expected !os.IsNotExist(err), ", err)
	}
}

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
	systemVol := NewImageVolume("system", RecreateIfNotPresent, "ubuntu-image")
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
				{"-netdev", "tap,id=net1,ifname=boot_net1,script=no,downscript=no,vhost=on"},
				{"-smbios", "type=1,manufacturer=cybozu,product=mk2,serial=1234abcd"},
				{"-drive", "if=virtio,cache=none,aio=native,file=/tmp/volumes/boot_system.img"},
				{"-drive", "if=virtio,cache=none,aio=native,file=/tmp/volumes/boot_data.img"},
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
	q := QemuProvider{NoGraphic: false}
	err := q.SetupDataDir("/tmp")
	if err != nil {
		t.Fatal(err)
	}

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
