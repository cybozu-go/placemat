package placemat

import (
	"context"
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

func TestCreateNodeVolumes(t *testing.T) {
	spec := &Cluster{}
	spec.Nodes = []*Node{
		{Name: "host1", Spec: NodeSpec{Volumes: []*VolumeSpec{
			{Name: "vol1", Size: "10GB"}}}},
		{Name: "host2", Spec: NodeSpec{Volumes: []*VolumeSpec{
			{Name: "vol1", Size: "10GB"}, {Name: "vol2", Size: "20GB"}}}},
	}

	p := MockProvider{
		volumes: make(map[string]struct{}),
		nodes:   make(map[string]struct{}),
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	Run(ctx, &p, spec)

	if len(p.volumes) != 3 {
		t.Fatal("expected len(p.volumes) != 3, ", len(p.volumes))
	}
	if len(p.nodes) != 2 {
		t.Fatal("expected len(p.nodes) != 2, ", len(p.nodes))
	}
}
