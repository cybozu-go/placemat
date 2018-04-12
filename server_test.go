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
	volumes  int
	nodes    map[string]struct{}
	mutex    sync.Mutex
}

func newMockProvider() *MockProvider {
	return &MockProvider{
		networks: make(map[string]struct{}),
		nodes:    make(map[string]struct{}),
	}
}

func (m *MockProvider) ImageCache() *cache {
	return nil
}

func (m *MockProvider) DataCache() *cache {
	return nil
}

func (m *MockProvider) TempDir() string {
	return ""
}

func (m *MockProvider) PrepareNode(ctx context.Context, n *Node) error {
	m.mutex.Lock()
	m.volumes += len(n.Spec.Volumes)
	m.mutex.Unlock()
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
		networks: make(map[string]struct{}),
		nodes:    make(map[string]struct{}),
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

func (m *MockProvider) DestroyNetwork(ctx context.Context, n *Network) error {
	m.networks[n.Name] = struct{}{}
	return nil
}

func TestRun(t *testing.T) {
	vol1 := NewRawVolume("vol1", RecreateAlways, "10GB")
	vol2 := NewRawVolume("vol2", RecreateAlways, "20GB")
	spec := &Cluster{}
	spec.Nodes = []*Node{
		{Name: "host1", Spec: NodeSpec{Volumes: []Volume{vol1}}},
		{Name: "host2", Spec: NodeSpec{Volumes: []Volume{vol1, vol2}}},
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
	if p.volumes != 5 {
		t.Fatal("expected p.volumes != 5, ", p.volumes)
	}
	if len(p.nodes) != 4 {
		t.Fatal("expected len(p.nodes) != 4, ", len(p.nodes))
	}
}

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
