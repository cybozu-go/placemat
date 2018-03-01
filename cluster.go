package placemat

import (
	"context"

	"github.com/cybozu-go/cmd"
)

type Provider interface {
	VolumeExists(ctx context.Context, node, vol string) (bool, error)

	CreateVolume(ctx context.Context, node string, vol *VolumeSpec) error

	StartNode(ctx context.Context, n *Node) error
}

func createNodeVolumes(ctx context.Context, provider Provider, cluster *Cluster) error {
	for _, n := range cluster.Nodes {
		for _, v := range n.Spec.Volumes {
			exists, err := provider.VolumeExists(ctx, n.Name, v.Name)
			if err != nil {
				return err
			}
			if v.RecreatePolicy == RecreateAlways ||
				v.RecreatePolicy == RecreateIfNotPresent && !exists {
				provider.CreateVolume(ctx, n.Name, v)
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

func Run(ctx context.Context, provider Provider, cluster *Cluster) error {
	err := createNodeVolumes(ctx, provider, cluster)
	if err != nil {
		return err
	}

	return startNodes(ctx, provider, cluster)
}
