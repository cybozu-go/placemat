package placemat

import (
	"bufio"
	"bytes"
	"testing"
)

func testReadYaml(t *testing.T) {
	t.Parallel()
	yaml := `
kind: Network
name: net1
type: internal
---
kind: Image
name: ubuntu
file: hoge
---
kind: DataFolder
name: hostdata
dir: /home/cybozu/ubuntu
---
kind: Node
name: node1
---
kind: Node
name: node2
---
kind: Pod
name: pod1
apps:
  - name: bird
    image: docker://quay.io/cybozu/bird:2.0
---
kind: Certificate
name: cert1
key: xxx
cert: yyy
`

	cluster, err := ReadYaml(bufio.NewReader(bytes.NewReader([]byte(yaml))))
	if err != nil {
		t.Error(err)
	}
	if len(cluster.Networks) != 1 {
		t.Error("len(cluster.Networks) != 1, ", len(cluster.Networks))
	}
	if len(cluster.Images) != 1 {
		t.Error("len(cluster.Images) != 1, ", len(cluster.Images))
	}
	if len(cluster.DataFolders) != 1 {
		t.Error("len(cluster.DataFolders) != 1, ", len(cluster.DataFolders))
	}
	if len(cluster.Nodes) != 2 {
		t.Error("len(cluster.Nodes) != 2, ", len(cluster.Nodes))
	}
	if len(cluster.Pods) != 1 {
		t.Error("len(cluster.Pod) != 1,", len(cluster.Pods))
	}
	if len(cluster.Certificates) != 1 {
		t.Error("len(cluster.Certificates) != 1,", len(cluster.Certificates))
	}
}

func TestYAML(t *testing.T) {
	t.Run("ReadYaml", testReadYaml)
}
