package placemat

import (
	"context"
	"sync"
	"testing"
)

type MockProvider struct {
	networks map[string]struct{}
	volumes  map[string]struct{}
	nodes    map[string]struct{}
	mutex    sync.Mutex
}

func newMockProvider() *MockProvider {
	return &MockProvider{
		networks: make(map[string]struct{}),
		volumes:  make(map[string]struct{}),
		nodes:    make(map[string]struct{}),
	}
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

func (m *MockProvider) CreateNetwork(ctx context.Context, n *Network) error {
	m.networks[n.Name] = struct{}{}
	return nil
}

func (m *MockProvider) DestroyNetwork(ctx context.Context, name string) error {
	m.networks[name] = struct{}{}
	return nil
}

func TestRun(t *testing.T) {
	spec := &Cluster{}
	spec.Nodes = []*Node{
		{Name: "host1", Spec: NodeSpec{Volumes: []*VolumeSpec{
			{Name: "vol1", Size: "10GB"}}}},
		{Name: "host2", Spec: NodeSpec{Volumes: []*VolumeSpec{
			{Name: "vol1", Size: "10GB"}, {Name: "vol2", Size: "20GB"}}}},
	}
	spec.Networks = []*Network{
		&Network{Name: "net1"},
		&Network{Name: "net2"},
	}

	p := newMockProvider()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	Run(ctx, p, spec)

	if len(p.volumes) != 3 {
		t.Fatal("expected len(p.volumes) != 3, ", len(p.volumes))
	}
	if len(p.nodes) != 2 {
		t.Fatal("expected len(p.nodes) != 2, ", len(p.nodes))
	}
	if len(p.networks) != 2 {
		t.Fatal("expected len(p.networks) != 2, ", len(p.networks))
	}
}
