package mtest

import (
	"encoding/json"
	"errors"
	"strconv"
	"strings"

	"github.com/cybozu-go/placemat/web"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("pod launch test", func() {
	It("should launch pods", func() {
		var session *gexec.Session
		By("launch placemat", func() {
			session = runPlacemat(clusterYaml, "-force")
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

		By("terminate placemat", func() {
			terminatePlacemat(session)
			Eventually(session.Exited).Should(BeClosed())
		})
	})
})
