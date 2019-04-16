package mtest

import (
	"encoding/json"
	"errors"
	"strings"

	"github.com/cybozu-go/placemat/web"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

// TestExample tests example launch
func TestExample() {
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
		By("saving a snapshot", func() {
			_, err := pmctl("snapshot", "save", "test")
			Expect(err).NotTo(HaveOccurred())
		})
		By("loading a snapshot", func() {
			_, err := pmctl("snapshot", "load", "test")
			Expect(err).NotTo(HaveOccurred())
		})
		By("listing snapshots", func() {
			out, err := pmctl("snapshot", "list")
			Expect(err).NotTo(HaveOccurred())
			var result map[string]interface{}
			err = json.NewDecoder(strings.NewReader(string(out))).Decode(&result)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(result)).Should(Equal(1))
			for _, node := range result {
				Expect(node).NotTo(Equal("There is no snapshot available."))
			}
		})
		By("terminate placemat", func() {
			terminatePlacemat(session)
			Eventually(session.Exited).Should(BeClosed())
		})
	})
}
