package placemat

import (
	"context"
	"os"
	"path"

	"github.com/cybozu-go/cmd"
	"github.com/cybozu-go/log"
)

// QemuProvider is an implementation of Provider interface.  It launches
// qemu-system-x86_64 as a VM engine, and qemu-img to create image.
type QemuProvider struct {
	BaseDir string
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
	c := cmd.CommandContext(ctx, "ip", "link", "add", net.Name, "type", "bridge")
	log.Info("Creating network", map[string]interface{}{"name": net.Name})
	return c.Run()
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
	for _, v := range n.Spec.Volumes {
		p := q.volumePath(n.Name, v.Name)
		params = append(params, "-drive")
		params = append(params, "if=virtio,cache=none,aio=native,file="+p)
	}
	return cmd.CommandContext(ctx, "qemu-system-x86_64", params...).Run()
}
