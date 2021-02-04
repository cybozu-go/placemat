package e2e

import (
	"context"
	"fmt"

	"github.com/cybozu-go/well"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"github.com/stmcginnis/gofish"
	"github.com/stmcginnis/gofish/redfish"
)

const virtualBMCPort = "/dev/virtio-ports/placemat"

var _ = Describe("Virtual BMC", func() {
	var session *gexec.Session

	AfterEach(func() {
		terminatePlacemat(session)
	})

	It("should serve IPMI2.0 and Redfish", func() {
		By("launching", func() {
			session = runPlacemat(clusterYAML)
			Expect(prepareSSHClients(node1, node2)).NotTo(HaveOccurred())
		})

		By("writing to "+virtualBMCPort, func() {
			execSafeAt(node1, "echo", bmc1, "|", "sudo", "dd", "of="+virtualBMCPort)
			execSafeAt(node2, "echo", bmc2, "|", "sudo", "dd", "of="+virtualBMCPort)
		})

		By("serving IPMI2.0", func() {
			Eventually(func() error {
				// Power Off
				ipmipower := well.CommandContext(context.Background(),
					"ipmipower", "--off", "--wait-until-off", "-u", "cybozu", "-p", "cybozu", "-h", bmc1, "-D", "LAN_2_0")
				output, err := ipmipower.Output()
				if err != nil {
					return err
				}
				if string(output) != fmt.Sprintf("%s: ok\n", bmc1) {
					return fmt.Errorf("ipmipowert off reponse is not %s: ok, actual is: %s", bmc1, string(output))
				}

				// Power State
				ipmipower = well.CommandContext(context.Background(),
					"ipmipower", "--stat", "-u", "cybozu", "-p", "cybozu", "-h", bmc1, "-D", "LAN_2_0")
				output, err = ipmipower.Output()
				if err != nil {
					return err
				}
				if string(output) != fmt.Sprintf("%s: off\n", bmc1) {
					return fmt.Errorf("ipmipowert stat reponse is not %s: off, actual is: %s", bmc1, string(output))
				}

				return nil
			}).Should(Succeed())
		})

		By("serving Redfish", func() {
			Eventually(func() error {
				config := gofish.ClientConfig{
					Endpoint:  fmt.Sprintf("https://%s", bmc2),
					Username:  "cybozu",
					Password:  "cybozu",
					BasicAuth: true,
					Insecure:  true,
				}
				c, err := gofish.Connect(config)
				if err != nil {
					return err
				}
				defer c.Logout()

				system, err := getComputerSystem(c.Service)
				if err != nil {
					return err
				}

				// Graceful Shutdown
				err = system.Reset(redfish.GracefulShutdownResetType)
				if err != nil {
					return err
				}

				system, err = getComputerSystem(c.Service)
				if err != nil {
					return err
				}

				// Check if the powerState is Off
				if system.PowerState != redfish.OffPowerState {
					return fmt.Errorf("powerState is not Off, actual: %s", system.PowerState)
				}

				return nil
			}).Should(Succeed())
		})

		By("terminating placemat", func() {
			terminatePlacemat(session)
			Eventually(session.Exited).Should(BeClosed())
		})
	})
})

func getComputerSystem(service *gofish.Service) (*redfish.ComputerSystem, error) {
	systems, err := service.Systems()
	if err != nil {
		return nil, err
	}

	// Check if the collection contains 1 computer system
	if len(systems) != 1 {
		return nil, fmt.Errorf("computer Systems length should be 1, actual: %d", len(systems))
	}

	return systems[0], nil
}
