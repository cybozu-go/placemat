package placemat

// VolumeRecreatePolicy represents a policy to recreate a volume
type VolumeRecreatePolicy int

// Common recreate policies.  The default recreate policy is
// RecreateIfNotPresent which causes Placemat to skip creating an image if it
// already exists RecreateAlways causes Placemat to create always create
// an image even if the image is exists.  QEMU will be failed if no images
// exist and RecreateNever is specified.
const (
	RecreateIfNotPresent VolumeRecreatePolicy = iota
	RecreateAlways
	RecreateNever
)

// Network represents a network configuration
type Network struct{}

// VolumeSpec represents a volume specification
type VolumeSpec struct {
	Name           string
	Size           string
	RecreatePolicy VolumeRecreatePolicy
}

// NodeSpec represents a node specification
type NodeSpec struct {
	Interfaces []string
	Volumes    []*VolumeSpec
}

// Node represents a node configuration
type Node struct {
	Name string
	Spec NodeSpec
}

// NodeSetSpec represents a node-set specification
type NodeSetSpec struct {
	Replicas int
	Template *NodeSpec
}

// NodeSet represents a node-set configuration
type NodeSet struct {
	Name string
	Spec NodeSetSpec
}

// Cluster represents cluster configuration
type Cluster struct {
	Networks []*Network
	Nodes    []*Node
	NodeSets []*NodeSet
}

// Append appends the other cluster into the receiver
func (c *Cluster) Append(other *Cluster) *Cluster {
	c.Networks = append(c.Networks, other.Networks...)
	c.Nodes = append(c.Nodes, other.Nodes...)
	c.NodeSets = append(c.NodeSets, other.NodeSets...)
	return c
}
