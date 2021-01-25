package e2e

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"

	"github.com/cybozu-go/placemat/v2/cmd/pmctl2/cmd"
	"github.com/cybozu-go/placemat/v2/pkg/placemat"
	"github.com/cybozu-go/placemat/v2/pkg/virtualbmc"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("Placemat", func() {
	var session *gexec.Session

	AfterEach(func() {
		terminatePlacemat(session)
	})

	It("should setup a cluster as specified", func() {
		By("launching", func() {
			session = runPlacemat(clusterYAML)
			Expect(prepareSSHClients(node1, node2)).NotTo(HaveOccurred())
		})

		By("using vhost_net", func() {
			data, err := ioutil.ReadFile("/proc/modules")
			Expect(err).NotTo(HaveOccurred())
			Expect(bytes.Contains(data, []byte("vhost_net"))).To(BeTrue())
		})

		By("creating socket files on a host", func() {
			_, err := os.Stat("/tmp/node1/swtpm.socket")
			Expect(err).NotTo(HaveOccurred())
			_, err = os.Stat("/tmp/node2/swtpm.socket")
			Expect(err).NotTo(HaveOccurred())
		})

		By("creating device files on guests", func() {
			execSafeAt(node1, "test", "-c", "/dev/tpm0")
			execSafeAt(node2, "test", "-c", "/dev/tpm0")
		})

		By("serving node status", func() {
			status := &placemat.NodeStatus{}
			Eventually(func() error {
				stdout, err := pmctl("node", "show", "node1")
				if err != nil {
					return err
				}
				err = json.Unmarshal(stdout, status)
				if err != nil {
					return err
				}
				if status.PowerStatus != virtualbmc.PowerStatusOn {
					return errors.New("node1 is not running")
				}
				return nil
			}).Should(Succeed())
		})

		By("mounting vdc (raw volume, qcow2 format)", func() {
			_, _, err := execAt(node1, "sudo", "dd", "if=/dev/zero", "of=/dev/vdc", "bs=1M", "count=1")
			Expect(err).NotTo(HaveOccurred())
		})

		By("mounting vdd (raw volume, raw format)", func() {
			_, _, err := execAt(node1, "sudo", "dd", "if=/dev/zero", "of=/dev/vdd", "bs=1M", "count=1")
			Expect(err).NotTo(HaveOccurred())
		})

		By("mounting vde (raw volume, qcow2 format, cache=writeback)", func() {
			_, _, err := execAt(node1, "sudo", "dd", "if=/dev/zero", "of=/dev/vde", "bs=1M", "count=1")
			Expect(err).NotTo(HaveOccurred())
		})

		By("sharing files between a host and a guest", func() {
			_, _, err := execAt(node1, "sudo", "mount", "-t", "9p", "-o", "trans=virtio", "data", "/mnt", "-oversion=9p2000.L")
			Expect(err).NotTo(HaveOccurred())
			_, _, err = execAt(node1, "echo", "hello", "|", "sudo", "tee", "/mnt/hello.txt")
			Expect(err).NotTo(HaveOccurred())
			defer execAt(node1, "sudo", "rm", "-f", "/mnt/hello.txt")

			f, err := os.Open("/mnt/placemat/node1/hello.txt")
			Expect(err).NotTo(HaveOccurred())
			defer f.Close()

			b, err := ioutil.ReadAll(f)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(b)).To(Equal("hello\n"))
		})

		By("powering on and off accordingly", func() {
			_, err := pmctl("node", "action", "stop", "node1")
			Expect(err).NotTo(HaveOccurred())

			status := &placemat.NodeStatus{}
			stdout, err := pmctl("node", "show", "node1")
			Expect(err).NotTo(HaveOccurred())
			Expect(json.Unmarshal(stdout, status)).NotTo(HaveOccurred())
			Expect(status.PowerStatus).To(Equal(virtualbmc.PowerStatusOff))

			_, err = pmctl("node", "action", "start", "node1")
			Expect(err).NotTo(HaveOccurred())

			status = &placemat.NodeStatus{}
			stdout, err = pmctl("node", "show", "node1")
			Expect(err).NotTo(HaveOccurred())
			Expect(json.Unmarshal(stdout, status)).NotTo(HaveOccurred())
			Expect(status.PowerStatus).To(Equal(virtualbmc.PowerStatusOn))
		})

		By("forwarding to netns2 through netns1", func() {
			err := exec.Command("sudo", pmctlPath, "forward", "add", "30000", "netns1:"+netns2+":8000").Run()
			Expect(err).NotTo(HaveOccurred())

			var forwards []*cmd.ForwardSetting
			stdout, err := pmctl("forward", "list", "--json")
			Expect(err).NotTo(HaveOccurred())
			err = json.NewDecoder(strings.NewReader(string(stdout))).Decode(&forwards)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(forwards)).Should(Equal(1))
			Expect(forwards[0].LocalPort).Should(Equal(30000))
			Expect(forwards[0].PodName).Should(Equal("netns1"))
			Expect(forwards[0].RemoteHost).Should(Equal(netns2))
			Expect(forwards[0].RemotePort).Should(Equal(8000))

			err = exec.Command("curl", "localhost:30000").Run()
			Expect(err).NotTo(HaveOccurred())

			err = exec.Command("sudo", pmctlPath, "forward", "delete", "30000").Run()
			Expect(err).NotTo(HaveOccurred())
		})

		By("cleaning up garbage when it ends", func() {
			terminatePlacemat(session)
			Eventually(session.Exited).Should(BeClosed())
		})
	})

	It("should run after cleaning up garbage", func() {
		By("launching with force option", func() {
			// Throw away trash
			Expect(os.MkdirAll("/tmp/node1", 0755)).NotTo(HaveOccurred())
			_, err := os.Create("/tmp/node1/swtpm.socket")
			Expect(err).NotTo(HaveOccurred())

			session = runPlacemat(clusterYAML, "--force")
			Expect(prepareSSHClients(node1, node2)).NotTo(HaveOccurred())
		})

		By("terminating placemat", func() {
			terminatePlacemat(session)
			Eventually(session.Exited).Should(BeClosed())
		})
	})
})
