package placemat

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"strconv"
	"sync"

	"github.com/cybozu-go/cmd"
	"github.com/cybozu-go/log"
	"github.com/rmxymh/infra-ecosphere/bmc"
	"github.com/rmxymh/infra-ecosphere/ipmi"
	"github.com/rmxymh/infra-ecosphere/utils"
)

type bmcServer struct {
	nodeCh   chan bmcInfo
	nodeVMs  map[string]*nodeVM // key: serial
	networks map[string][]*net.IPNet

	mu          sync.Mutex
	nodeSerials map[string]string // key: address
}

func newBMCServer() *bmcServer {
	return &bmcServer{
		nodeCh:      make(chan bmcInfo),
		nodeVMs:     make(map[string]*nodeVM),
		networks:    make(map[string][]*net.IPNet),
		nodeSerials: make(map[string]string),
	}
}

func (s *bmcServer) setup(networks []*Network) error {
	for _, n := range networks {
		if n.Spec.Type == NetworkBMC {
			s.networks[n.Name] = make([]*net.IPNet, len(n.Spec.Addresses))
			for i, address := range n.Spec.Addresses {
				_, ipnet, err := net.ParseCIDR(address)
				if err != nil {
					return err
				}
				s.networks[n.Name][i] = ipnet
			}
		}
	}

	bmc.AddBMCUser("cybozu", "cybozu")

	ipmi.IPMI_CHASSIS_SetHandler(ipmi.IPMI_CMD_GET_CHASSIS_STATUS, s.handleIPMIGetChassisStatus)
	ipmi.IPMI_CHASSIS_SetHandler(ipmi.IPMI_CMD_CHASSIS_CONTROL, s.handleIPMIChassisControl)

	return nil
}

func (s *bmcServer) getVMByAddress(addr string) (*nodeVM, error) {
	s.mu.Lock()
	serial, ok := s.nodeSerials[addr]
	s.mu.Unlock()

	if !ok {
		return nil, errors.New("address not registered: " + addr)
	}

	vm, ok := s.nodeVMs[serial]
	if !ok {
		return nil, errors.New("serial not registered: " + serial)
	}

	return vm, nil
}

// This function is largely copied from github.com/rmxymh/infra-ecosphere,
// which licensed under the MIT License by Yu-Ming Huang.
func (s *bmcServer) handleIPMIGetChassisStatus(addr *net.UDPAddr, server *net.UDPConn, wrapper ipmi.IPMISessionWrapper, message ipmi.IPMIMessage) {
	session, ok := ipmi.GetSession(wrapper.SessionId)
	if !ok {
		fmt.Printf("Unable to find session 0x%08x\n", wrapper.SessionId)
		return
	}

	localIP := utils.GetLocalIP(server)
	vm, err := s.getVMByAddress(localIP)
	if err != nil {
		fmt.Println(err)
		return
	}

	session.Inc()

	response := ipmi.IPMIGetChassisStatusResponse{}
	if vm.isRunning() {
		response.CurrentPowerState |= ipmi.CHASSIS_POWER_STATE_BITMASK_POWER_ON
	}
	response.LastPowerEvent = 0
	response.MiscChassisState = 0
	response.FrontPanelButtonCapabilities = 0

	dataBuf := bytes.Buffer{}
	binary.Write(&dataBuf, binary.LittleEndian, response)

	responseWrapper, responseMessage := ipmi.BuildResponseMessageTemplate(
		wrapper, message, (ipmi.IPMI_NETFN_CHASSIS | ipmi.IPMI_NETFN_RESPONSE), ipmi.IPMI_CMD_GET_CHASSIS_STATUS)
	responseMessage.Data = dataBuf.Bytes()

	responseWrapper.SessionId = wrapper.SessionId
	responseWrapper.SequenceNumber = session.RemoteSessionSequenceNumber
	rmcp := ipmi.BuildUpRMCPForIPMI()

	obuf := bytes.Buffer{}
	ipmi.SerializeRMCP(&obuf, rmcp)
	ipmi.SerializeIPMI(&obuf, responseWrapper, responseMessage, session.User.Password)
	server.WriteToUDP(obuf.Bytes(), addr)
}

// This function is largely copied from github.com/rmxymh/infra-ecosphere,
// which licensed under the MIT License by Yu-Ming Huang.
func (s *bmcServer) handleIPMIChassisControl(addr *net.UDPAddr, server *net.UDPConn, wrapper ipmi.IPMISessionWrapper, message ipmi.IPMIMessage) {
	buf := bytes.NewBuffer(message.Data)
	request := ipmi.IPMIChassisControlRequest{}
	binary.Read(buf, binary.LittleEndian, &request)

	session, ok := ipmi.GetSession(wrapper.SessionId)
	if !ok {
		fmt.Printf("Unable to find session 0x%08x\n", wrapper.SessionId)
		return
	}

	bmcUser := session.User
	code := ipmi.GetAuthenticationCode(wrapper.AuthenticationType, bmcUser.Password, wrapper.SessionId, message, wrapper.SequenceNumber)
	if bytes.Compare(wrapper.AuthenticationCode[:], code[:]) == 0 {
		fmt.Println("      IPMI Authentication Pass.")
	} else {
		fmt.Println("      IPMI Authentication Failed.")
	}

	localIP := utils.GetLocalIP(server)
	vm, err := s.getVMByAddress(localIP)
	if err != nil {
		fmt.Println(err)
		return
	}

	switch request.ChassisControl {
	case ipmi.CHASSIS_CONTROL_POWER_DOWN:
		vm.powerOff()
	case ipmi.CHASSIS_CONTROL_POWER_UP:
		vm.powerOn()
	case ipmi.CHASSIS_CONTROL_POWER_CYCLE:
		vm.powerOff()
		vm.powerOn()
	case ipmi.CHASSIS_CONTROL_HARD_RESET:
		vm.powerOff()
		vm.powerOn()
	case ipmi.CHASSIS_CONTROL_PULSE:
		// do nothing
	case ipmi.CHASSIS_CONTROL_POWER_SOFT:
		//vm.powerSoft()
	}

	session.Inc()

	responseWrapper, responseMessage := ipmi.BuildResponseMessageTemplate(
		wrapper, message, (ipmi.IPMI_NETFN_CHASSIS | ipmi.IPMI_NETFN_RESPONSE), ipmi.IPMI_CMD_CHASSIS_CONTROL)

	responseWrapper.SessionId = wrapper.SessionId
	responseWrapper.SequenceNumber = session.RemoteSessionSequenceNumber
	rmcp := ipmi.BuildUpRMCPForIPMI()

	obuf := bytes.Buffer{}
	ipmi.SerializeRMCP(&obuf, rmcp)
	ipmi.SerializeIPMI(&obuf, responseWrapper, responseMessage, bmcUser.Password)
	server.WriteToUDP(obuf.Bytes(), addr)
}

func (s *bmcServer) listenIPMI(ctx context.Context, addr string) error {
	serverAddr, err := net.ResolveUDPAddr("udp", addr+":623")
	if err != nil {
		return err
	}

	server, err := net.ListenUDP("udp", serverAddr)
	if err != nil {
		return err
	}
	go func() {
		<-ctx.Done()
		server.Close()
	}()

	buf := make([]byte, 1024)
	for {
		_, addr, err := server.ReadFromUDP(buf)
		if err != nil {
			return err
		}

		bytebuf := bytes.NewBuffer(buf)
		ipmi.DeserializeAndExecute(bytebuf, addr, server)
	}
	return nil
}

func (s *bmcServer) handleNode(ctx context.Context) error {
	for {
		select {
		case info := <-s.nodeCh:
			err := s.addPort(ctx, info)
			if err != nil {
				log.Warn("failed to add BMC port", map[string]interface{}{
					log.FnError:   err,
					"serial":      info.serial,
					"bmc_address": info.bmcAddress,
				})
			}
			go s.listenIPMI(ctx, info.bmcAddress)
		case <-ctx.Done():
			return nil
		}
	}
}

func (s *bmcServer) addPort(ctx context.Context, info bmcInfo) error {
	s.mu.Lock()
	s.nodeSerials[info.bmcAddress] = info.serial
	s.mu.Unlock()

	br, network, err := s.findBridge(info.bmcAddress)
	if err != nil {
		return err
	}

	prefixLen, _ := network.Mask.Size()
	address := info.bmcAddress + "/" + strconv.Itoa(prefixLen)

	log.Info("creating BMC port", map[string]interface{}{
		"serial":      info.serial,
		"bmc_address": address,
		"bridge":      br,
	})

	c := cmd.CommandContext(ctx, "ip", "addr", "add", address, "dev", br)
	c.Severity = log.LvDebug
	return c.Run()
}

func (s *bmcServer) findBridge(address string) (string, *net.IPNet, error) {
	ip := net.ParseIP(address)

	for name, network := range s.networks {
		for _, ipnet := range network {
			if ipnet.Contains(ip) {
				return name, ipnet, nil
			}
		}
	}

	return "", nil, errors.New("BMC address not in range of BMC networks: " + address)
}
