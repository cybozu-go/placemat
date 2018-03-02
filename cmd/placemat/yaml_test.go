package main

import (
	"reflect"
	"testing"

	"github.com/cybozu-go/placemat"
)

func TestUnmarshalNode(t *testing.T) {
	cases := []struct {
		source string

		expected placemat.Node
	}{
		{
			source: `
kind: Node
name: node1
spec:
  interfaces:
    - br0
    - br1
  volumes:
    - name: boot
      size: 10GB
      recreatePolicy: Always
    - name: data
      size: 20GB
`,

			expected: placemat.Node{
				Name: "node1",
				Spec: placemat.NodeSpec{
					Interfaces: []string{"br0", "br1"},
					Volumes: []*placemat.VolumeSpec{
						{Name: "boot", Size: "10GB", RecreatePolicy: placemat.RecreateAlways},
						{Name: "data", Size: "20GB", RecreatePolicy: placemat.RecreateIfNotPresent},
					},
				},
			},
		},
		{
			source: `
 kind: Node
 name: node2
 `,

			expected: placemat.Node{
				Name: "node2",
				Spec: placemat.NodeSpec{
					Interfaces: []string{},
					Volumes:    []*placemat.VolumeSpec{},
				},
			},
		},
	}

	for _, c := range cases {
		actual, err := unmarshalNode([]byte(c.source))
		if err != nil {
			t.Error(err)
		}
		if !reflect.DeepEqual(*actual, c.expected) {
			t.Errorf("%v != %v", *actual, c.expected)
		}
	}

	errorSources := []string{
		`kind: Node`,
	}
	for _, c := range errorSources {
		_, err := unmarshalNode([]byte(c))
		if err == nil {
			t.Error("err == nil, ", err)
		}
	}

}

func TestUnmarshalCluster(t *testing.T) {
	yaml := `
kind: Node
name: node1
---
kind: Node
name: node2
`

	cluster, err := unmarshalCluster([]byte(yaml))
	if err != nil {
		t.Error(err)
	}
	if len(cluster.Nodes) != 2 {
		t.Error("len(cluster.Nodes) != 2, ", len(cluster.Nodes))
	}

}
