package placemat

import (
	"context"
	"os"
	"path"

	"github.com/cybozu-go/cmd"
	"github.com/cybozu-go/log"
)

// QemuProvider is an implementation of Provider interface
type QemuProvider struct {
	BaseDir string
}

func (q QemuProvider) volumePath(host, name string) string {
	return path.Join(q.BaseDir, host+"_"+name+".img")
}

// VolumeExists checks if the volume exists
func (q QemuProvider) VolumeExists(ctx context.Context, node, vol string) (bool, error) {
	path := q.volumePath(node, vol)
	_, err := os.Stat(path)
	return !os.IsNotExist(err), nil
}

// CreateVolume creates the named by node and vol
func (q QemuProvider) CreateVolume(ctx context.Context, node string, vol *VolumeSpec) error {
	path := q.volumePath(node, vol.Name)
	cmd := cmd.CommandContext(ctx, "qemu-img", "create", "-f", "qcow2", path, vol.Size)
	log.Info("Created volume", map[string]interface{}{"node": node, "volume": vol.Name})
	return cmd.Run()
}

// StartNode starts a QEMU vm
func (q QemuProvider) StartNode(ctx context.Context, n *Node) error {
	params := []string{"-enable-kvm"}
	for _, v := range n.Spec.Volumes {
		path := q.volumePath(n.Name, v.Name)
		params = append(params, "-drive")
		params = append(params, "if=virtio,file="+path)
	}
	return cmd.CommandContext(ctx, "qemu-system-x86_64", params...).Run()
}
