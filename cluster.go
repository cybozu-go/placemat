package placemat

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

// Resolve resolves references between resources
func (c *Cluster) Resolve(pv Provider) error {
	for _, node := range c.Nodes {
		for _, vs := range node.volumes {
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
	return nil
}
