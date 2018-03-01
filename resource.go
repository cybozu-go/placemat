package placemat

type VolumeRecreatePolicy int

const (
	RecreateIfNotPresent VolumeRecreatePolicy = iota
	RecreateAlways
	RecreateNever
)

type Network struct{}

type VolumeSpec struct {
	Name           string
	Size           string
	RecreatePolicy VolumeRecreatePolicy
}

type NodeSpec struct {
	Interfaces []string
	Volumes    []*VolumeSpec
}

type Node struct {
	Name string
	Spec NodeSpec
}

type NodeSet struct{}
