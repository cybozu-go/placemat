package placemat

import (
	"context"
)

// Provider represents a back-end of VM engine
type Provider interface {
	// These three are for Cluster.Resolve()
	ImageCache() *cache
	DataCache() *cache
	TempDir() string

	// These two are for Run()
	Start(context.Context, *Cluster) error
	Destroy(*Cluster) error
}

// Run runs VMs and Pods in cluster by provider.
func Run(ctx context.Context, provider Provider, cluster *Cluster) error {
	defer provider.Destroy(cluster)
	return provider.Start(ctx, cluster)
}
