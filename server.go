package placemat

import (
	"context"
	"os"
	"strconv"

	"github.com/cybozu-go/cmd"
	"github.com/cybozu-go/log"
)

// Provider represents a back-end of VM engine
type Provider interface {
	ImageCache() *cache

	DataCache() *cache

	TempDir() string

	CreateNetwork(context.Context, *Network) error

	DestroyNetwork(context.Context, *Network) error

	PrepareNode(context.Context, *Node) error

	StartNode(context.Context, *Node) error
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
	defer func() {
		err := os.RemoveAll(provider.TempDir())
		if err != nil {
			log.Error("Failed to remove temporary directory", map[string]interface{}{
				"dir":       provider.TempDir(),
				log.FnError: err,
			})
		}
	}()

	err := createNetworks(ctx, provider, cluster.Networks)
	if err != nil {
		return err
	}
	defer destroyNetworks(provider, cluster.Networks)

	nodes := interpretNodesFromNodeSet(cluster)
	nodes = append(nodes, cluster.Nodes...)

	for _, n := range nodes {
		err = provider.PrepareNode(ctx, n)
		if err != nil {
			return err
		}
	}

	env := cmd.NewEnvironment(ctx)
	startNodes(env, provider, nodes)
	env.Stop()
	return env.Wait()
}
