package placemat

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha1"
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/cybozu-go/cmd"
	"github.com/cybozu-go/log"
)

const (
	defaultOVMFCodePath = "/usr/share/OVMF/OVMF_CODE.fd"
	defaultOVMFVarsPath = "/usr/share/OVMF/OVMF_VARS.fd"

	defaultRebootTimeout = 30 * time.Second
	maxBufferSize        = 256
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

type nodeProcess struct {
	cmd *cmd.LogCmd
}

// QemuProvider is an implementation of Provider interface.  It launches
// qemu-system-x86_64 as a VM engine, and qemu-img to create image.
type QemuProvider struct {
	NoGraphic bool
	Debug     bool
	RunDir    string
	Cluster   *Cluster

	tng        nameGenerator
	vng        nameGenerator
	dataDir    string
	imageCache *cache
	dataCache  *cache
	tempDir    string

	nodeCh        chan BMCInfo
	nodeProcesses map[string]nodeProcess
}

// ImageCache implements Provier interface.
func (q *QemuProvider) ImageCache() *cache {
	return q.imageCache
}

// DataCache implements Provier interface.
func (q *QemuProvider) DataCache() *cache {
	return q.dataCache
}

// TempDir implements Provider interface.
func (q *QemuProvider) TempDir() string {
	return q.tempDir
}

// Setup initializes QemuProvider.
func (q *QemuProvider) Setup(dataDir, cacheDir string) error {
	q.tng.prefix = "pmtap"
	q.vng.prefix = "pmveth"

	err := q.setupDataDir(dataDir)
	if err != nil {
		return err
	}

	err = q.setupCacheDir(cacheDir)
	if err != nil {
		return err
	}

	q.nodeCh = make(chan BMCInfo)
	q.nodeProcesses = make(map[string]nodeProcess)

	return nil
}

func (q *QemuProvider) setupDataDir(dataDir string) error {
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

	rktDir := filepath.Join(dataDir, "rkt")
	err = os.MkdirAll(rktDir, 0755)
	if err != nil {
		return err
	}

	tempDir := filepath.Join(dataDir, "temp")
	err = os.MkdirAll(tempDir, 0755)
	if err != nil {
		return err
	}
	myTempDir, err := ioutil.TempDir(tempDir, "")
	if err != nil {
		return err
	}
	q.tempDir = myTempDir

	q.dataDir = dataDir
	return nil
}

func (q *QemuProvider) setupCacheDir(cacheDir string) error {
	fi, err := os.Stat(cacheDir)
	switch {
	case err == nil:
		if !fi.IsDir() {
			return errors.New(cacheDir + " is not a directory")
		}
	case os.IsNotExist(err):
		err = os.MkdirAll(cacheDir, 0755)
		if err != nil {
			return err
		}
	default:
		return err
	}

	imageCacheDir := filepath.Join(cacheDir, "image_cache")
	err = os.MkdirAll(imageCacheDir, 0755)
	if err != nil {
		return err
	}

	q.imageCache = &cache{dir: imageCacheDir}

	dataCacheDir := filepath.Join(cacheDir, "data_cache")
	err = os.MkdirAll(dataCacheDir, 0755)
	if err != nil {
		return err
	}

	q.dataCache = &cache{dir: dataCacheDir}

	return nil
}

func execCommands(ctx context.Context, commands [][]string) error {
	for _, cmds := range commands {
		c := cmd.CommandContext(ctx, cmds[0], cmds[1:]...)
		c.Severity = log.LvDebug
		err := c.Run()
		if err != nil {
			return err
		}
	}
	return nil
}

func execCommandsForce(ctx context.Context, commands [][]string) error {
	var firstError error
	for _, cmds := range commands {
		c := cmd.CommandContext(ctx, cmds[0], cmds[1:]...)
		c.Severity = log.LvDebug
		err := c.Run()
		if err != nil && firstError == nil {
			firstError = err
		}
	}
	return firstError
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

func createVeth(ctx context.Context, veth string, network string) error {
	log.Info("Creating VETH pair", map[string]interface{}{"name": veth})
	cmds := [][]string{
		{"ip", "link", "add", veth, "type", "veth", "peer", "name", veth + "_"},
		{"ip", "link", "set", veth + "_", "master", network, "up"},
	}
	return execCommands(ctx, cmds)
}

func deleteVeth(ctx context.Context, veth string) error {
	return cmd.CommandContext(ctx, "ip", "link", "delete", veth+"_").Run()
}

func makePodNS(ctx context.Context, pod string, veths []string, ips map[string][]string) error {
	log.Info("Creating Pod network namespace", map[string]interface{}{"pod": pod})
	ns := "pm_" + pod
	cmds := [][]string{
		{"ip", "netns", "add", ns},
		{"ip", "netns", "exec", ns, "ip", "link", "set", "lo", "up"},
		// 127.0.0.1 is auto-assigned to lo.
		//{"ip", "netns", "exec", ns, "ip", "a", "add", "127.0.0.1/8", "dev", "lo"},
	}
	for i, veth := range veths {
		eth := fmt.Sprintf("eth%d", i)
		cmds = append(cmds, []string{
			"ip", "link", "set", veth, "netns", ns, "name", eth, "up",
		})
		for _, ip := range ips[veth] {
			cmds = append(cmds, []string{
				"ip", "netns", "exec", ns, "ip", "a", "add", ip, "dev", eth,
			})
		}
	}
	return execCommands(ctx, cmds)
}

func runInPodNS(ctx context.Context, pod string, script string) error {
	return cmd.CommandContext(ctx, "ip", "netns", "exec", "pm_"+pod, script).Run()
}

func deletePodNS(ctx context.Context, pod string) error {
	return cmd.CommandContext(ctx, "ip", "netns", "del", "pm_"+pod).Run()
}

func (q *QemuProvider) socketPath(host string) string {
	return filepath.Join(q.RunDir, host+".socket")
}

func (q *QemuProvider) nvramPath(host string) string {
	return filepath.Join(q.dataDir, "nvram", host+".fd")
}

// Destroy destroys a temporary directory and network settings
func (q *QemuProvider) Destroy(c *Cluster) error {
	err := os.RemoveAll(q.tempDir)
	if err != nil {
		log.Error("failed to remove temporary directory", map[string]interface{}{
			"dir":       q.tempDir,
			log.FnError: err,
		})
	}

	for _, tap := range q.tng.GeneratedNames() {
		err := deleteTap(context.Background(), tap)
		if err != nil {
			log.Error("failed to delete a TAP", map[string]interface{}{
				"name":      tap,
				log.FnError: err,
			})
		}
	}

	for _, veth := range q.vng.GeneratedNames() {
		err := deleteVeth(context.Background(), veth)
		if err != nil {
			log.Error("failed to delete a VETH pair", map[string]interface{}{
				"name":      veth,
				log.FnError: err,
			})
		}
	}

	for _, pod := range c.Pods {
		err := deletePodNS(context.Background(), pod.Name)
		if err != nil {
			log.Error("failed to delete Pod NS", map[string]interface{}{
				"pod":       pod.Name,
				log.FnError: err,
			})
		}
	}

	for _, n := range c.Networks {
		err := q.destroyNetwork(context.Background(), n)
		if err != nil {
			log.Error("failed to destroy networks", map[string]interface{}{
				"name":  n.Name,
				"error": err,
			})
		}
	}

	return nil
}

func (q *QemuProvider) createNetwork(ctx context.Context, nt *Network) error {
	err := createBridge(ctx, nt)
	if err != nil {
		log.Error("Failed to create a bridge", map[string]interface{}{"name": nt.Name, "error": err})
		return err
	}
	if nt.Spec.UseNAT {
		err = createNatRules(ctx, nt)
		if err != nil {
			log.Error("Failed to create NAT rules", map[string]interface{}{"name": nt.Name, "error": err})
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

// destroyNetwork destroys a bridge and iptables rules by the name
func (q *QemuProvider) destroyNetwork(ctx context.Context, nt *Network) error {
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

func (q *QemuProvider) qemuParams(n *Node) []string {
	params := []string{"-enable-kvm"}

	if n.Spec.IgnitionFile != "" {
		params = append(params, "-fw_cfg")
		params = append(params, "opt/com.coreos/config,file="+n.Spec.IgnitionFile)
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
	if n.Spec.SMBIOS.Serial == "" {
		n.Spec.SMBIOS.Serial = nodeSerial(n.Name)
	}
	smbios += ",serial=" + n.Spec.SMBIOS.Serial
	params = append(params, "-smbios", smbios)
	return params
}

func (q *QemuProvider) prepareNode(ctx context.Context, n *Node) error {
	for _, vol := range n.Spec.Volumes {
		vname := vol.Name()
		log.Info("Creating volume", map[string]interface{}{"node": n.Name, "volume": vname})
		p := filepath.Join(q.dataDir, "volumes", n.Name)
		err := os.MkdirAll(p, 0755)
		if err != nil {
			return err
		}
		args, err := vol.Create(ctx, p)
		if err != nil {
			return err
		}

		n.params = append(n.params, args...)
	}
	return nil
}

func (q *QemuProvider) fetchImage(ctx context.Context, image string) error {
	log.Info("fetching image", map[string]interface{}{
		"image": image,
	})
	args := []string{
		"--pull-policy=new",
		"--insecure-options=image",
		"fetch",
		image,
	}
	return cmd.CommandContext(ctx, "rkt", args...).Run()
}

func (q *QemuProvider) preparePod(ctx context.Context, p *Pod) error {
	for _, a := range p.Apps {
		err := q.fetchImage(ctx, a.Image)
		if err != nil {
			return err
		}
	}
	return nil
}

func (q *QemuProvider) startNode(ctx context.Context, n *Node) error {
	params := append(n.params, q.qemuParams(n)...)

	for _, br := range n.Spec.Interfaces {
		tap := q.tng.New()
		err := createTap(ctx, tap, br)
		if err != nil {
			return err
		}

		netdev := "tap,id=" + br + ",ifname=" + tap + ",script=no,downscript=no"
		if vhostNetSupported {
			netdev += ",vhost=on"
		}

		params = append(params, "-netdev", netdev)

		devParams := []string{
			"virtio-net-pci",
			fmt.Sprintf("netdev=%s", br),
			fmt.Sprintf("mac=%s", generateRandomMACForKVM()),
		}
		if n.Spec.BIOS == UEFI {
			// disable iPXE boot
			devParams = append(devParams, "romfile=")
		}
		params = append(params, "-device", strings.Join(devParams, ","))
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
	params = append(params, "-boot", fmt.Sprintf("reboot-timeout=%d", int64(defaultRebootTimeout/time.Millisecond)))
	params = append(params, "-monitor", "none")
	params = append(params, "-serial", "stdio")

	log.Info("Starting VM", map[string]interface{}{"name": n.Name})
	qemuCommand := cmd.CommandContext(ctx, "qemu-system-x86_64", params...)
	w := processWriter{
		serial: n.Spec.SMBIOS.Serial,
		ch:     q.nodeCh,
	}
	qemuCommand.Stdout = &w
	if q.Debug {
		qemuCommand.Stderr = newColoredLogWriter("qemu", n.Name, os.Stderr)
	}
	err := qemuCommand.Run()
	if err != nil {
		log.Error("QEMU exited with an error", map[string]interface{}{
			"error": err,
		})
	}
	q.nodeProcesses[n.Spec.SMBIOS.Serial] = nodeProcess{
		cmd: qemuCommand,
	}

	return err
}

// BMCInfo represents BMC information notified by a guest VM.
type BMCInfo struct {
	serial     string
	bmcAddress string
}

type processWriter struct {
	data   []byte
	serial string
	ch     chan<- BMCInfo
}

func (w *processWriter) Write(p []byte) (n int, err error) {
	n = len(p)
	w.data = append(w.data, p...)
	if len(w.data) > maxBufferSize {
		log.Warn("discard data received from guest VM, because it is too large.", nil)
		w.data = nil
		return
	}
	index := bytes.IndexByte(w.data, '\n')

	switch index {
	case 0:
		w.data = w.data[1:]
		fallthrough
	case -1:
		return
	}
	bmcAddress := string(w.data[:index])
	w.data = w.data[index+1:]
	w.ch <- BMCInfo{
		serial:     w.serial,
		bmcAddress: bmcAddress,
	}
	return
}

func generateRandomMACForKVM() string {
	vendorPrefix := "52:54:00" // QEMU's vendor prefix
	bytes := make([]byte, 3)
	rand.Read(bytes)
	return fmt.Sprintf("%s:%02x:%02x:%02x", vendorPrefix, bytes[0], bytes[1], bytes[2])
}

func (q *QemuProvider) startPod(ctx context.Context, p *Pod, root string) error {
	veths := make([]string, len(p.Interfaces))
	ips := make(map[string][]string)
	for i, iface := range p.Interfaces {
		veth := q.vng.New()
		err := createVeth(ctx, veth, iface.NetworkName)
		if err != nil {
			return err
		}
		veths[i] = veth
		ips[veth] = iface.Addresses
	}

	err := makePodNS(ctx, p.Name, veths, ips)
	if err != nil {
		return err
	}

	for _, script := range p.InitScripts {
		err := runInPodNS(ctx, p.Name, script)
		if err != nil {
			return err
		}
	}

	params := []string{
		"--insecure-options=all-run",
		"run",
		"--net=host",
		"--dns=host",
	}
	params = p.appendParams(params)

	log.Info("rkt run", map[string]interface{}{"name": p.Name, "params": params})
	args := []string{
		"netns", "exec", "pm_" + p.Name, "chroot", root, "rkt",
	}
	args = append(args, params...)
	rkt := exec.Command("ip", args...)
	rkt.Stdout = newColoredLogWriter("rkt", p.Name, os.Stdout)
	rkt.Stderr = newColoredLogWriter("rkt", p.Name, os.Stderr)
	err = rkt.Start()
	if err != nil {
		log.Error("failed to start rkt", map[string]interface{}{
			log.FnError: err,
		})
		return err
	}

	go func() {
		<-ctx.Done()
		rkt.Process.Signal(syscall.SIGTERM)
	}()
	return rkt.Wait()
}

// Start implements Provider interface.
func (q *QemuProvider) Start(ctx context.Context, c *Cluster) error {
	root, err := NewRootfs()
	if err != nil {
		return err
	}
	defer root.Destroy()

	for _, n := range c.Networks {
		log.Info("Creating network", map[string]interface{}{"name": n.Name})
		err := q.createNetwork(ctx, n)
		if err != nil {
			return err
		}
	}

	for _, df := range c.DataFolders {
		log.Info("initializing data folder", map[string]interface{}{
			"name": df.Name,
		})
		err := df.setup(ctx)
		if err != nil {
			return err
		}
	}

	nodes := c.NodesFromNodeSets()
	nodes = append(nodes, c.Nodes...)
	for _, n := range nodes {
		err := q.prepareNode(ctx, n)
		if err != nil {
			return err
		}
	}

	for _, p := range c.Pods {
		err := q.preparePod(ctx, p)
		if err != nil {
			return err
		}
	}

	env := cmd.NewEnvironment(ctx)

	env.Go(func(ctx context.Context) error {
		for {
			select {
			case info := <-q.nodeCh:
				fmt.Printf("============================%v\n", info)
			case <-ctx.Done():
				return nil
			}
		}
		return nil
	})
	for _, n := range nodes {
		node := n
		env.Go(func(ctx context.Context) error {
			return q.startNode(ctx, node)
		})
	}
	for _, p := range c.Pods {
		pod := p
		env.Go(func(ctx context.Context) error {
			return q.startPod(ctx, pod, root.Path())
		})
	}
	env.Stop()
	return env.Wait()
}
