package placemat

import (
	"context"
	"os"
	"path"

	"github.com/cybozu-go/cmd"
	"github.com/cybozu-go/log"
)

type QemuProvider struct {
	BaseDir string
}

func (q QemuProvider) volumePath(host, name string) string {
	return path.Join(q.BaseDir, host+"_"+name+".img")
}

func (p QemuProvider) VolumeExists(ctx context.Context, n *Node, v *VolumeSpec) (bool, error) {
	path := p.volumePath(n.Name, v.Name)
	_, err := os.Stat(path)
	return os.IsExist(err), nil
}

func (p QemuProvider) CreateVolume(ctx context.Context, n *Node, v *VolumeSpec) error {
	path := p.volumePath(n.Name, v.Name)
	cmd := cmd.CommandContext(ctx, "qemu-img", "create", "-f", "qcow2", path, v.Size)
	log.Info("Created volume", map[string]interface{}{"node": n.Name, "volume": v.Name})
	return cmd.Run()
}

func (p QemuProvider) StartNode(ctx context.Context, n *Node) error {
	params := []string{"-enable-kvm"}
	for _, v := range n.Spec.Volumes {
		path := p.volumePath(n.Name, v.Name)
		params = append(params, "-drive")
		params = append(params, "if=virtio,file="+path)
	}
	return cmd.CommandContext(ctx, "qemu-system-x86_64", params...).Run()
}
