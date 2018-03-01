package placemat

import (
	"context"

	"github.com/cybozu-go/cmd"
)

type Provider interface {
	VolumeExists(ctx context.Context, n *Node, v *VolumeSpec) (bool, error)

	CreateVolume(ctx context.Context, n *Node, v *VolumeSpec) error

	StartNode(ctx context.Context, n *Node) error
}

func createNodeVolumes(ctx context.Context, provider Provider, nodes []*Node) error {
	for _, n := range nodes {
		for _, v := range n.Spec.Volumes {
			exists, err := provider.VolumeExists(ctx, n, v)
			if err != nil {
				return err
			}
			if !(v.RecreatePolicy == RecreateAlways ||
				v.RecreatePolicy == RecreateIfNotPresent && exists) {
				continue
			}
		}
	}
	return nil
}

func startNodes(ctx context.Context, provider Provider, nodes []*Node) error {
	env := cmd.NewEnvironment(ctx)
	for _, n := range nodes {
		env.Go(func(ctx context.Context) error {
			return provider.StartNode(ctx, n)
		})
	}
	env.Stop()
	return env.Wait()
}

func Run(ctx context.Context, networks []*Network, nodes []*Node, nodeSets []*NodeSet) error {
	var qemu QemuProvider

	err := createNodeVolumes(ctx, qemu, nodes)
	if err != nil {
		return err
	}

	return startNodes(ctx, qemu, nodes)
}
