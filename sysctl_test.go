// build +linux

package placemat

import (
	"os"
	"testing"
)

func testSysctlGet(t *testing.T) {
	t.Parallel()

	val, err := sysctlGet("kernel.version")
	if err != nil {
		t.Fatal(err)
	}

	t.Log(val)
}

func testSysctlSet(t *testing.T) {
	if os.Getenv("TEST_SYSCTL_SET") == "" {
		t.Skip("no TEST_SYSCTL_SET envvar")
	}

	t.Parallel()

	err := sysctlSet("net.ipv4.ip_forward", "0")
	if err != nil {
		t.Fatal(err)
	}

	val, err := sysctlGet("net.ipv4.ip_forward")
	if err != nil {
		t.Fatal(err)
	}

	if val != "0\n" {
		t.Error("net.ipv4.ip_forward is not 0")
	}
}

func TestSysctl(t *testing.T) {
	t.Run("Get", testSysctlGet)
	t.Run("Set", testSysctlSet)
}
