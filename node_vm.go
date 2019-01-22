package placemat

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"strings"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/well"
)

const (
	stopCommand   = "stop\n"
	resumeCommand = "cont\n"
)

// NodeVM holds resources to manage and monitor a QEMU process.
type NodeVM struct {
	cmd     *well.LogCmd
	monitor string
	running bool
	cleanup func()
}

// IsRunning returns true if the VM is running.
func (n *NodeVM) IsRunning() bool {
	return n.running
}

// PowerOn turns on the power of the VM.
func (n *NodeVM) PowerOn() error {
	if n.running {
		return nil
	}

	conn, err := net.Dial("unix", n.monitor)
	if err != nil {
		return err
	}
	defer conn.Close()
	go func() {
		io.Copy(ioutil.Discard, conn)
	}()

	_, err = io.WriteString(conn, "system_reset\ncont\n")
	if err != nil {
		return err
	}

	n.running = true
	return nil
}

// PowerOff turns off the power of the VM.
func (n *NodeVM) PowerOff() error {
	if !n.running {
		return nil
	}

	conn, err := net.Dial("unix", n.monitor)
	if err != nil {
		return err
	}
	defer conn.Close()
	go func() {
		io.Copy(ioutil.Discard, conn)
	}()

	_, err = io.WriteString(conn, "stop\n")
	if err != nil {
		return err
	}

	n.running = false
	return nil
}

var (
	qemuPrompt = []byte("\r\n(qemu) ")
	ansiCSI_K  = []byte("\x1b[K")
)

func execQEMUCommand(ctx context.Context, monitor, cmd string) (string, error) {
	var d net.Dialer
	conn, err := d.DialContext(ctx, "unix", monitor)
	if err != nil {
		return "", err
	}
	defer conn.Close()

	resp := make(chan string)
	go func() error {
		var buf bytes.Buffer
		b := make([]byte, 100)
		for {
			n, err := conn.Read(b)
			if err != nil {
				return err
			}
			buf.Write(b[:n])
			have := buf.Bytes()
			//fmt.Printf("---- %s\n", string(have))
			if bytes.HasSuffix(have, qemuPrompt) {
				buf.Reset()
				ret := bytes.TrimSuffix(have, qemuPrompt)
				if i := bytes.LastIndex(ret, ansiCSI_K); i != -1 {
					ret = ret[i+len(ansiCSI_K):]
				}
				r := strings.TrimSpace(strings.Replace(string(ret), "\r\n", "\n", -1))
				if strings.Contains(r, "monitor - type 'help' for more information") {
					buf.Reset()
					continue
				}
				resp <- r
				break
			}
		}
		return nil
	}()

	_, err = io.WriteString(conn, cmd)
	if err != nil {
		return "", err
	}
	result := <-resp
	return result, nil
}

func removeBlockDevices(ctx context.Context, monitor string, volumes []NodeVolumeSpec) error {
	for i, v := range volumes {
		if v.Kind == "localds" || v.Kind == "vvfat" {
			out, err := execQEMUCommand(ctx, monitor, fmt.Sprintf("drive_del virtio%d\n", i))
			if err != nil {
				return err
			}
			log.Info(fmt.Sprintf("monitor log: %s", out), map[string]interface{}{
				"monitor": monitor,
				"command": "drive_del",
			})
		}
	}
	return nil
}

// SaveVM saves a snapshot of the VM. To save a snapshot, localds and vvfat devices have to be detached.
// NOTE: virtio block device does not support hot add. After saving snapshot, you will no longer access mounted block device other than rootfs.
//       https://github.com/ceph/qemu-kvm/blob/de4eb6c5347e40b02dbe72cda18b58654ad11242/hw/pci-hotplug.c#L143
func (n *NodeVM) SaveVM(ctx context.Context, node *Node, tag string) error {
	if !n.running {
		return nil
	}

	// Stop VM
	out, err := execQEMUCommand(ctx, n.monitor, stopCommand)
	if err != nil {
		return err
	}
	log.Info(fmt.Sprintf("monitor log: %s", out), map[string]interface{}{
		"monitor": n.monitor,
		"command": stopCommand,
	})

	// Detach localds and vvfat
	err = removeBlockDevices(ctx, n.monitor, node.Volumes)
	if err != nil {
		return err
	}

	// Save snapshot
	out, err = execQEMUCommand(ctx, n.monitor, fmt.Sprintf("savevm %s\n", tag))
	if err != nil {
		return err
	}
	log.Info(fmt.Sprintf("monitor log: %s", out), map[string]interface{}{
		"monitor": n.monitor,
		"command": "savevm",
	})

	// Resume VM
	out, err = execQEMUCommand(ctx, n.monitor, resumeCommand)
	if err != nil {
		return err
	}
	log.Info(fmt.Sprintf("monitor log: %s", out), map[string]interface{}{
		"monitor": n.monitor,
		"command": "cont",
	})

	return nil
}

// LoadVM loads a snapshot of the VM. To load a snapshot, localds and vvfat devices have to be detached.
// NOTE: virtio block device does not support hot add. After loading snapshot, you will no longer access mounted block device other than rootfs.
//       https://github.com/ceph/qemu-kvm/blob/de4eb6c5347e40b02dbe72cda18b58654ad11242/hw/pci-hotplug.c#L143
func (n *NodeVM) LoadVM(ctx context.Context, node *Node, tag string) error {
	if !n.running {
		return nil
	}

	// Stop VM
	out, err := execQEMUCommand(ctx, n.monitor, stopCommand)
	if err != nil {
		return err
	}
	log.Info(fmt.Sprintf("monitor log: %s", out), map[string]interface{}{
		"monitor": n.monitor,
		"command": stopCommand,
	})

	// Remove block devices
	for i, v := range node.Volumes {
		if v.Kind == "localds" || v.Kind == "vvfat" {
			out, err := execQEMUCommand(ctx, n.monitor, fmt.Sprintf("drive_del virtio%d\n", i))
			if err != nil {
				return err
			}
			log.Info(fmt.Sprintf("monitor log: %s", out), map[string]interface{}{
				"monitor": n.monitor,
				"command": "drive_del",
			})
		}
	}

	// Load snapshot
	out, err = execQEMUCommand(ctx, n.monitor, fmt.Sprintf("loadvm %s\n", tag))
	if err != nil {
		return err
	}
	log.Info(fmt.Sprintf("monitor log: %s", out), map[string]interface{}{
		"monitor": n.monitor,
		"command": "loadvm",
	})

	// Resume VM
	out, err = execQEMUCommand(ctx, n.monitor, resumeCommand)
	if err != nil {
		return err
	}
	log.Info(fmt.Sprintf("monitor log: %s", out), map[string]interface{}{
		"monitor": n.monitor,
		"command": "cont",
	})

	return nil
}
