package vm

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/placemat/v2/pkg/types"
	"github.com/cybozu-go/placemat/v2/pkg/util"
	"github.com/cybozu-go/placemat/v2/pkg/virtualbmc"
	"github.com/cybozu-go/well"
)

// Node represents a virtual machine.
type Node interface {
	// Prepare initializes node volumes
	Prepare(context.Context, *util.Cache) error
	// Setup creates volumes and taps, and then run a virtual machine as a QEMU process
	Setup(context.Context, *Runtime, int, chan<- BMCInfo) (VM, string, error)
	// Taps returns Tap information
	Taps() map[string]string
	// Cleanup removes taps placemat added
	Cleanup()
	// CleanupGarbage cleanups all garbage
	CleanupGarbage(*Runtime)
}

type node struct {
	name         string
	taps         []*tap
	volumes      []nodeVolume
	ignitionFile string
	cpu          int
	memory       string
	uefi         bool
	tpm          bool
	smbios       smBIOSConfig
}

type smBIOSConfig struct {
	manufacturer string
	product      string
	serial       string
}

// NewNode creates a Node from spec.
func NewNode(spec *types.NodeSpec, imageSpecs []*types.ImageSpec) (Node, error) {
	n := &node{
		name:         spec.Name,
		ignitionFile: spec.IgnitionFile,
		cpu:          spec.CPU,
		memory:       spec.Memory,
		uefi:         spec.UEFI,
		tpm:          spec.TPM,
		smbios: smBIOSConfig{
			manufacturer: spec.SMBIOS.Manufacturer,
			product:      spec.SMBIOS.Product,
			serial:       spec.SMBIOS.Serial,
		},
	}

	for _, v := range spec.Volumes {
		vol, err := newNodeVolume(v, imageSpecs)
		if err != nil {
			return nil, fmt.Errorf("failed to create the node volume %s: %w", v.Name, err)
		}
		n.volumes = append(n.volumes, vol)
	}

	for _, i := range spec.Interfaces {
		tap, err := newTap(i)
		if err != nil {
			return nil, fmt.Errorf("failed to new type tap: bridge is %s: %w", i, err)
		}
		n.taps = append(n.taps, tap)
	}

	return n, nil
}

func (n *node) Prepare(ctx context.Context, c *util.Cache) error {
	for _, v := range n.volumes {
		if err := v.prepare(ctx, c); err != nil {
			return err
		}
	}

	return nil
}

func (n *node) Setup(ctx context.Context, r *Runtime, mtu int, nodeCh chan<- BMCInfo) (VM, string, error) {
	if r.Force {
		n.CleanupGarbage(r)
	}

	if n.tpm {
		err := n.startSWTPM(ctx, r)
		if err != nil {
			return nil, "", err
		}
	}
	vArgs, err := n.createVolumes(ctx, r.DataDir)
	if err != nil {
		return nil, "", err
	}

	tapInfos, err := n.createTaps(mtu)
	if err != nil {
		return nil, "", err
	}

	if n.uefi {
		p := r.nvramPath(n.name)
		err := createNVRAM(ctx, p)
		if err != nil {
			log.Error("Failed to create nvram", map[string]interface{}{
				"error": err,
			})
			return nil, "", err
		}
	}

	qemu := newQemu(n.name, tapInfos, vArgs, n.ignitionFile, n.cpu, n.memory, n.uefi, n.tpm, n.smbios)
	c := qemu.command(r)
	qemuCommand := well.CommandContext(ctx, c[0], c[1:]...)
	qemuCommand.Stdout = util.NewColoredLogWriter("qemu", n.name, os.Stdout)
	qemuCommand.Stderr = util.NewColoredLogWriter("qemu", n.name, os.Stderr)

	if err := qemuCommand.Start(); err != nil {
		return nil, "", fmt.Errorf("failed to start qemuCommand: %w", err)
	}

	guest := r.guestSocketPath(n.name)
	monitor := r.monitorSocketPath(n.name)
	for {
		_, err := os.Stat(monitor)
		if err != nil && !os.IsNotExist(err) {
			return nil, "", err
		}

		_, err2 := os.Stat(guest)
		if err2 != nil && !os.IsNotExist(err2) {
			return nil, "", err2
		}

		if err == nil && err2 == nil {
			break
		}

		select {
		case <-time.After(100 * time.Millisecond):
		case <-ctx.Done():
			return nil, "", nil
		}
	}

	connGuest, err := net.Dial("unix", guest)
	if err != nil {
		return nil, "", err
	}
	gc := &guestConnection{
		serial: n.smbios.serial,
		guest:  connGuest,
		ch:     nodeCh,
	}
	go gc.handle()

	vm := &vm{
		cmd:         qemuCommand,
		powerStatus: virtualbmc.PowerStatusOn,
		monitor:     monitor,
		connGuest:   connGuest,
		guest:       guest,
		socket:      r.socketPath(n.name),
		swtpm:       r.swtpmSocketPath(n.name),
	}

	return vm, n.smbios.serial, nil
}

func (n *node) createVolumes(ctx context.Context, dataDir string) ([]volumeArgs, error) {
	volumePath := filepath.Join(dataDir, "volumes", n.name)
	if err := os.MkdirAll(volumePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to make the directory %s: %w", volumePath, err)
	}
	var argsList []volumeArgs
	for _, v := range n.volumes {
		args, err := v.create(ctx, volumePath)
		if err != nil {
			return nil, fmt.Errorf("failed to create the volume: %w", err)
		}
		argsList = append(argsList, args)
	}

	return argsList, nil
}

func (n *node) createTaps(mtu int) ([]*tapInfo, error) {
	var tapInfos []*tapInfo
	for _, tap := range n.taps {
		tapInfo, err := tap.create(mtu)
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

func (n *node) startSWTPM(ctx context.Context, r *Runtime) error {
	err := os.Mkdir(r.swtpmSocketDirPath(n.name), 0755)
	if err != nil {
		return err
	}

	log.Info("Starting swtpm for node", map[string]interface{}{
		"name":   n.name,
		"socket": r.swtpmSocketPath(n.name),
	})
	c := well.CommandContext(ctx, "swtpm", "socket",
		"--tpmstate", "dir="+r.swtpmSocketDirPath(n.name),
		"--tpm2",
		"--ctrl",
		"type=unixio,path="+r.swtpmSocketPath(n.name),
	)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	err = c.Start()
	if err != nil {
		return err
	}

	for {
		_, err := os.Stat(r.swtpmSocketPath(n.name))
		if err != nil && !os.IsNotExist(err) {
			return err
		}
		if err == nil {
			break
		}

		select {
		case <-time.After(100 * time.Millisecond):
		case <-ctx.Done():
			return nil
		}
	}

	return nil
}

func (n *node) Taps() map[string]string {
	var taps = make(map[string]string)
	for _, tap := range n.taps {
		taps[tap.bridge.Attrs().Name] = tap.tapName
	}

	return taps
}

func (n *node) Cleanup() {
	for _, tap := range n.taps {
		tap.Cleanup()
	}
}

func (n *node) CleanupGarbage(r *Runtime) {
	files := []string{
		r.guestSocketPath(n.name),
		r.monitorSocketPath(n.name),
		r.socketPath(n.name),
	}
	for _, f := range files {
		_, err := os.Stat(f)
		if err == nil {
			err = os.Remove(f)
			if err != nil {
				log.Warn("failed to clean", map[string]interface{}{
					"filename":  f,
					log.FnError: err,
				})
			}
		}
	}
	dir := r.swtpmSocketDirPath(n.name)
	_, err := os.Stat(dir)
	if err == nil {
		err = os.RemoveAll(dir)
		if err != nil {
			log.Warn("failed to clean", map[string]interface{}{
				"directory": dir,
				log.FnError: err,
			})
		}
	}
}

type VM interface {
	virtualbmc.Machine
	// Wait waits until VM process exits
	Wait() error
	// SocketPath returns socket path
	SocketPath() string
	// Cleanup remove all socket files created by the VM
	Cleanup()
}

type vm struct {
	cmd         *well.LogCmd
	powerStatus virtualbmc.PowerStatus
	monitor     string
	connGuest   net.Conn
	guest       string
	socket      string
	swtpm       string
	mu          sync.Mutex
}

func (n *vm) PowerStatus() virtualbmc.PowerStatus {
	return n.powerStatus
}

func (n *vm) PowerOn() error {
	n.mu.Lock()
	err := n.powerOn()
	n.mu.Unlock()
	return err
}

func (n *vm) powerOn() error {
	if n.powerStatus == virtualbmc.PowerStatusOn || n.powerStatus == virtualbmc.PowerStatusPoweringOn {
		return nil
	}

	n.powerStatus = virtualbmc.PowerStatusPoweringOn

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

	n.powerStatus = virtualbmc.PowerStatusOn
	return nil
}

func (n *vm) PowerOff() error {
	n.mu.Lock()
	err := n.powerOff()
	n.mu.Unlock()
	return err
}

func (n *vm) powerOff() error {
	if n.powerStatus == virtualbmc.PowerStatusOff || n.powerStatus == virtualbmc.PowerStatusPoweringOff {
		return nil
	}

	n.powerStatus = virtualbmc.PowerStatusPoweringOff

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

	n.powerStatus = virtualbmc.PowerStatusOff
	return nil
}

func (n *vm) Wait() error {
	return n.cmd.Wait()
}

func (n *vm) SocketPath() string {
	return n.socket
}

func (n *vm) Cleanup() {
	if err := n.connGuest.Close(); err != nil {
		log.Warn("failed to close guest connection", map[string]interface{}{
			log.FnError: err,
		})
	}
	if err := os.Remove(n.guest); err != nil {
		log.Warn("failed to remove guest socket", map[string]interface{}{
			log.FnError: err,
		})
	}
	if err := os.Remove(n.monitor); err != nil {
		log.Warn("failed to remove monitor socket", map[string]interface{}{
			log.FnError: err,
		})
	}
	if err := os.Remove(n.socket); err != nil {
		log.Warn("failed to remove socket", map[string]interface{}{
			log.FnError: err,
		})
	}
	if err := os.RemoveAll(n.swtpm); err != nil {
		log.Warn("failed to remove swtpm socket", map[string]interface{}{
			log.FnError: err,
		})
	}
}
