package mtest

import (
	"os"
)

var (
	bridgeAddress      = os.Getenv("BRIDGE_ADDRESS")
	node1              = os.Getenv("NODE1")
	node2              = os.Getenv("NODE2")
	pod1               = os.Getenv("POD1")
	pod2               = os.Getenv("POD2")
	bmc1               = os.Getenv("BMC1")
	sshKeyFile         = os.Getenv("SSH_PRIVKEY")
	placematPath       = os.Getenv("PLACEMAT")
	pmctlPath          = os.Getenv("PMCTL")
	clusterYAML        = os.Getenv("CLUSTER_YAML")
	exampleClusterYAML = os.Getenv("EXAMPLE_CLUSTER_YAML")
	bmcCert            = os.Getenv("BMC_CERT")
	bmcKey             = os.Getenv("BMC_KEY")
)
