package placemat

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

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
	BaseDir string

	NoGraphic bool
	RunDir    string
}

func createTap(ctx context.Context, tap string, network string) error {
	log.Info("Creating TAP", map[string]interface{}{"name": tap})
	err := cmd.CommandContext(ctx, "ip", "tuntap", "add", tap, "mode", "tap").Run()
	if err != nil {
		return err
	}
	err = cmd.CommandContext(ctx, "ip", "link", "set", tap, "master", network).Run()
	if err != nil {
		return err
	}
	err = cmd.CommandContext(ctx, "ip", "link", "set", tap, "master", network).Run()
	if err != nil {
		return err
	}
	return cmd.CommandContext(ctx, "ip", "link", "set", tap, "up").Run()
}

func deleteTap(ctx context.Context, tap string) error {
	return cmd.CommandContext(ctx, "ip", "tuntap", "delete", tap, "mode", "tap").Run()
}

func (q QemuProvider) socketPath(host string) string {
	return path.Join(q.RunDir, host+".socket")
}

func (q QemuProvider) volumePath(host, name string) string {
	return path.Join(q.BaseDir, host+"_"+name+".img")
}

func (q QemuProvider) nvramPath(host string) string {
	return path.Join(q.BaseDir, host+"_nvram.fd")
}

// VolumeExists checks if the volume exists
func (q QemuProvider) VolumeExists(ctx context.Context, node, vol string) (bool, error) {
	p := q.volumePath(node, vol)
	_, err := os.Stat(p)
	return !os.IsNotExist(err), nil
}

// CreateNetwork creates a bridge by the Network
func (q QemuProvider) CreateNetwork(ctx context.Context, net *Network) error {
	log.Info("Creating network", map[string]interface{}{"name": net.Name, "spec": net})
	err := cmd.CommandContext(ctx, "ip", "link", "add", net.Name, "type", "bridge").Run()
	if err != nil {
		return err
	}
	err = cmd.CommandContext(ctx, "ip", "link", "set", net.Name, "up").Run()
	if err != nil {
		return err
	}
	for _, addr := range net.Spec.Addresses {
		err := cmd.CommandContext(ctx, "ip", "addr", "add", addr, "dev", net.Name).Run()
		if err != nil {
			return err
		}
	}
	return nil
}

// DestroyNetwork destroys a bridge by the name
func (q QemuProvider) DestroyNetwork(ctx context.Context, name string) error {
	c := cmd.CommandContext(ctx, "ip", "link", "delete", name, "type", "bridge")
	log.Info("Destroying network", map[string]interface{}{"name": name})
	return c.Run()
}

func createEmptyVolume(ctx context.Context, p string, size string) error {
	c := cmd.CommandContext(ctx, "qemu-img", "create", "-f", "qcow2", p, size)
	return c.Run()
}

func showDownloadProgress(ctx context.Context, totalSize int, fileName string) error {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			stat, err := os.Stat(fileName)
			if err != nil {
				log.Error("Failed to get file statistics", map[string]interface{}{"file": fileName, "error": err})
				continue
			}
			var progress = fmt.Sprintf("%.1f%%", float64(stat.Size())/float64(totalSize)*100)

			log.Info("Downloading...", map[string]interface{}{"file_name": fileName, "current_size": stat.Size(), "total_size": totalSize, "progress": progress})
		}
	}
	return nil
}

func createVolumeFromURL(ctx context.Context, path string, url string) error {
	dir := filepath.Dir(path)
	temp, err := ioutil.TempFile(dir, "temp-placemat-image-")
	if err != nil {
		return err
	}
	defer temp.Close()

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req = req.WithContext(ctx)

	client := &cmd.HTTPClient{
		Client:   &http.Client{},
		Severity: log.LvDebug,
	}
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download: %s: %s", res.Status, url)
	}

	size, err := strconv.Atoi(res.Header.Get("Content-Length"))
	if err != nil {
		return err
	}
	env := cmd.NewEnvironment(ctx)
	env.Go(func(ctx context.Context) error {
		return showDownloadProgress(ctx, size, temp.Name())
	})
	defer env.Cancel(nil)

	_, err = io.Copy(temp, res.Body)
	if err != nil {
		return err
	}
	err = temp.Close()
	if err != nil {
		os.Remove(temp.Name())
		return err
	}

	return os.Rename(temp.Name(), path)
}

func createVolumeFromCloudConfig(ctx context.Context, p string, spec CloudConfigSpec) error {
	if spec.NetworkConfig == "" {
		c := cmd.CommandContext(ctx, "cloud-localds", p, spec.UserData)
		return c.Run()
	}
	c := cmd.CommandContext(ctx, "cloud-localds", p, spec.UserData, "--network-config", spec.NetworkConfig)
	return c.Run()
}

// CreateVolume creates a volume with specified options
func (q QemuProvider) CreateVolume(ctx context.Context, node string, vol *VolumeSpec) error {
	p := q.volumePath(node, vol.Name)
	log.Info("Creating volume", map[string]interface{}{"node": node, "volume": vol.Name})
	if vol.Size != "" {
		return createEmptyVolume(ctx, p, vol.Size)
	} else if vol.Source != "" {
		return createVolumeFromURL(ctx, p, vol.Source)
	} else if vol.CloudConfig.UserData != "" {
		return createVolumeFromCloudConfig(ctx, p, vol.CloudConfig)
	}
	return errors.New("invalid volume type")
}

func createNVRAM(ctx context.Context, p string) error {
	_, err := os.Stat(p)
	if !os.IsNotExist(err) {
		return nil
	}
	return cmd.CommandContext(ctx, "cp", defaultOVMFVarsPath, p).Run()
}

// StartNode starts a QEMU vm
func (q QemuProvider) StartNode(ctx context.Context, n *Node) error {
	params := []string{"-enable-kvm"}

	for _, br := range n.Spec.Interfaces {
		tap := n.Name + "_" + br
		err := createTap(ctx, tap, br)
		if err != nil {
			return err
		}
		netdev := "tap,id=" + br + ",ifname=" + tap + ",script=no,downscript=no"
		if vhostNetSupported {
			netdev += ",vhost=on"
		}

		params = append(params, "-netdev", netdev)
		params = append(params, "-device",
			fmt.Sprintf("virtio-net-pci,netdev=%s,romfile=,mac=%s", br, generateRandomMACForKVM()))
	}
	for _, v := range n.Spec.Volumes {
		p := q.volumePath(n.Name, v.Name)
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
		err := createNVRAM(ctx, p)
		if err != nil {
			log.Error("Failed to create nvram", map[string]interface{}{
				"error": err,
			})
			return err
		}
		params = append(params, "-drive", "if=pflash,file="+defaultOVMFCodePath+",format=raw,readonly")
		params = append(params, "-drive", "if=pflash,file="+p+",format=raw")
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
