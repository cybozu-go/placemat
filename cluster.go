package placemat

import "strconv"

// Cluster represents cluster configuration
type Cluster struct {
	Networks    []*Network
	Images      []*Image
	DataFolders []*DataFolder
	Nodes       []*Node
	NodeSets    []*NodeSet
	Pods        []*Pod
}

// Append appends the other cluster into the receiver
func (c *Cluster) Append(other *Cluster) *Cluster {
	c.Networks = append(c.Networks, other.Networks...)
	c.Nodes = append(c.Nodes, other.Nodes...)
	c.NodeSets = append(c.NodeSets, other.NodeSets...)
	c.Images = append(c.Images, other.Images...)
	c.DataFolders = append(c.DataFolders, other.DataFolders...)
	c.Pods = append(c.Pods, other.Pods...)
	return c
}

// Resolve resolves references between resources
func (c *Cluster) Resolve(pv Provider) error {
	for _, node := range c.Nodes {
		for _, vs := range node.Spec.Volumes {
			err := vs.Resolve(c)
			if err != nil {
				return err
			}
		}
	}
	for _, nodeSet := range c.NodeSets {
		for _, vs := range nodeSet.Spec.Template.Volumes {
			err := vs.Resolve(c)
			if err != nil {
				return err
			}
		}
	}

	for _, p := range c.Pods {
		err := p.resolve(c)
		if err != nil {
			return err
		}
	}

	ic := pv.ImageCache()
	for _, img := range c.Images {
		img.cache = ic
	}

	dc := pv.DataCache()
	td := pv.TempDir()
	for _, folder := range c.DataFolders {
		folder.cache = dc
		folder.baseTempDir = td
	}
	return nil
}

// NodesFromNodeSets instantiates Node resources from NodeSets.
func (c *Cluster) NodesFromNodeSets() []*Node {
	var nodes []*Node
	for _, nodeSet := range c.NodeSets {
		for i := 1; i <= nodeSet.Spec.Replicas; i++ {
			var node Node
			node.Name = nodeSet.Name + "-" + strconv.Itoa(i)
			node.Spec = nodeSet.Spec.Template
			nodes = append(nodes, &node)
		}
	}
	return nodes
}
