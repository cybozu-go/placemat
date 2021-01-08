package virtualbmc

import (
	"context"
	"fmt"
	"net"
	"strings"

	"github.com/cybozu-go/placemat/v2/pkg/dcnet"
	"github.com/cybozu-go/placemat/v2/pkg/types"
	"github.com/cybozu-go/well"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Virtual BMC", func() {
	It("should turn on and off VM power via IPMI v2.0", func() {
		clusterYaml := `
kind: Network
name: bmc
type: bmc
use-nat: false
address: 10.0.0.1/24
`
		cluster, err := types.Parse(strings.NewReader(clusterYaml))
		Expect(err).NotTo(HaveOccurred())

		// Create bridge networks
		var networks []*dcnet.Network
		for _, network := range cluster.Networks {
			network, err := dcnet.NewNetwork(network)
			Expect(err).NotTo(HaveOccurred())
			Expect(network.Create(1460)).NotTo(HaveOccurred())
			networks = append(networks, network)
		}
		defer func() {
			for _, network := range networks {
				network.Cleanup()
			}
		}()

		_, ipNet, err := net.ParseCIDR("10.0.0.1/24")
		Expect(err).NotTo(HaveOccurred())
		ipmi, err := NewIPMIServer([]*Bridge{
			{
				Name:  "bmc",
				ipNet: ipNet,
			},
		})
		Expect(err).NotTo(HaveOccurred())
		well.Go(func(ctx context.Context) error {
			return ipmi.listen(ctx, "10.0.0.2", 9623, &VMMock{running: false})
		})

		Eventually(func() error {
			ipmipower := well.CommandContext(context.Background(),
				"ipmipower", "--stat", "-u", "cybozu", "-p", "cybozu", "-h", "10.0.0.2:9623", "-D", "LAN_2_0")
			output, err := ipmipower.Output()
			if err != nil {
				return err
			}
			if string(output) != "10.0.0.2: off\n" {
				return fmt.Errorf("ipmipowert stat reponse is not 127.0.0.1: off, actual is: %s", string(output))
			}

			ipmipower = well.CommandContext(context.Background(),
				"ipmipower", "--on", "--wait-until-on", "-u", "cybozu", "-p", "cybozu", "-h", "10.0.0.2:9623", "-D", "LAN_2_0")
			output, err = ipmipower.Output()
			if err != nil {
				return err
			}
			if string(output) != "10.0.0.2: ok\n" {
				return fmt.Errorf("ipmipowert on reponse is not 127.0.0.1: ok, actual is: %s", string(output))
			}

			ipmipower = well.CommandContext(context.Background(),
				"ipmipower", "--stat", "-u", "cybozu", "-p", "cybozu", "-h", "10.0.0.2:9623", "-D", "LAN_2_0")
			output, err = ipmipower.Output()
			if err != nil {
				return err
			}
			if string(output) != "10.0.0.2: on\n" {
				return fmt.Errorf("ipmipowert stat reponse is not 127.0.0.1: on, actual is: %s", string(output))
			}

			ipmipower = well.CommandContext(context.Background(),
				"ipmipower", "--off", "--wait-until-off", "-u", "cybozu", "-p", "cybozu", "-h", "10.0.0.2:9623", "-D", "LAN_2_0")
			output, err = ipmipower.Output()
			if err != nil {
				return err
			}
			if string(output) != "10.0.0.2: ok\n" {
				return fmt.Errorf("ipmipowert off reponse is not 127.0.0.1: ok, actual is: %s", string(output))
			}

			ipmipower = well.CommandContext(context.Background(),
				"ipmipower", "--stat", "-u", "cybozu", "-p", "cybozu", "-h", "10.0.0.2:9623", "-D", "LAN_2_0")
			output, err = ipmipower.Output()
			if err != nil {
				return err
			}
			if string(output) != "10.0.0.2: off\n" {
				return fmt.Errorf("ipmipowert stat reponse is not 127.0.0.1: off, actual is: %s", string(output))
			}

			ipmipower = well.CommandContext(context.Background(),
				"ipmipower", "--reset", "-u", "cybozu", "-p", "cybozu", "-h", "10.0.0.2:9623", "-D", "LAN_2_0")
			output, err = ipmipower.Output()
			if err != nil {
				return err
			}
			if string(output) != "10.0.0.2: ok\n" {
				return fmt.Errorf("ipmipowert reset reponse is not 127.0.0.1: ok, actual is: %s", string(output))
			}

			ipmipower = well.CommandContext(context.Background(),
				"ipmipower", "--stat", "-u", "cybozu", "-p", "cybozu", "-h", "10.0.0.2:9623", "-D", "LAN_2_0")
			output, err = ipmipower.Output()
			if err != nil {
				return err
			}
			if string(output) != "10.0.0.2: on\n" {
				return fmt.Errorf("ipmipowert stat reponse is not 127.0.0.1: on, actual is: %s", string(output))
			}

			return err
		}).Should(Succeed())

		well.Stop()
	})
})

type VMMock struct {
	running bool
}

func (v *VMMock) IsRunning() bool {
	return v.running
}

func (v *VMMock) PowerOn() error {
	v.running = true
	return nil
}

func (v *VMMock) PowerOff() error {
	v.running = false
	return nil
}
