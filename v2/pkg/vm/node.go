package vm

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/placemat/v2/pkg/types"
	"github.com/cybozu-go/placemat/v2/pkg/util"
	"github.com/cybozu-go/well"
)

// Node represents a virtual machine.
type Node struct {
	name         string
	taps         []*Tap
	volumes      []NodeVolume
	ignitionFile string
	cpu          int
	memory       string
	uefi         bool
	tpm          bool
	smbios       SMBIOSConfig
}

type SMBIOSConfig struct {
	manufacturer string
	product      string
	serial       string
}

// NewNode creates a Node from spec.
func NewNode(spec *types.NodeSpec, imageSpecs []*types.ImageSpec) (*Node, error) {
	n := &Node{
		name:         spec.Name,
		ignitionFile: spec.IgnitionFile,
		cpu:          spec.CPU,
		memory:       spec.Memory,
		uefi:         spec.UEFI,
		tpm:          spec.TPM,
		smbios: SMBIOSConfig{
			manufacturer: spec.SMBIOS.Manufacturer,
			product:      spec.SMBIOS.Product,
			serial:       spec.SMBIOS.Serial,
		},
	}

	for _, v := range spec.Volumes {
		vol, err := NewNodeVolume(v, imageSpecs)
		if err != nil {
			return nil, fmt.Errorf("failed to create the node volume %s: %w", v.Name, err)
		}
		n.volumes = append(n.volumes, vol)
	}

	for _, i := range spec.Interfaces {
		tap, err := NewTap(i)
		if err != nil {
			return nil, fmt.Errorf("failed to new type tap: bridge is %s: %w", i, err)
		}
		n.taps = append(n.taps, tap)
	}

	return n, nil
}

// Setup creates volumes and taps, and then run a virtual machine as a QEMU process
func (n *Node) Setup(ctx context.Context, r *Runtime, mtu int) (*VM, error) {
	vArgs, err := n.createVolumes(ctx, r.dataDir)
	if err != nil {
		return nil, err
	}

	tapInfos, err := n.createTaps(mtu)
	if err != nil {
		return nil, err
	}

	if n.uefi {
		p := r.nvramPath(n.name)
		err := createNVRAM(ctx, p)
		if err != nil {
			log.Error("Failed to create nvram", map[string]interface{}{
				"error": err,
			})
			return nil, err
		}
	}

	qemu := NewQemu(n.name, tapInfos, vArgs, n.ignitionFile, n.cpu, n.memory, n.uefi, n.tpm, n.smbios)
	c := qemu.Command(r)
	qemuCommand := well.CommandContext(ctx, c[0], c[1:]...)
	qemuCommand.Stdout = util.NewColoredLogWriter("qemu", n.name, os.Stdout)
	qemuCommand.Stderr = util.NewColoredLogWriter("qemu", n.name, os.Stderr)

	if err := qemuCommand.Start(); err != nil {
		return nil, fmt.Errorf("failed to start qemuCommand: %w", err)
	}

	guest := r.guestSocketPath(n.name)
	monitor := r.monitorSocketPath(n.name)
	for {
		_, err := os.Stat(monitor)
		if err != nil && !os.IsNotExist(err) {
			return nil, err
		}

		_, err2 := os.Stat(guest)
		if err2 != nil && !os.IsNotExist(err2) {
			return nil, err2
		}

		if err == nil && err2 == nil {
			break
		}

		select {
		case <-time.After(100 * time.Millisecond):
		case <-ctx.Done():
			return nil, nil
		}
	}

	vm := &VM{
		cmd:     qemuCommand,
		running: true,
		monitor: monitor,
		guest:   guest,
		socket:  r.socketPath(n.name),
		swtpm:   r.swtpmSocketPath(n.name),
	}

	return vm, nil
}

func (n *Node) createVolumes(ctx context.Context, dataDir string) ([]VolumeArgs, error) {
	volumePath := filepath.Join(dataDir, "volumes", n.name)
	if err := os.MkdirAll(volumePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to make the directory %s: %w", volumePath, err)
	}
	var argsList []VolumeArgs
	for _, v := range n.volumes {
		args, err := v.Create(ctx, dataDir)
		if err != nil {
			return nil, fmt.Errorf("failed to create the volume: %w", err)
		}
		argsList = append(argsList, args)
	}

	return argsList, nil
}

func (n *Node) createTaps(mtu int) ([]*TapInfo, error) {
	var tapInfos []*TapInfo
	for _, tap := range n.taps {
		tapInfo, err := tap.Create(mtu)
		if err != nil {
			return nil, fmt.Errorf("failed to create the tap: %w", err)
		}

		tapInfos = append(tapInfos, tapInfo)
	}

	return tapInfos, nil
}

func createNVRAM(ctx context.Context, p string) error {
	_, err := os.Stat(p)
	if !os.IsNotExist(err) {
		return nil
	}
	return well.CommandContext(ctx, "cp", defaultOVMFVarsPath, p).Run()
}

// Cleanup
func (n *Node) Cleanup() {
	for _, tap := range n.taps {
		tap.Cleanup()
	}
}

// VM holds resources to manage and monitor a QEMU process.
type VM struct {
	cmd     *well.LogCmd
	running bool
	monitor string
	guest   string
	socket  string
	swtpm   string
}

// IsRunning returns true if the VM is running.
func (n *VM) IsRunning() bool {
	return n.running
}

// PowerOn turns on the power of the VM.
func (n *VM) PowerOn() error {
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
func (n *VM) PowerOff() error {
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

// Cleanup remove all socket files created by the VM
func (n *VM) Cleanup() {
	os.Remove(n.guest)
	os.Remove(n.monitor)
	os.Remove(n.socket)
	os.RemoveAll(n.swtpm)
}
