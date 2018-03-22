package placemat

import (
	"context"
	"io/ioutil"
	"net"
	"os"
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

	var qemu = QemuProvider{BaseDir: dir}

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

	var qemu = QemuProvider{BaseDir: dir}

	err = qemu.CreateVolume(context.Background(), "host1", &VolumeSpec{Name: "volume1", Size: "10G"})
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

func TestStartNode(t *testing.T) {
	// TODO add tests
}
