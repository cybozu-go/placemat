package placemat

import (
	"context"
	"strconv"

	"github.com/cybozu-go/cmd"
	"github.com/cybozu-go/log"
)

// Provider represents a back-end of VM engine
type Provider interface {
	VolumeExists(ctx context.Context, node, vol string) (bool, error)

	CreateVolume(ctx context.Context, node string, vol *VolumeSpec) error

	CreateNetwork(ctx context.Context, n *Network) error

	DestroyNetwork(ctx context.Context, name string) error

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

func createNetworks(ctx context.Context, provider Provider, networks []*Network) error {
	for _, n := range networks {
		err := provider.CreateNetwork(ctx, n)
		if err != nil {
			return err
		}
	}
	return nil
}

func startNodes(env *cmd.Environment, provider Provider, nodes []*Node) {
	for _, n := range nodes {
		node := n
		env.Go(func(ctx context.Context) error {
			return provider.StartNode(ctx, node)
		})
	}
}

func handleDestroyNetwork(env *cmd.Environment, provider Provider, networks []*Network) {
	names := make([]string, len(networks))
	for i, n := range networks {
		names[i] = n.Name
	}
	env.Go(func(ctx context.Context) error {
		<-ctx.Done()

		for _, name := range names {
			err := provider.DestroyNetwork(context.Background(), name)
			if err != nil {
				log.Info("Failed to destroy networks", map[string]interface{}{
					"name":  name,
					"error": err,
				})
			}

		}
		return nil
	})
}

// Run runs VMs described in cluster by provider
func Run(ctx context.Context, provider Provider, cluster *Cluster) error {
	err := createNetworks(ctx, provider, cluster.Networks)
	if err != nil {
		return err
	}

	nodes := interpretNodesFromNodeSet(cluster)
	nodes = append(nodes, cluster.Nodes...)
	err = createNodeVolumes(ctx, provider, nodes)
	if err != nil {
		return err
	}

	env := cmd.NewEnvironment(ctx)
	startNodes(env, provider, nodes)
	handleDestroyNetwork(env, provider, cluster.Networks)
	env.Stop()
	return env.Wait()
}
