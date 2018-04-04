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

	DestroyNetwork(ctx context.Context, n *Network) error

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
		log.Info("Creating network", map[string]interface{}{"name": n.Name})
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

func destroyNetworks(provider Provider, networks []*Network) {
	for _, n := range networks {
		err := provider.DestroyNetwork(context.Background(), n)
		if err != nil {
			log.Error("Failed to destroy networks", map[string]interface{}{
				"name":  n.Name,
				"error": err,
			})
		} else {
			log.Info("Destroyed network", map[string]interface{}{"name": n.Name})
		}
	}
}

// Run runs VMs described in cluster by provider
func Run(ctx context.Context, provider Provider, cluster *Cluster) error {
	err := createNetworks(ctx, provider, cluster.Networks)
	if err != nil {
		return err
	}
	defer destroyNetworks(provider, cluster.Networks)

	nodes := interpretNodesFromNodeSet(cluster)
	nodes = append(nodes, cluster.Nodes...)
	err = createNodeVolumes(ctx, provider, nodes)
	if err != nil {
		return err
	}

	env := cmd.NewEnvironment(ctx)
	startNodes(env, provider, nodes)
	env.Stop()
	return env.Wait()
}
