package virtualbmc

import (
	"context"
	"fmt"
	"net"

	"github.com/cybozu-go/well"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stmcginnis/gofish"
	"github.com/stmcginnis/gofish/redfish"
)

var _ = Describe("Virtual BMC", func() {
	It("should turn on and off Machine power via ipmi v2.0", func() {
		serverAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", "127.0.0.1", 9623))
		Expect(err).NotTo(HaveOccurred())
		conn, err := net.ListenUDP("udp", serverAddr)
		Expect(err).NotTo(HaveOccurred())

		env := well.NewEnvironment(context.Background())
		env.Go(func(ctx context.Context) error {
			return StartIPMIServer(ctx, conn, &MachineMock{status: PowerStatusOff})
		})

		Eventually(func() error {
			// Power State
			ipmipower := well.CommandContext(context.Background(),
				"ipmipower", "--stat", "-u", "cybozu", "-p", "cybozu", "-h", "127.0.0.1:9623", "-D", "LAN_2_0")
			output, err := ipmipower.Output()
			if err != nil {
				return err
			}
			if string(output) != "127.0.0.1: off\n" {
				return fmt.Errorf("ipmipowert stat reponse is not 127.0.0.1: off, actual is: %s", string(output))
			}

			// Power On
			ipmipower = well.CommandContext(context.Background(),
				"ipmipower", "--on", "--wait-until-on", "-u", "cybozu", "-p", "cybozu", "-h", "127.0.0.1:9623", "-D", "LAN_2_0")
			output, err = ipmipower.Output()
			if err != nil {
				return err
			}
			if string(output) != "127.0.0.1: ok\n" {
				return fmt.Errorf("ipmipowert on reponse is not 127.0.0.1: ok, actual is: %s", string(output))
			}

			// Power State
			ipmipower = well.CommandContext(context.Background(),
				"ipmipower", "--stat", "-u", "cybozu", "-p", "cybozu", "-h", "127.0.0.1:9623", "-D", "LAN_2_0")
			output, err = ipmipower.Output()
			if err != nil {
				return err
			}
			if string(output) != "127.0.0.1: on\n" {
				return fmt.Errorf("ipmipowert stat reponse is not 127.0.0.1: on, actual is: %s", string(output))
			}

			// Power Reset
			ipmipower = well.CommandContext(context.Background(),
				"ipmipower", "--reset", "-u", "cybozu", "-p", "cybozu", "-h", "127.0.0.1:9623", "-D", "LAN_2_0")
			output, err = ipmipower.Output()
			if err != nil {
				return err
			}
			if string(output) != "127.0.0.1: ok\n" {
				return fmt.Errorf("ipmipowert reset reponse is not 127.0.0.1: ok, actual is: %s", string(output))
			}

			// Power State
			ipmipower = well.CommandContext(context.Background(),
				"ipmipower", "--stat", "-u", "cybozu", "-p", "cybozu", "-h", "127.0.0.1:9623", "-D", "LAN_2_0")
			output, err = ipmipower.Output()
			if err != nil {
				return err
			}
			if string(output) != "127.0.0.1: on\n" {
				return fmt.Errorf("ipmipowert stat reponse is not 127.0.0.1: on, actual is: %s", string(output))
			}

			// Power Off
			ipmipower = well.CommandContext(context.Background(),
				"ipmipower", "--off", "--wait-until-off", "-u", "cybozu", "-p", "cybozu", "-h", "127.0.0.1:9623", "-D", "LAN_2_0")
			output, err = ipmipower.Output()
			if err != nil {
				return err
			}
			if string(output) != "127.0.0.1: ok\n" {
				return fmt.Errorf("ipmipowert off reponse is not 127.0.0.1: ok, actual is: %s", string(output))
			}

			// Power State
			ipmipower = well.CommandContext(context.Background(),
				"ipmipower", "--stat", "-u", "cybozu", "-p", "cybozu", "-h", "127.0.0.1:9623", "-D", "LAN_2_0")
			output, err = ipmipower.Output()
			if err != nil {
				return err
			}
			if string(output) != "127.0.0.1: off\n" {
				return fmt.Errorf("ipmipowert stat reponse is not 127.0.0.1: off, actual is: %s", string(output))
			}

			return nil
		}).Should(Succeed())

		env.Cancel(nil)
		env.Wait()
	})

	It("should turn on and off Machine power via redfish", func() {
		addr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:%d", "127.0.0.1", 9443))
		Expect(err).NotTo(HaveOccurred())
		listener, err := net.ListenTCP("tcp", addr)
		Expect(err).NotTo(HaveOccurred())

		env := well.NewEnvironment(context.Background())
		env.Go(func(ctx context.Context) error {
			return StartRedfishServer(ctx, listener, &MachineMock{status: PowerStatusOff})
		})

		By("Retrieving a ComputerSystem resource and manipulate it")

		Eventually(func() error {
			config := gofish.ClientConfig{
				Endpoint:  "https://127.0.0.1:9443",
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

			// Check if the powerState is Off
			if system.PowerState != redfish.OffPowerState {
				return fmt.Errorf("powerState is not Off, actual: %s", system.PowerState)
			}

			// Power On
			err = system.Reset(redfish.OnResetType)
			if err != nil {
				return err
			}

			system, err = getComputerSystem(c.Service)
			if err != nil {
				return err
			}

			// Check if the powerState is On
			if system.PowerState != redfish.OnPowerState {
				return fmt.Errorf("powerState is not On, actual: %s", system.PowerState)
			}

			// Force Restart
			err = system.Reset(redfish.ForceRestartResetType)
			if err != nil {
				return err
			}

			system, err = getComputerSystem(c.Service)
			if err != nil {
				return err
			}

			// Check if the powerState is On
			if system.PowerState != redfish.OnPowerState {
				return fmt.Errorf("powerState is not On, actual: %s", system.PowerState)
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

		By("Retrieving a Chassis resource and manipulate it")

		Eventually(func() error {
			config := gofish.ClientConfig{
				Endpoint:  "https://127.0.0.1:9443",
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

			chassis, err := getChassis(c.Service)
			if err != nil {
				return err
			}

			// Check if the powerState is Off
			if chassis.PowerState != redfish.OffPowerState {
				return fmt.Errorf("powerState is not Off, actual: %s", chassis.PowerState)
			}

			// Power On
			err = chassis.Reset(redfish.OnResetType)
			if err != nil {
				return err
			}

			chassis, err = getChassis(c.Service)
			if err != nil {
				return err
			}

			// Check if the powerState is On
			if chassis.PowerState != redfish.OnPowerState {
				return fmt.Errorf("powerState is not On, actual: %s", chassis.PowerState)
			}

			// Force Off
			err = chassis.Reset(redfish.ForceOffResetType)
			if err != nil {
				return err
			}

			chassis, err = getChassis(c.Service)
			if err != nil {
				return err
			}

			// Check if the powerState is Off
			if chassis.PowerState != redfish.OffPowerState {
				return fmt.Errorf("powerState is not Off, actual: %s", chassis.PowerState)
			}

			return nil
		}).Should(Succeed())

		env.Cancel(nil)
		env.Wait()
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

func getChassis(service *gofish.Service) (*redfish.Chassis, error) {
	chassisCollection, err := service.Chassis()
	if err != nil {
		return nil, err
	}

	// Check if the collection contains 1 computer system
	if len(chassisCollection) != 1 {
		return nil, fmt.Errorf("chassis collection length should be 1, actual: %d", len(chassisCollection))
	}

	return chassisCollection[0], nil
}

type MachineMock struct {
	status PowerStatus
}

func (v *MachineMock) PowerStatus() (PowerStatus, error) {
	return v.status, nil
}

func (v *MachineMock) PowerOn() error {
	v.status = PowerStatusOn
	return nil
}

func (v *MachineMock) PowerOff() error {
	v.status = PowerStatusOff
	return nil
}
