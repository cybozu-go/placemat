package placemat

import (
	"bufio"
	"context"
	"crypto/sha1"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/cybozu-go/cmd"
	"github.com/cybozu-go/log"
)

const (
	defaultOVMFCodePath = "/usr/share/OVMF/OVMF_CODE.fd"
	defaultOVMFVarsPath = "/usr/share/OVMF/OVMF_VARS.fd"
)

var vhostNetSupported bool

func init() {
	f, err := os.Open("/proc/modules")
	if err != nil {
		log.Error("failed to open /proc/modules", map[string]interface{}{
			"error": err,
		})
	}
	defer f.Close()

	s := bufio.NewScanner(f)
	for s.Scan() {
		if strings.Contains(s.Text(), "vhost_net") {
			vhostNetSupported = true
			return
		}
	}
}

// QemuProvider is an implementation of Provider interface.  It launches
// qemu-system-x86_64 as a VM engine, and qemu-img to create image.
type QemuProvider struct {
	NoGraphic bool
	RunDir    string
	Cluster   *Cluster

	dataDir    string
	imageCache *cache
}

// ImageCache returns a *cache for cloud images.
func (q *QemuProvider) ImageCache() *cache {
	return q.imageCache
}

// SetupDataDir creates directories under dataDir for later use.
func (q *QemuProvider) SetupDataDir(dataDir string) error {
	fi, err := os.Stat(dataDir)
	switch {
	case err == nil:
		if !fi.IsDir() {
			return errors.New(dataDir + " is not a directory")
		}
	case os.IsNotExist(err):
		err = os.MkdirAll(dataDir, 0755)
		if err != nil {
			return err
		}
	default:
		return err
	}

	volumeDir := filepath.Join(dataDir, "volumes")
	err = os.MkdirAll(volumeDir, 0755)
	if err != nil {
		return err
	}

	nvramDir := filepath.Join(dataDir, "nvram")
	err = os.MkdirAll(nvramDir, 0755)
	if err != nil {
		return err
	}

	imageCacheDir := filepath.Join(dataDir, "image_cache")
	err = os.MkdirAll(imageCacheDir, 0755)
	if err != nil {
		return err
	}

	q.imageCache = &cache{dir: imageCacheDir}

	q.dataDir = dataDir
	return nil
}

func execCommands(ctx context.Context, commands [][]string) error {
	for _, cmds := range commands {
		err := cmd.CommandContext(ctx, cmds[0], cmds[1:]...).Run()
		if err != nil {
			return err
		}
	}
	return nil
}

func execCommandsForce(ctx context.Context, commands [][]string) error {
	var err error
	for _, cmds := range commands {
		err = cmd.CommandContext(ctx, cmds[0], cmds[1:]...).Run()
	}
	return err
}

func createTap(ctx context.Context, tap string, network string) error {
	log.Info("Creating TAP", map[string]interface{}{"name": tap})
	cmds := [][]string{
		{"ip", "tuntap", "add", tap, "mode", "tap"},
		{"ip", "link", "set", tap, "master", network},
		{"ip", "link", "set", tap, "up"},
	}
	return execCommands(ctx, cmds)
}

func deleteTap(ctx context.Context, tap string) error {
	return cmd.CommandContext(ctx, "ip", "tuntap", "delete", tap, "mode", "tap").Run()
}

func (q QemuProvider) socketPath(host string) string {
	return filepath.Join(q.RunDir, host+".socket")
}

func (q QemuProvider) volumePath(host, name string) string {
	return filepath.Join(q.dataDir, "volumes", host+"_"+name+".img")
}

func (q QemuProvider) nvramPath(host string) string {
	return filepath.Join(q.dataDir, "nvram", host+".fd")
}

// VolumeExists checks if the volume exists
func (q QemuProvider) VolumeExists(ctx context.Context, node, vol string) (bool, error) {
	p := q.volumePath(node, vol)
	_, err := os.Stat(p)
	return !os.IsNotExist(err), nil
}

// CreateNetwork creates a bridge and iptables rules by the Network
func (q QemuProvider) CreateNetwork(ctx context.Context, nt *Network) error {
	err := createBridge(ctx, nt)
	if err != nil {
		log.Error("Failed to create a bridge", map[string]interface{}{"naem": nt.Name, "error": err})
		return err
	}
	if nt.Spec.UseNAT {
		err = createNatRules(ctx, nt)
		if err != nil {
			log.Error("Failed to create NAT rules", map[string]interface{}{"naem": nt.Name, "error": err})
			return err
		}
	}
	return nil
}

func createBridge(ctx context.Context, nt *Network) error {
	cmds := [][]string{
		{"ip", "link", "add", nt.Name, "type", "bridge"},
		{"ip", "link", "set", nt.Name, "up"},
	}
	for _, addr := range nt.Spec.Addresses {
		cmds = append(cmds,
			[]string{"ip", "addr", "add", addr, "dev", nt.Name},
		)
	}
	return execCommands(ctx, cmds)
}

func createNatRules(ctx context.Context, nt *Network) error {
	cmds := [][]string{}
	for _, iptables := range []string{"iptables", "ip6tables"} {
		cmds = append(cmds,
			[]string{iptables, "-N", "PLACEMAT", "-t", "filter"},
			[]string{iptables, "-N", "PLACEMAT", "-t", "nat"},

			[]string{iptables, "-t", "nat", "-A", "POSTROUTING", "-j", "PLACEMAT"},
			[]string{iptables, "-t", "filter", "-A", "FORWARD", "-j", "PLACEMAT"},

			[]string{iptables, "-t", "filter", "-A", "PLACEMAT", "-i", nt.Name, "-j", "ACCEPT"},
			[]string{iptables, "-t", "filter", "-A", "PLACEMAT", "-o", nt.Name, "-j", "ACCEPT"},
		)
	}

	for _, addr := range nt.Spec.Addresses {
		ip, ipNet, err := net.ParseCIDR(addr)
		if err != nil {
			return err
		}
		cmds = append(cmds,
			[]string{iptables(ip), "-t", "nat", "-A", "PLACEMAT", "-j", "MASQUERADE",
				"--source", ipNet.String(), "!", "--destination", ipNet.String()})
	}
	return execCommands(ctx, cmds)
}

func isIPv4(ip net.IP) bool {
	return ip.To4() != nil
}

func iptables(ip net.IP) string {
	if isIPv4(ip) {
		return "iptables"
	}
	return "ip6tables"
}

// DestroyNetwork destroys a bridge and iptables rules by the name
func (q QemuProvider) DestroyNetwork(ctx context.Context, nt *Network) error {
	cmds := [][]string{
		{"ip", "link", "delete", nt.Name, "type", "bridge"},
	}

	if nt.Spec.UseNAT {
		for _, iptables := range []string{"iptables", "ip6tables"} {
			cmds = append(cmds,
				[]string{iptables, "-t", "filter", "-D", "FORWARD", "-j", "PLACEMAT"},
				[]string{iptables, "-t", "nat", "-D", "POSTROUTING", "-j", "PLACEMAT"},

				[]string{iptables, "-F", "PLACEMAT", "-t", "filter"},
				[]string{iptables, "-X", "PLACEMAT", "-t", "filter"},

				[]string{iptables, "-F", "PLACEMAT", "-t", "nat"},
				[]string{iptables, "-X", "PLACEMAT", "-t", "nat"},
			)
		}
	}
	return execCommandsForce(ctx, cmds)
}

// CreateVolume creates a volume with specified options
func (q QemuProvider) CreateVolume(ctx context.Context, node string, vol Volume) error {
	vname := vol.Name()
	p := q.volumePath(node, vname)
	log.Info("Creating volume", map[string]interface{}{"node": node, "volume": vname})
	return vol.Create(ctx, p)
}

func createNVRAM(ctx context.Context, p string) error {
	_, err := os.Stat(p)
	if !os.IsNotExist(err) {
		return nil
	}
	return cmd.CommandContext(ctx, "cp", defaultOVMFVarsPath, p).Run()
}

func nodeSerial(name string) string {
	return fmt.Sprintf("%x", sha1.Sum([]byte(name)))
}

func (q QemuProvider) qemuParams(n *Node) []string {
	params := []string{"-enable-kvm"}

	if n.Spec.IgnitionFile != "" {
		params = append(params, "-fw_cfg")
		params = append(params, "opt/com.coreos/config,file="+n.Spec.IgnitionFile)
	}

	for _, br := range n.Spec.Interfaces {
		tap := n.Name + "_" + br
		netdev := "tap,id=" + br + ",ifname=" + tap + ",script=no,downscript=no"
		if vhostNetSupported {
			netdev += ",vhost=on"
		}

		params = append(params, "-netdev", netdev)
		params = append(params, "-device",
			fmt.Sprintf("virtio-net-pci,netdev=%s,romfile=,mac=%s", br, generateRandomMACForKVM()))
	}
	for _, v := range n.Spec.Volumes {
		p := q.volumePath(n.Name, v.Name())
		params = append(params, "-drive", "if=virtio,cache=none,aio=native,file="+p)
	}
	if n.Spec.Resources.CPU != "" {
		params = append(params, "-smp", n.Spec.Resources.CPU)
	}
	if n.Spec.Resources.Memory != "" {
		params = append(params, "-m", n.Spec.Resources.Memory)
	}
	if q.NoGraphic {
		p := q.socketPath(n.Name)
		defer os.Remove(p)
		params = append(params, "-nographic")
		params = append(params, "-serial", "unix:"+p+",server,nowait")
	}
	if n.Spec.BIOS == UEFI {
		p := q.nvramPath(n.Name)
		params = append(params, "-drive", "if=pflash,file="+defaultOVMFCodePath+",format=raw,readonly")
		params = append(params, "-drive", "if=pflash,file="+p+",format=raw")
	}

	smbios := "type=1"
	if n.Spec.SMBIOS.Manufacturer != "" {
		smbios += ",manufacturer=" + n.Spec.SMBIOS.Manufacturer
	}
	if n.Spec.SMBIOS.Product != "" {
		smbios += ",product=" + n.Spec.SMBIOS.Product
	}
	if n.Spec.SMBIOS.Serial != "" {
		smbios += ",serial=" + n.Spec.SMBIOS.Serial
	} else {
		smbios += ",serial=" + nodeSerial(n.Name)
	}
	params = append(params, "-smbios", smbios)
	return params
}

// StartNode starts a QEMU vm
func (q QemuProvider) StartNode(ctx context.Context, n *Node) error {
	params := q.qemuParams(n)

	for _, br := range n.Spec.Interfaces {
		tap := n.Name + "_" + br
		err := createTap(ctx, tap, br)
		if err != nil {
			return err
		}
	}
	if n.Spec.BIOS == UEFI {
		p := q.nvramPath(n.Name)
		err := createNVRAM(ctx, p)
		if err != nil {
			log.Error("Failed to create nvram", map[string]interface{}{
				"error": err,
			})
			return err
		}
	}

	log.Info("Starting VM", map[string]interface{}{"name": n.Name})
	err := cmd.CommandContext(ctx, "qemu-system-x86_64", params...).Run()
	if err != nil {
		log.Error("QEMU exited with an error", map[string]interface{}{
			"error": err,
		})
	}

	for _, br := range n.Spec.Interfaces {
		tap := n.Name + "_" + br
		err := deleteTap(context.Background(), tap)
		if err != nil {
			log.Error("Failed to delete a TAP", map[string]interface{}{
				"name":  tap,
				"error": err,
			})
		}
	}
	return err
}

func generateRandomMACForKVM() string {
	vendorPrefix := "52:54:00" // QEMU's vendor prefix
	bytes := make([]byte, 3)
	rand.Read(bytes)
	return fmt.Sprintf("%s:%02x:%02x:%02x", vendorPrefix, bytes[0], bytes[1], bytes[2])
}
