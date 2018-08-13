package mtest

import (
	"os"
)

var (
	bridgeAddress = os.Getenv("BRIDGE_ADDRESS")
	node1         = os.Getenv("NODE1")
	node2         = os.Getenv("NODE2")
	sshKeyFile    = os.Getenv("SSH_PRIVKEY")
	placemat      = os.Getenv("PLACEMAT")
	clusterYaml   = os.Getenv("CLUSTER_YAML")
	debug         = os.Getenv("DEBUG") == "1"
)
