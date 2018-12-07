package mtest

import (
	"encoding/json"
	"errors"

	"github.com/cybozu-go/placemat/web"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("example launch test", func() {
	It("should launch nodes", func() {
		var session *gexec.Session
		By("checking that boot is running", func() {
			session = runPlacemat(exampleClusterYAML, "-force")
			status := new(web.NodeStatus)
			Eventually(func() error {
				stdout, err := pmctl("node", "show", "boot")
				if err != nil {
					return err
				}
				err = json.Unmarshal(stdout, status)
				if err != nil {
					return err
				}
				if !status.IsRunning {
					return errors.New("boot is not running")
				}
				return nil
			}).Should(Succeed())
		})
		By("terminate placemat", func() {
			terminatePlacemat(session)
			Eventually(session.Exited).Should(BeClosed())
		})
	})
})
