package placemat

import (
	"context"
	"os"

	"os/exec"

	"github.com/cybozu-go/cmd"
	"github.com/cybozu-go/log"
)

// Cluster represents cluster configuration
type Cluster struct {
	Networks    []*Network
	Images      []*Image
	DataFolders []*DataFolder
	Nodes       []*Node
	Pods        []*Pod
}

// Append appends the other cluster into the receiver
func (c *Cluster) Append(other *Cluster) *Cluster {
	c.Networks = append(c.Networks, other.Networks...)
	c.Nodes = append(c.Nodes, other.Nodes...)
	c.Images = append(c.Images, other.Images...)
	c.DataFolders = append(c.DataFolders, other.DataFolders...)
	c.Pods = append(c.Pods, other.Pods...)
	return c
}

func (c *Cluster) Resolve() error {
	for _, n := range c.Nodes {
		err := n.Resolve(c)
		if err != nil {
			return err
		}
	}
	for _, p := range c.Pods {
		err := p.Resolve(c)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *Cluster) Start(ctx context.Context, r *Runtime) error {
	defer os.RemoveAll(r.tempDir)

	root, err := NewRootfs()
	if err != nil {
		return err
	}
	defer root.Destroy()

	err = createNatRules()
	if err != nil {
		return err
	}
	defer destroyNatRules()

	for _, n := range c.Networks {
		log.Info("Creating network", map[string]interface{}{"name": n.Name})
		err := n.Create(r.nameGenerator())
		if err != nil {
			return err
		}
		defer n.Destroy()
	}

	for _, df := range c.DataFolders {
		log.Info("initializing data folder", map[string]interface{}{
			"name": df.Name,
		})
		err := df.Prepare(ctx, r.tempDir, r.dataCache)
		if err != nil {
			return err
		}
	}

	for _, img := range c.Images {
		log.Info("initializing image resource", map[string]interface{}{
			"name": img.Name,
		})
		err := img.Prepare(ctx, r.imageCache)
		if err != nil {
			return err
		}
	}

	for _, p := range c.Pods {
		err := p.Prepare(ctx)
		if err != nil {
			return err
		}
	}

	env := cmd.NewEnvironment(ctx)

	nodeCh := make(chan bmcInfo, 10)

	vms := make(map[string]*nodeVM)
	for _, n := range c.Nodes {
		vm, err := n.Start(ctx, r, nodeCh)
		if err != nil {
			return err
		}
		vms[n.SMBIOS.Serial] = vm
	}

	bmcServer := newBMCServer(vms, c.Networks, nodeCh)
	env.Go(bmcServer.handleNode)

	var pods []*exec.Cmd
	for _, p := range c.Pods {
		c, err := p.Start(ctx, r, root.Path())
		if err != nil {
			return err
		}
		pods = append(pods, c)
		defer p.Destroy()
	}

	env.Stop()

	for _, vm := range vms {
		vm.cmd.Wait()
	}
	for _, pod := range pods {
		pod.Wait()
	}

	return env.Wait()
}
