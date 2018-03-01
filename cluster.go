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

func createNodeVolumes(ctx context.Context, provider Provider, cluster *Cluster) error {
	for _, n := range cluster.Nodes {
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

func startNodes(ctx context.Context, provider Provider, cluster *Cluster) error {
	env := cmd.NewEnvironment(ctx)
	for _, n := range cluster.Nodes {
		env.Go(func(ctx context.Context) error {
			return provider.StartNode(ctx, n)
		})
	}
	env.Stop()
	return env.Wait()
}

func Run(ctx context.Context, cluster *Cluster) error {
	var qemu QemuProvider

	err := createNodeVolumes(ctx, qemu, cluster)
	if err != nil {
		return err
	}

	return startNodes(ctx, qemu, cluster)
}
