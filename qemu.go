package placemat

import (
	"context"
	"fmt"
	"os"

	"github.com/cybozu-go/cmd"
)

type QemuProvider struct{}

func volumePath(host, name string) string {
	return host + "_" + name + ".img"
}

func (p QemuProvider) VolumeExists(ctx context.Context, n *Node, v *VolumeSpec) (bool, error) {
	path := volumePath(n.Name, v.Name)
	_, err := os.Stat(path)
	return os.IsExist(err), nil
}

func (p QemuProvider) CreateVolume(ctx context.Context, n *Node, v *VolumeSpec) error {
	path := volumePath(n.Name, v.Name)
	cmd := cmd.CommandContext(ctx, "qemu-img", "create", "-f", "qcow2", path, v.Size)
	return cmd.Run()
}

func (p QemuProvider) StartNode(ctx context.Context, n *Node) error {
	params := []string{"-enable-kvm"}
	for _, v := range n.Spec.Volumes {
		params = append(params, "-drive")
		params = append(params, "if=virtio,file="+v.Name)
	}

	fmt.Println("starts", params)
	return cmd.CommandContext(ctx, "qemu-system-x86_64", params...).Run()
}
