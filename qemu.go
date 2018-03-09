package placemat

import (
	"context"
	"os"
	"path"
	"time"

	"github.com/cybozu-go/cmd"
	"github.com/cybozu-go/log"
)

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

// DestroyNetwork destroies a bridge by the name
func (q QemuProvider) DestroyNetwork(ctx context.Context, name string) error {
	c := cmd.CommandContext(ctx, "ip", "link", "delete", name, "type", "bridge")
	log.Info("Destroying network", map[string]interface{}{"name": name})
	return c.Run()
}

// CreateVolume creates the named by node and vol
func (q QemuProvider) CreateVolume(ctx context.Context, node string, vol *VolumeSpec) error {
	p := q.volumePath(node, vol.Name)
	c := cmd.CommandContext(ctx, "qemu-img", "create", "-f", "qcow2", p, vol.Size)
	log.Info("Creating volume", map[string]interface{}{"node": node, "volume": vol.Name})
	return c.Run()
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

		params = append(params, "-netdev", "tap,id="+br+",ifname="+tap+",script=no,downscript=no")
		params = append(params, "-device", "virtio-net-pci,netdev="+br+",romfile=")
	}
	for _, v := range n.Spec.Volumes {
		p := q.volumePath(n.Name, v.Name)
		params = append(params, "-drive", "if=virtio,cache=none,aio=native,file="+p)
	}
	err := cmd.CommandContext(ctx, "qemu-system-x86_64", params...).Run()
	if err != nil {
		log.Error("QEMU exited with an error", map[string]interface{}{
			"error": err,
		})
	}

	for _, br := range n.Spec.Interfaces {
		tap := n.Name + "_" + br
		func() {
			ctx2, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			err := deleteTap(ctx2, tap)
			if err != nil {
				log.Error("Failed to delete a TAP", map[string]interface{}{
					"name":  tap,
					"error": err,
				})
			}
		}()
	}
	return nil
}
