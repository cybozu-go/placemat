package e2e

import (
	"os"
)

var (
	node1              = os.Getenv("NODE1")
	node2              = os.Getenv("NODE2")
	netns2             = os.Getenv("NETNS2")
	bmc1               = os.Getenv("BMC1")
	bmc2               = os.Getenv("BMC2")
	sshKeyFile         = os.Getenv("SSH_PRIVKEY")
	placematPath       = os.Getenv("PLACEMAT")
	pmctlPath          = os.Getenv("PMCTL")
	clusterYAML        = os.Getenv("CLUSTER_YAML")
	exampleClusterYAML = os.Getenv("EXAMPLE_CLUSTER_YAML")
)
