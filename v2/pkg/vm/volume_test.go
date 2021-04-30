package vm

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cybozu-go/placemat/v2/pkg/types"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("NodeVolume", func() {
	It("should create an image volume as specified", func() {
		// Set up runtime
		cur, err := os.Getwd()
		Expect(err).NotTo(HaveOccurred())
		temp := filepath.Join(cur, "temp")
		Expect(os.Mkdir(temp, 0755)).NotTo(HaveOccurred())

		// Create dummy files and directories
		_, err = os.Create("temp/cybozu-ubuntu-18.04-server-cloudimg-amd64.img")
		Expect(err).NotTo(HaveOccurred())
		defer os.RemoveAll(temp)

		clusterYaml := `
kind: Node
name: boot-0
cpu: 8
memory: 2G
volumes:
- kind: image
  name: root
  image: custom-ubuntu-image
  cache: writeback
  copy-on-write: true
smbios:
  serial: fb8f2417d0b4db30050719c31ce02a2e8141bbd8
---
kind: Image
name: custom-ubuntu-image
file: temp/cybozu-ubuntu-18.04-server-cloudimg-amd64.img
`
		cluster, err := types.Parse(strings.NewReader(clusterYaml))
		Expect(err).NotTo(HaveOccurred())

		nodeSpec := cluster.Nodes[0]
		volumeSpec := nodeSpec.Volumes[0]

		volume, err := newNodeVolume(volumeSpec, cluster.Images)
		Expect(err).NotTo(HaveOccurred())
		args, err := volume.create(context.Background(), temp)
		Expect(err).NotTo(HaveOccurred())
		Expect(args.args()).To(Equal([]string{
			"-drive",
			fmt.Sprintf("if=virtio,cache=writeback,aio=threads,file=%s/root.img", temp),
		}))
		_, err = os.Stat(filepath.Join(temp, "root.img"))
		Expect(err).NotTo(HaveOccurred())
	})

	It("should create a localds volume as specified", func() {
		// Set up runtime
		cur, err := os.Getwd()
		Expect(err).NotTo(HaveOccurred())
		temp := filepath.Join(cur, "temp")
		Expect(os.Mkdir(temp, 0755)).NotTo(HaveOccurred())

		// Create dummy files and directories
		_, err = os.Create("temp/seed_boot-0.yml")
		Expect(err).NotTo(HaveOccurred())
		_, err = os.Create("temp/network.yml")
		Expect(err).NotTo(HaveOccurred())
		defer os.RemoveAll(temp)

		clusterYaml := `
kind: Node
name: boot-0
cpu: 8
memory: 2G
volumes:
- kind: localds
  name: seed
  network-config: temp/network.yml
  user-data: temp/seed_boot-0.yml
smbios:
  serial: fb8f2417d0b4db30050719c31ce02a2e8141bbd8
`
		cluster, err := types.Parse(strings.NewReader(clusterYaml))
		Expect(err).NotTo(HaveOccurred())

		nodeSpec := cluster.Nodes[0]
		volumeSpec := nodeSpec.Volumes[0]

		volume, err := newNodeVolume(volumeSpec, cluster.Images)
		Expect(err).NotTo(HaveOccurred())
		args, err := volume.create(context.Background(), temp)
		Expect(err).NotTo(HaveOccurred())
		Expect(args.args()).To(Equal([]string{
			"-drive",
			fmt.Sprintf("if=virtio,cache=none,aio=native,format=qcow2,file=%s/seed.img", temp),
		}))
		_, err = os.Stat(filepath.Join(temp, "seed.img"))
		Expect(err).NotTo(HaveOccurred())
	})

	It("should create a hostPath volume as specified", func() {
		// Set up runtime
		cur, err := os.Getwd()
		Expect(err).NotTo(HaveOccurred())
		temp := filepath.Join(cur, "temp")
		Expect(os.Mkdir(temp, 0755)).NotTo(HaveOccurred())

		// Create a shared directory
		Expect(os.Mkdir("temp/shared-dir", 0755)).NotTo(HaveOccurred())
		shareDir, err := filepath.Abs("temp/shared-dir")
		Expect(err).NotTo(HaveOccurred())
		defer os.RemoveAll(temp)

		clusterYaml := fmt.Sprintf(`
kind: Node
name: boot-0
cpu: 8
memory: 2G
volumes:
- kind: hostPath
  name: sabakan
  path: %s
smbios:
  serial: fb8f2417d0b4db30050719c31ce02a2e8141bbd8
`, shareDir)
		cluster, err := types.Parse(strings.NewReader(clusterYaml))
		Expect(err).NotTo(HaveOccurred())

		nodeSpec := cluster.Nodes[0]
		volumeSpec := nodeSpec.Volumes[0]

		volume, err := newNodeVolume(volumeSpec, cluster.Images)
		Expect(err).NotTo(HaveOccurred())
		args, err := volume.create(context.Background(), temp)
		Expect(err).NotTo(HaveOccurred())
		Expect(args.args()).To(Equal([]string{
			"-virtfs",
			fmt.Sprintf("local,path=%s/shared-dir,mount_tag=sabakan,security_model=none,readonly", temp),
		}))
	})

	It("should create a raw volume as specified", func() {
		// Set up runtime
		cur, err := os.Getwd()
		Expect(err).NotTo(HaveOccurred())
		temp := filepath.Join(cur, "temp")
		Expect(os.Mkdir(temp, 0755)).NotTo(HaveOccurred())
		defer os.RemoveAll(temp)

		clusterYaml := `
kind: Node
name: boot-0
cpu: 8
memory: 2G
volumes:
- kind: raw
  name: data
  size: 10G
smbios:
  serial: fb8f2417d0b4db30050719c31ce02a2e8141bbd8
`
		cluster, err := types.Parse(strings.NewReader(clusterYaml))
		Expect(err).NotTo(HaveOccurred())

		nodeSpec := cluster.Nodes[0]
		volumeSpec := nodeSpec.Volumes[0]

		volume, err := newNodeVolume(volumeSpec, cluster.Images)
		Expect(err).NotTo(HaveOccurred())
		args, err := volume.create(context.Background(), temp)
		Expect(err).NotTo(HaveOccurred())
		Expect(args.args()).To(Equal([]string{
			"-drive",
			fmt.Sprintf("if=virtio,cache=none,aio=native,format=qcow2,file=%s/data.img", temp),
		}))
	})
})
