package placemat

import (
	"context"
	"github.com/cybozu-go/cmd"
	"strconv"
)

// Provider represents a back-end of VM engine
type Provider interface {
	VolumeExists(ctx context.Context, node, vol string) (bool, error)

	CreateVolume(ctx context.Context, node string, vol *VolumeSpec) error

	StartNode(ctx context.Context, n *Node) error
}

func interpretNodesFromNodeSet(cluster *Cluster) []*Node {
	var nodes []*Node
	for _, nodeSet := range cluster.NodeSets {
		for i := 1; i <= nodeSet.Spec.Replicas; i++ {
			var node Node
			node.Name = nodeSet.Name + "-" + strconv.Itoa(i)
			node.Spec = nodeSet.Spec.Template
			nodes = append(nodes, &node)
		}
	}
	return nodes
}

func createNodeVolumes(ctx context.Context, provider Provider, nodes []*Node) error {
	for _, n := range nodes {
		for _, v := range n.Spec.Volumes {
			exists, err := provider.VolumeExists(ctx, n.Name, v.Name)
			if err != nil {
				return err
			}
			if !(v.RecreatePolicy == RecreateAlways ||
				v.RecreatePolicy == RecreateIfNotPresent && !exists) {
				continue
			}
			err = provider.CreateVolume(ctx, n.Name, v)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func startNodes(ctx context.Context, provider Provider, nodes []*Node) error {
	env := cmd.NewEnvironment(ctx)
	for _, n := range nodes {
		node := n
		env.Go(func(ctx context.Context) error {
			return provider.StartNode(ctx, node)
		})
	}
	env.Stop()
	return env.Wait()
}

// Run runs VMs described in cluster by provider
func Run(ctx context.Context, provider Provider, cluster *Cluster) error {
	nodes := interpretNodesFromNodeSet(cluster)
	nodes = append(nodes, cluster.Nodes...)

	err := createNodeVolumes(ctx, provider, nodes)
	if err != nil {
		return err
	}

	return startNodes(ctx, provider, nodes)
}
