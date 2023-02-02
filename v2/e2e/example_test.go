package e2e

import (
	"encoding/json"
	"errors"
	"os"

	"github.com/cybozu-go/placemat/v2/pkg/placemat"
	"github.com/cybozu-go/placemat/v2/pkg/virtualbmc"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("Example Cluster", func() {
	var session *gexec.Session

	AfterEach(func() {
		terminatePlacemat(session)
	})

	It("should set up nodes", func() {
		By("checking that boot is running", func() {
			session = runPlacemat(exampleClusterYAML)
			status := &placemat.NodeStatus{}
			Eventually(func() error {
				stdout, err := pmctl("node", "show", "boot")
				if err != nil {
					return err
				}
				err = json.Unmarshal(stdout, status)
				if err != nil {
					return err
				}
				if status.PowerStatus != virtualbmc.PowerStatusOn {
					return errors.New("boot is not running")
				}
				return nil
			}).Should(Succeed())
		})

		By("checking that a socket file does not exist on host", func() {
			_, err := os.Stat("/tmp/boot/swtpm.socket")
			Expect(os.IsNotExist(err)).To(BeTrue())
		})

		By("terminating placemat", func() {
			terminatePlacemat(session)
			Eventually(session.Exited).Should(BeClosed())
		})
	})
})
