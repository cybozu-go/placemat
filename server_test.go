package placemat

import (
	"context"
	"strconv"
	"sync"
	"testing"

	"github.com/cybozu-go/cmd"
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
	return nil
}

func TestInterpretNodesFromNodeSet(t *testing.T) {
	spec := &Cluster{}
	expectedReplicas := 2
	spec.NodeSets = getNodeSet(expectedReplicas)

	p := MockProvider{
		volumes: make(map[string]struct{}),
		nodes:   make(map[string]struct{}),
	}
	nodes := interpretNodesFromNodeSet(spec)
	if len(nodes) != expectedReplicas {
		t.Fatal("expected len(p.nodes) != "+strconv.Itoa(expectedReplicas)+", ",
			len(p.nodes))
	}

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

	expectedReplicas := 2
	spec.NodeSets = getNodeSet(expectedReplicas)

	p := newMockProvider()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	env := cmd.NewEnvironment(ctx)
	env.Go(func(ctx context.Context) error {
		return Run(ctx, p, spec)
	})
	env.Stop()
	err := env.Wait()
	if err != nil && err != context.Canceled {
		t.Fatal(err)
	}

	if len(p.networks) != 2 {
		t.Fatal("expected len(p.networks) != 2, ", len(p.networks))
	}
	if len(p.volumes) != 5 {
		t.Fatal("expected len(p.volumes) != 5, ", len(p.volumes))
	}
	if len(p.nodes) != 4 {
		t.Fatal("expected len(p.nodes) != 4, ", len(p.nodes))
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
				Template: template,
			},
		},
	}
}
