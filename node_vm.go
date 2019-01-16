package placemat

import (
	"fmt"
	"io"
	"net"

	"github.com/cybozu-go/well"
)

// NodeVM holds resources to manage and monitor a QEMU process.
type NodeVM struct {
	cmd     *well.LogCmd
	monitor net.Conn
	running bool
	cleanup func()
}

// IsRunning returns true if the VM is running.
func (n *NodeVM) IsRunning() bool {
	return n.running
}

// PowerOn turns on the power of the VM.
func (n *NodeVM) PowerOn() {
	if n.running {
		return
	}

	io.WriteString(n.monitor, "system_reset\ncont\n")
	n.running = true
}

// PowerOff turns off the power of the VM.
func (n *NodeVM) PowerOff() {
	if !n.running {
		return
	}

	io.WriteString(n.monitor, "stop\n")
	n.running = false
}

func (n *NodeVM) SaveVM(node *Node, tag string) {
	io.WriteString(n.monitor, "stop\n")
	for i, v := range node.Volumes {
		if v.Kind == "localds" || v.Kind == "vvfat" {
			io.WriteString(n.monitor, fmt.Sprintf("drive_del virtio%d\n", i))
		}
	}
	io.WriteString(n.monitor, fmt.Sprintf("savevm %s\n", tag))
	io.WriteString(n.monitor, "cont\n")
}

func (n *NodeVM) LoadVM(node *Node, tag string) {
	io.WriteString(n.monitor, "stop\n")
	for i, v := range node.Volumes {
		if v.Kind == "localds" || v.Kind == "vvfat" {
			io.WriteString(n.monitor, fmt.Sprintf("drive_del virtio%d\n", i))
		}
	}
	io.WriteString(n.monitor, fmt.Sprintf("loadvm %s\n", tag))
	io.WriteString(n.monitor, "cont\n")
}
