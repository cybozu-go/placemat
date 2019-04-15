package mtest

import (
	"encoding/json"
	"errors"
	"os/exec"
	"strconv"
	"strings"

	pmctlCmd "github.com/cybozu-go/placemat/pkg/pmctl/cmd"
	"github.com/cybozu-go/placemat/web"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

// TestPod tests pod launch
func TestPod() {
	It("should launch pods", func() {
		var session *gexec.Session
		By("launch placemat", func() {
			session = runPlacemat(clusterYAML, "-force")
			err := prepareSSHClients(node1, node2)
			Expect(err).To(Succeed())
		})

		By("check pod", func() {
			status := new(web.PodStatus)
			Eventually(func() error {
				stdout, err := pmctl("pod", "show", "pod1")
				if err != nil {
					return err
				}
				err = json.Unmarshal(stdout, status)
				if err != nil {
					return err
				}
				if status.PID == 0 {
					return errors.New("pid is empty")
				}
				if len(status.UUID) == 0 {
					return errors.New("uuid is empty")
				}
				return nil
			}).Should(Succeed())

			stdout, err := rkt("status", status.UUID)
			Expect(err).NotTo(HaveOccurred())
			rktStatus := make(map[string]string)
			for _, line := range strings.Split(string(stdout), "\n") {
				items := strings.Split(line, "=")
				if len(items) == 2 {
					rktStatus[strings.TrimSpace(items[0])] = strings.TrimSpace(items[1])
				}
			}
			Expect(rktStatus["state"]).Should(Equal("running"))
			Expect(rktStatus["pid"]).Should(Equal(strconv.Itoa(status.PID)))
		})

		By("forwarding to pod2 through pod1", func() {
			err := exec.Command("sudo", pmctlPath, "forward", "add", "30000", "pod1:"+pod2+":80").Run()
			Expect(err).NotTo(HaveOccurred())

			var forwards []*pmctlCmd.ForwardSetting
			stdout, err := pmctl("forward", "list", "--json")
			err = json.NewDecoder(strings.NewReader(string(stdout))).Decode(&forwards)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(forwards)).Should(Equal(1))
			Expect(forwards[0].LocalPort).Should(Equal(30000))
			Expect(forwards[0].PodName).Should(Equal("pod1"))
			Expect(forwards[0].RemoteHost).Should(Equal(pod2))
			Expect(forwards[0].RemotePort).Should(Equal(80))

			err = exec.Command("curl", "localhost:30000").Run()
			Expect(err).NotTo(HaveOccurred())

			err = exec.Command("sudo", pmctlPath, "forward", "delete", "30000").Run()
			Expect(err).NotTo(HaveOccurred())
		})

		By("terminate placemat", func() {
			terminatePlacemat(session)
			Eventually(session.Exited).Should(BeClosed())
		})
	})
}
