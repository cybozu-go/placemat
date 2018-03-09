package placemat

import (
	"context"
	"strconv"
	"sync"
	"testing"
)

type MockProvider struct {
	volumes map[string]struct{}
	nodes   map[string]struct{}
	mutex   sync.Mutex
}

func (m *MockProvider) VolumeExists(ctx context.Context, node, vol string) (bool, error) {
	_, ok := m.volumes[node+"/"+vol]
	return ok, nil
}

func (m *MockProvider) CreateVolume(ctx context.Context, node string, vol *VolumeSpec) error {
	m.volumes[node+"/"+vol.Name] = struct{}{}
	return nil
}

func (m *MockProvider) StartNode(ctx context.Context, n *Node) error {
	m.mutex.Lock()
	m.nodes[n.Name] = struct{}{}
	m.mutex.Unlock()
	<-ctx.Done()
	return ctx.Err()
}

func TestInterpretNodesFromNodeSet(t *testing.T) {
	spec := &Cluster{}
	expectedReplicas := 2
	spec.NodeSets = getNodeSet(expectedReplicas)

	p := MockProvider{
		volumes: make(map[string]struct{}),
		nodes:   make(map[string]struct{}),
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	nodes := interpretNodesFromNodeSet(ctx, &p, spec)
	if len(nodes) != expectedReplicas {
		t.Fatal("expected len(p.volumes) != "+strconv.Itoa(expectedReplicas)+", ",
			len(p.volumes))
	}

}

func TestCreateNodeVolumes(t *testing.T) {
	spec := &Cluster{}

	spec.Nodes = []*Node{
		{Name: "host1", Spec: NodeSpec{Volumes: []*VolumeSpec{
			{Name: "vol1", Size: "10GB"}}}},
		{Name: "host2", Spec: NodeSpec{Volumes: []*VolumeSpec{
			{Name: "vol1", Size: "10GB"}, {Name: "vol2", Size: "20GB"}}}},
	}

	expectedReplicas := 2
	spec.NodeSets = getNodeSet(expectedReplicas)

	p := MockProvider{
		volumes: make(map[string]struct{}),
		nodes:   make(map[string]struct{}),
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	Run(ctx, &p, spec)

	if len(p.volumes) != 5 {
		t.Fatal("expected len(p.volumes) != 5, ", len(p.volumes))
	}
	if len(p.nodes) != 2 {
		t.Fatal("expected len(p.nodes) != 2, ", len(p.nodes))
	}
}

func getNodeSet(replicas int) []*NodeSet {
	template := NodeSpec{
		Volumes: []*VolumeSpec{
			{
				Name: "template-vol",
				Size: "10GB",
			},
		},
	}
	return []*NodeSet{
		{
			Name: "node",
			Spec: NodeSetSpec{
				Replicas: replicas,
				Template: &template,
			},
		},
	}
}
