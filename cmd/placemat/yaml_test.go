package main

import (
	"bufio"
	"bytes"
	"net/url"
	"reflect"
	"testing"

	"github.com/cybozu-go/placemat"
)

func testUnmarshalImage(t *testing.T) {
	t.Parallel()

	url, _ := url.Parse("https://cloud-images.ubuntu.com/releases/16.04/release/ubuntu-16.04-server-cloudimg-amd64-disk1.img")

	cases := []struct {
		source   string
		expected placemat.Image
	}{
		{
			source: `
kind: Image
name: ubuntu-image
spec:
  url: https://cloud-images.ubuntu.com/releases/16.04/release/ubuntu-16.04-server-cloudimg-amd64-disk1.img
`,
			expected: placemat.Image{
				Name: "ubuntu-image",
				Spec: placemat.ImageSpec{
					URL: url,
				},
			},
		},
		{
			source: `
kind: Image
name: ubuntu-image
spec:
  file: /home/cybozu/ubuntu-18.04.img
`,
			expected: placemat.Image{
				Name: "ubuntu-image",
				Spec: placemat.ImageSpec{
					File: "/home/cybozu/ubuntu-18.04.img",
				},
			},
		},
	}

	for _, c := range cases {
		actual, err := unmarshalImage([]byte(c.source))
		if err != nil {
			t.Error(err)
		}
		if !reflect.DeepEqual(*actual, c.expected) {
			t.Errorf("%v != %v", *actual, c.expected)
		}
	}

	errorCases := []string{
		`
kind: Image
spec:
  file: "/home/cybozu/ubuntu.img"
`,
		`
kind: Image
name: "empty-spec"
spec:
`,
		`
kind: Image
name: "both-spec"
spec:
  file: "/home/cybozu/ubuntu.img"
  url: https://cloud-images.ubuntu.com/releases/16.04/release/ubuntu-16.04-server-cloudimg-amd64-disk1.img
`,
		`
kind: Image
name: "invalid-url"
spec:
  url: $://cloud-images.ubuntu.com/releases/16.04/release/ubuntu-16.04-server-cloudimg-amd64-disk1.img
`,
	}

	for _, c := range errorCases {
		image, err := unmarshalImage([]byte(c))
		if err == nil {
			t.Errorf("%s should be error", image.Name)
		}
	}
}

func testUnmarshalDataFolder(t *testing.T) {
	t.Parallel()

	url, _ := url.Parse("https://quay.io/cybozu/bird/bird.img")

	cases := []struct {
		source   string
		expected placemat.DataFolder
	}{
		{
			source: `
kind: DataFolder
name: containers
spec:
  dir: /home/cybozu/containers
`,
			expected: placemat.DataFolder{
				Name: "containers",
				Spec: placemat.DataFolderSpec{
					Dir: "/home/cybozu/containers",
				},
			},
		},
		{
			source: `
kind: DataFolder
name: containers
spec:
  files:
    - name: bird.img
      url: https://quay.io/cybozu/bird/bird.img
    - name: ubuntu.img
      file: /home/cybozu/containers/ubuntu18.04.img
`,
			expected: placemat.DataFolder{
				Name: "containers",
				Spec: placemat.DataFolderSpec{
					Files: []placemat.DataFolderFile{
						placemat.DataFolderFile{
							Name: "bird.img",
							URL:  url,
						},
						placemat.DataFolderFile{
							Name: "ubuntu.img",
							File: "/home/cybozu/containers/ubuntu18.04.img",
						},
					},
				},
			},
		},
	}

	for _, c := range cases {
		actual, err := unmarshalDataFolder([]byte(c.source))
		if err != nil {
			t.Error(err)
		}
		if !reflect.DeepEqual(*actual, c.expected) {
			t.Errorf("%v != %v", *actual, c.expected)
		}
	}

	errorCases := []string{
		`
kind: DataFolder
spec:
  dir: "/home/cybozu/ubuntu"
`,
		`
kind: DataFolder
name: "empty-spec"
spec:
`,
		`
kind: DataFolder
name: "both-spec"
spec:
  dir: "/home/cybozu/ubuntu"
  files:
    - name: "ubuntu.img"
      file: "/home/cybozu/ubuntu/ubuntu.img"
`,
		`
kind: DataFolder
name: "both-location"
spec:
  files:
    - name: "ubuntu.img"
      file: "/home/cybozu/ubuntu/ubuntu.img"
      url: "https://quay.io/cybozu/ubuntu/ubuntu.img"
`,
	}

	for _, c := range errorCases {
		dataFolder, err := unmarshalDataFolder([]byte(c))
		if err == nil {
			t.Errorf("%s should be error", dataFolder.Name)
		}
	}
}

func testUnmarshalNetwork(t *testing.T) {
	t.Parallel()
	cases := []struct {
		source string

		expected placemat.Network
	}{
		{
			source: `
kind: Network
name: net1
spec:
  internal: false
  use-nat: true
  addresses:
    - 10.0.0.1
    - 10.0.0.2
`,
			expected: placemat.Network{
				Name: "net1",
				Spec: placemat.NetworkSpec{
					Internal:  false,
					UseNAT:    true,
					Addresses: []string{"10.0.0.1", "10.0.0.2"},
				},
			},
		},
		{
			source: `

kind: Network
name: net2
spec:
  internal: true
`,
			expected: placemat.Network{
				Name: "net2",
				Spec: placemat.NetworkSpec{
					Internal: true,
				},
			},
		},
	}

	for _, c := range cases {
		actual, err := unmarshalNetwork([]byte(c.source))
		if err != nil {
			t.Error(err)
		}
		if !reflect.DeepEqual(*actual, c.expected) {
			t.Errorf("%v != %v", *actual, c.expected)
		}
	}

	errorCases := []struct {
		source string

		expected string
	}{
		{
			source: `
kind: Network
name: net1
spec:
  internal: true
  use-nat: true
  addresses:
    - 10.0.0.1
    - 10.0.0.2
`,
			expected: "'use-nat' and 'addresses' are meaningless for internal network",
		},
		{
			source: `
kind: Network
name: net2
spec:
  internal: false
  use-nat: true
  addresses:
`,
			expected: "addresses is empty for non-internal network",
		},
	}

	for _, c := range errorCases {
		_, err := unmarshalNetwork([]byte(c.source))
		if err.Error() != c.expected {
			t.Errorf("%v != %v", err.Error(), c.expected)
		}
	}

}

func testUnmarshalNode(t *testing.T) {
	t.Parallel()
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
    - kind: image
      name: ubuntu
      recreatePolicy: IfNotPresent
      spec:
        image: ubuntu-image
    - kind: localds
      name: seed
      recreatePolicy: Always
      spec:
        network-config: network.yml
        user-data: user-data.yml
    - kind: raw
      name: data
      spec:
        size: 20GB
    - kind: vvfat
      name: hostdata
      spec:
        folder: containers
  resources:
    cpu: 4
    memory: 8G
  bios: legacy
  smbios:
    manufacturer: QEMU
    product: Mk2
    serial: 1234abcd
`,

			expected: placemat.Node{
				Name: "node1",
				Spec: placemat.NodeSpec{
					Interfaces: []string{"br0", "br1"},
					Volumes: []placemat.Volume{
						placemat.NewImageVolume("ubuntu", placemat.RecreateIfNotPresent, "ubuntu-image", false),
						placemat.NewLocalDSVolume("seed", placemat.RecreateAlways, "user-data.yml", "network.yml"),
						placemat.NewRawVolume("data", placemat.RecreateIfNotPresent, "20GB"),
						placemat.NewVVFATVolume("hostdata", placemat.RecreateIfNotPresent, "containers"),
					},
					Resources: placemat.ResourceSpec{CPU: "4", Memory: "8G"},
					BIOS:      placemat.LegacyBIOS,
					SMBIOS:    placemat.SMBIOSSpec{Manufacturer: "QEMU", Product: "Mk2", Serial: "1234abcd"},
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
					Volumes:    []placemat.Volume{},
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

	errorSources := []struct {
		source string

		expected string
	}{
		{
			source:   `kind: Node`,
			expected: "node name is empty",
		},
		{
			source: `
kind: Node
name: node1
spec:
  bios: None
`,
			expected: "invalid BIOS: None",
		},
		{
			source: `
kind: Node
name: node1
spec:
  volumes:
    - name: vol
      recreatePolicy: Sometime
`,
			expected: "invalid RecreatePolicy: Sometime",
		},
	}
	for _, c := range errorSources {
		_, err := unmarshalNode([]byte(c.source))
		if err.Error() != c.expected {
			t.Errorf("%v != %v", err.Error(), c.expected)
		}
	}

}

func testUnmarshalNodeSet(t *testing.T) {
	t.Parallel()
	cases := []struct {
		source   string
		expected placemat.NodeSet
	}{
		{
			source: `
kind: NodeSet
name: worker
spec:
  replicas: 3
  template:
    interfaces:
      - my-net
    volumes:
      - kind: raw
        name: data
        spec:
          size: 10GB
`,
			expected: placemat.NodeSet{
				Name: "worker",
				Spec: placemat.NodeSetSpec{
					Replicas: 3,
					Template: placemat.NodeSpec{
						Interfaces: []string{"my-net"},
						Volumes: []placemat.Volume{
							placemat.NewRawVolume("data", placemat.RecreateIfNotPresent, "10GB"),
						},
					},
				},
			},
		},
	}
	for _, c := range cases {
		actual, err := unmarshalNodeSet([]byte(c.source))
		if err != nil {
			t.Error(err)
		}
		if !reflect.DeepEqual(*actual, c.expected) {
			t.Errorf("%v != %v", *actual, c.expected)
		}
	}
	errorSources := []string{
		`kind: NodeSet`,
	}
	for _, c := range errorSources {
		_, err := unmarshalNodeSet([]byte(c))
		if err == nil {
			t.Error("err == nil, ", err)
		}
	}
}

func testUnmarshalPod(t *testing.T) {
	t.Parallel()

	bird := &placemat.PodApp{
		Name:           "bird",
		Image:          "docker://quay.io/cybozu/bird:2.0",
		ReadOnlyRootfs: true,
		User:           "10000",
		Group:          "10000",
		Exec:           "/sbin/bird",
		Args:           []string{"-d"},
		Env: map[string]string{
			"FOO": "bar",
		},
		CapsRetain: []string{"CAP_NET_ADMIN", "CAP_NET_BIND_SERVICE", "CAP_NET_RAW"},
	}
	bird.AddMountPoint("config", "/etc/bird")
	debug := &placemat.PodApp{
		Name:  "debug",
		Image: "docker://quay.io/cybozu/ubuntu-debug:18.04",
	}

	configVol, err := placemat.NewPodVolume("config", "host", "host-dir", "", "", "", true)
	if err != nil {
		t.Fatal(err)
	}
	runVol, err := placemat.NewPodVolume("run", "empty", "", "0700", "10000", "10000", false)
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		source   string
		expected *placemat.Pod
	}{
		{
			`
kind: Pod
name: pod1
`,
			nil,
		},
		{
			`
kind: Pod
name: pod1
spec:
  apps:
    - name: bird
      image: docker://quay.io/cybozu/bird:2.0
      readonly-rootfs: true
      user: 10000
      group: 10000
      exec: /sbin/bird
      args: ["-d"]
      env:
        FOO: bar
      mount:
        - volume: config
          target: /etc/bird
      caps-retain:
        - CAP_NET_ADMIN
        - CAP_NET_BIND_SERVICE
        - CAP_NET_RAW
    - name: debug
      image: docker://quay.io/cybozu/ubuntu-debug:18.04
`,
			&placemat.Pod{
				Name: "pod1",
				Apps: []*placemat.PodApp{bird, debug},
			},
		},
		{
			`
kind: Pod
name: pod1
spec:
  init-scripts:
    - /etc/profile
  apps:
    - name: debug
      image: docker://quay.io/cybozu/ubuntu-debug:18.04
`,
			&placemat.Pod{
				Name:        "pod1",
				InitScripts: []string{"/etc/profile"},
				Apps:        []*placemat.PodApp{debug},
			},
		},
		{
			`
kind: Pod
name: pod1
spec:
  interfaces:
    - network: net0
      addresses:
        - 10.0.0.1/24
  apps:
    - name: debug
      image: docker://quay.io/cybozu/ubuntu-debug:18.04
`,
			&placemat.Pod{
				Name: "pod1",
				Interfaces: []struct {
					NetworkName string
					Addresses   []string
				}{
					{
						"net0",
						[]string{"10.0.0.1/24"},
					},
				},
				Apps: []*placemat.PodApp{debug},
			},
		},
		{
			`
kind: Pod
name: pod1
spec:
  volumes:
    - name: config
      kind: host
      folder: host-dir
      readonly: true
    - name: run
      kind: empty
      mode: "0700"
      uid: 10000
      gid: 10000
  apps:
    - name: debug
      image: docker://quay.io/cybozu/ubuntu-debug:18.04
`,
			&placemat.Pod{
				Name:    "pod1",
				Volumes: []placemat.PodVolume{configVol, runVol},
				Apps:    []*placemat.PodApp{debug},
			},
		},
		{
			`
kind: Pod
name: pod1
spec:
  volumes:
    - name: config
      kind: bad
      folder: host-dir
      readonly: true
  apps:
    - name: debug
      image: docker://quay.io/cybozu/ubuntu-debug:18.04
`,
			nil,
		},
	}

	for _, c := range cases {
		actual, err := unmarshalPod([]byte(c.source))
		if c.expected == nil {
			if err == nil {
				t.Error("unmarshal should fail for", c.source)
			}
			continue
		}

		if err != nil {
			t.Error("unmarshal should not fail for", c.source, err)
			continue
		}

		if !reflect.DeepEqual(actual, c.expected) {
			t.Errorf("%#v != %#v", actual, c.expected)
		}
	}
}

func testUnmarshalCluster(t *testing.T) {
	t.Parallel()
	yaml := `
kind: Network
name: net1
spec:
  internal: true
---
kind: Image
name: ubuntu
spec:
  file: hoge
---
kind: DataFolder
name: hostdata
spec:
  dir: /home/cybozu/ubuntu
---
kind: Node
name: node1
---
kind: Node
name: node2
---
kind: NodeSet
name: nodeSet
---
kind: Pod
name: pod1
spec:
  apps:
    - name: bird
      image: docker://quay.io/cybozu/bird:2.0
`

	cluster, err := readYaml(bufio.NewReader(bytes.NewReader([]byte(yaml))))
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
	if len(cluster.NodeSets) != 1 {
		t.Error("len(cluster.NodeSets) != 1, ", len(cluster.NodeSets))
	}
	if len(cluster.Pods) != 1 {
		t.Error("len(cluster.Pod) != 1,", len(cluster.Pods))
	}
}

func TestYAML(t *testing.T) {
	t.Run("image", testUnmarshalImage)
	t.Run("dataFolder", testUnmarshalDataFolder)
	t.Run("network", testUnmarshalNetwork)
	t.Run("node", testUnmarshalNode)
	t.Run("nodeSet", testUnmarshalNodeSet)
	t.Run("pod", testUnmarshalPod)
	t.Run("cluster", testUnmarshalCluster)
}
