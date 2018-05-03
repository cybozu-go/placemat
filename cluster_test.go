package placemat

import (
	"strconv"
	"testing"
)

func getNodeSet(replicas int) []*NodeSet {
	template := NodeSpec{
		Volumes: []Volume{NewRawVolume("template-vol", RecreateAlways, "10GB")},
	}
	return []*NodeSet{
		{
			Name: "node",
			Spec: NodeSetSpec{
				Replicas: replicas,
				Template: template,
			},
		},
	}
}

func testNodesFromNodeSets(t *testing.T) {
	c := &Cluster{}
	expectedReplicas := 2
	c.NodeSets = getNodeSet(expectedReplicas)

	nodes := c.NodesFromNodeSets()
	if len(nodes) != expectedReplicas {
		t.Fatal("expected len(nodes) != "+strconv.Itoa(expectedReplicas)+", ",
			len(nodes))
	}
}

func TestCluster(t *testing.T) {
	t.Run("NodesFromNodeSet", testNodesFromNodeSets)
}
