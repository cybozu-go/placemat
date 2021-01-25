package placemat

import (
	"context"
	"fmt"
	"net"
	"os"
	"sync"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/placemat/v2/pkg/dcnet"
	"github.com/cybozu-go/placemat/v2/pkg/types"
	"github.com/cybozu-go/placemat/v2/pkg/vm"
	"github.com/cybozu-go/well"
)

// Cluster represents the interface to setup virtual data center
type Cluster interface {
	// Setup configures and starts virtual data center
	Setup(ctx context.Context, r *vm.Runtime) error
}

type cluster struct {
	networkSpecs []*types.NetworkSpec
	netNSSpecs   []*types.NetNSSpec
	nodeSpecs    []*types.NodeSpec
	imageSpecs   []*types.ImageSpec
	networks     []dcnet.Network
	netNss       []dcnet.NetNS
	nodes        []vm.Node
	vms          map[string]vm.VM
	networkMap   map[string]dcnet.Network
	nodeSpecMap  map[string]*types.NodeSpec
	nodeMap      map[string]vm.Node
}

// NewCluster creates a Cluster from spec.
func NewCluster(spec *types.ClusterSpec) (*cluster, error) {
	cluster := &cluster{
		networkSpecs: spec.Networks,
		netNSSpecs:   spec.NetNSs,
		nodeSpecs:    spec.Nodes,
		imageSpecs:   spec.Images,
		vms:          make(map[string]vm.VM),
		networkMap:   make(map[string]dcnet.Network),
		nodeSpecMap:  make(map[string]*types.NodeSpec),
		nodeMap:      make(map[string]vm.Node),
	}

	for _, node := range cluster.nodeSpecs {
		cluster.nodeSpecMap[node.Name] = node
	}

	return cluster, nil
}

func (c *cluster) Setup(ctx context.Context, r *vm.Runtime) error {
	defer c.cleanup(r)

	if r.Force {
		dcnet.CleanupNatRules()
		dcnet.CleanupAllLinks()
	}

	err := dcnet.CreateNatRules()
	if err != nil {
		return err
	}

	mtu, err := dcnet.DetectMTU()
	if err != nil {
		return fmt.Errorf("failed to detect MTU: %w", err)
	}

	for _, spec := range c.networkSpecs {
		network, err := dcnet.NewNetwork(spec)
		if err != nil {
			return err
		}
		c.networks = append(c.networks, network)
		c.networkMap[spec.Name] = network

		if err := network.Setup(mtu, r.Force); err != nil {
			return fmt.Errorf("failed to create Network: %w", err)
		}
	}

	for _, spec := range c.nodeSpecs {
		node, err := vm.NewNode(spec, c.imageSpecs)
		if err != nil {
			return err
		}
		c.nodes = append(c.nodes, node)
		c.nodeMap[spec.Name] = node

		if err := node.Prepare(ctx, r.ImageCache); err != nil {
			return fmt.Errorf("failed to prepare Node: %w", err)
		}
	}

	nodeCh := make(chan vm.BMCInfo, len(c.nodeSpecs))

	var mu sync.Mutex

	env := well.NewEnvironment(ctx)
	for _, n := range c.nodes {
		n := n
		env.Go(func(ctx2 context.Context) error {
			// reference the original context because ctx2 will soon be cancelled.
			vm, serial, err := n.Setup(ctx, r, mtu, nodeCh)
			if err != nil {
				return err
			}
			mu.Lock()
			c.vms[serial] = vm
			mu.Unlock()
			return nil
		})
	}
	env.Stop()
	err = env.Wait()
	defer func() {
		for _, vm := range c.vms {
			vm.Cleanup()
		}
	}()
	if err != nil {
		return err
	}

	bmcServer := vm.NewBMCServer(c.vms, c.networks, nodeCh, r.TempDir)
	env = well.NewEnvironment(ctx)
	env.Go(bmcServer.Start)

	for _, spec := range c.netNSSpecs {
		netNs, err := dcnet.NewNetNS(spec)
		if err != nil {
			return err
		}
		c.netNss = append(c.netNss, netNs)

		n := netNs
		env.Go(func(ctx context.Context) error {
			return n.Setup(ctx, mtu, r.Force)
		})
	}

	for _, vm := range c.vms {
		vm := vm
		env.Go(func(ctx context.Context) error {
			return vm.Wait()
		})
	}

	addr, err := net.ResolveTCPAddr("tcp", r.ListenAddr)
	if err != nil {
		log.Error("failed to resolve TCP address", map[string]interface{}{
			log.FnError: err,
			"address":   r.ListenAddr,
		})
	}
	listener, err := net.ListenTCP("tcp", addr)
	if err != nil {
		log.Error("failed to listen TCP address", map[string]interface{}{
			log.FnError: err,
			"address":   r.ListenAddr,
		})
	}
	apiServer := newAPIServer(c, r)
	if err := apiServer.start(ctx, listener); err != nil {
		return err
	}

	env.Stop()
	if err := env.Wait(); err != nil {
		return err
	}

	return nil
}

func (c *cluster) cleanup(r *vm.Runtime) {
	if err := os.RemoveAll(r.TempDir); err != nil {
		log.Warn("failed to remove temp files", map[string]interface{}{
			log.FnError: err,
		})
	}

	dcnet.CleanupNatRules()

	for _, n := range c.networks {
		n.Cleanup()
	}

	for _, n := range c.nodes {
		n.Cleanup()
	}

	for _, n := range c.netNss {
		n.Cleanup()
	}

	for _, vm := range c.vms {
		vm.Cleanup()
	}
}
