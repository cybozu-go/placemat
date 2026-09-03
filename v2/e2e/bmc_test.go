package e2e

import (
	"context"
	"fmt"

	"github.com/cybozu-go/well"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"github.com/stmcginnis/gofish"
	"github.com/stmcginnis/gofish/schemas"
)

const virtualBMCPort = "/dev/virtio-ports/placemat"

var _ = Describe("Virtual BMC", func() {
	var session *gexec.Session

	AfterEach(func() {
		_, _ = terminatePlacemat(session)
	})

	It("should serve IPMI2.0 and Redfish", func() {
		By("launching", func() {
			session = runPlacemat(clusterYAML)
			Expect(prepareSSHClients(node1, node2)).NotTo(HaveOccurred())
		})

		By("writing to "+virtualBMCPort, func() {
			execSafeAt(node1, "echo", bmc1, "|", "sudo", "tee", virtualBMCPort, ">", "/dev/null")
			execSafeAt(node2, "echo", bmc2, "|", "sudo", "tee", virtualBMCPort, ">", "/dev/null")
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
			config := gofish.ClientConfig{
				Endpoint:  fmt.Sprintf("https://%s", bmc2),
				Username:  "cybozu",
				Password:  "cybozu",
				BasicAuth: true,
				Insecure:  true,
			}
			var c *gofish.APIClient
			Eventually(func() error {
				var err error
				c, err = gofish.Connect(config)
				return err
			}).Should(Succeed())
			defer c.Logout()

			system, err := getComputerSystem(c.Service)
			Expect(err).NotTo(HaveOccurred())

			// Request a graceful shutdown once; the guest shuts down asynchronously.
			taskMonitor, err := system.Reset(schemas.GracefulShutdownResetType)
			Expect(err).NotTo(HaveOccurred())
			if taskMonitor != nil {
				_, err := schemas.WaitForTaskMonitor(context.Background(), c, 0, taskMonitor, nil)
				Expect(err).NotTo(HaveOccurred())
			}

			// Poll until the guest powers off.
			Eventually(func() error {
				system, err := getComputerSystem(c.Service)
				if err != nil {
					return err
				}
				if system.PowerState != schemas.OffPowerState {
					return fmt.Errorf("powerState is not Off, actual: %s", system.PowerState)
				}
				return nil
			}).Should(Succeed())
		})

		By("terminating placemat", func() {
			_, _ = terminatePlacemat(session)
			Eventually(session.Exited).Should(BeClosed())
		})
	})
})

func getComputerSystem(service *gofish.Service) (*schemas.ComputerSystem, error) {
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
