package placemat

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/cybozu-go/cmd"
	"github.com/cybozu-go/log"
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

func (q QemuProvider) volumePath(host, name string) string {
	return path.Join(q.BaseDir, host+"_"+name+".img")
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

func createVolumeFromURL(ctx context.Context, p string, url string) error {
	file, err := os.Create(p)
	if err != nil {
		return err
	}
	defer file.Close()

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil
	}
	req = req.WithContext(ctx)

	client := http.DefaultClient
	res, err := client.Do(req)
	if err != nil {
		return nil
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download: %s: %s", res.Status, url)
	}

	_, err = io.Copy(file, res.Body)
	if err != nil {
		return err
	}

	return nil
}

func createVolumeFromCloudConfig(ctx context.Context, p string, config string) error {
	c := cmd.CommandContext(ctx, "cloud-localds", p, config)
	return c.Run()
}

// CreateVolume creates the named by node and vol
func (q QemuProvider) CreateVolume(ctx context.Context, node string, vol *VolumeSpec) error {
	p := q.volumePath(node, vol.Name)
	log.Info("Creating volume", map[string]interface{}{"node": node, "volume": vol.Name})
	if vol.Size != "" {
		return createEmptyVolume(ctx, p, vol.Size)
	} else if vol.Source != "" {
		return createVolumeFromURL(ctx, p, vol.Source)
	} else if vol.CloudConfig.UserData != "" {
		return createVolumeFromCloudConfig(ctx, p, vol.CloudConfig.UserData)
	}
	return errors.New("invalid volume type")
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
		params = append(params, "-device", "virtio-net-pci,netdev="+br+",romfile=")
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
	return nil
}
