package virtualbmc

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/cybozu-go/log"
)

// Chassis Network Function
const (
	ipmiCmdGetChassisCapabilities = 0x00
	ipmiCmdGetChassisStatus       = 0x01
	ipmiCmdChassisControl         = 0x02
	ipmiCmdChassisReset           = 0x03
	ipmiCmdChassisIdentify        = 0x04
	ipmiCmdSetChassisCapabilities = 0x05
	ipmiCmdSetPowerRestorePolicy  = 0x06
	ipmiCmdGetSystemRestartCause  = 0x07
	ipmiCmdSetSystemBootOptions   = 0x08
	ipmiCmdGetSystemBootOptions   = 0x09
	ipmiCmdGetPOHCounter          = 0x0f
)

const (
	chassisControlPowerDown  = 0x00
	chassisControlPowerUp    = 0x01
	chassisControlPowerCycle = 0x02
	chassisControlHardReset  = 0x03
	chassisControlPulse      = 0x04
	chassisControlPowerSoft  = 0x05
)

const chassisPowerStateBitmaskPowerOn = 0x01

type ipmiChassisControlRequest struct {
	ChassisControl uint8
}

type ipmiGetChassisStatusResponse struct {
	CurrentPowerState            uint8
	LastPowerEvent               uint8
	MiscChassisState             uint8
	FrontPanelButtonCapabilities uint8
}

func (i *ipmi) handleIPMIChassis(message *ipmiMessage) ([]byte, error) {
	switch message.Command {
	case ipmiCmdGetChassisStatus:
		log.Info("      ipmi CHASSIS: Command = IPMI_CMD_GET_CHASSIS_STATUS", map[string]interface{}{})
		return i.handleIPMIGetChassisStatus()
	case ipmiCmdChassisControl:
		log.Info("      ipmi CHASSIS: Command = IPMI_CMD_CHASSIS_CONTROL", map[string]interface{}{})
		return nil, i.handleIPMIChassisControl(message)
	case ipmiCmdGetChassisCapabilities:
		log.Info("      ipmi CHASSIS: Command = IPMI_CMD_GET_CHASSIS_CAPABILITIES", map[string]interface{}{})
	case ipmiCmdChassisReset:
		log.Info("      ipmi CHASSIS: Command = IPMI_CMD_CHASSIS_RESET", map[string]interface{}{})
	case ipmiCmdChassisIdentify:
		log.Info("      ipmi CHASSIS: Command = IPMI_CMD_CHASSIS_IDENTIFY", map[string]interface{}{})
	case ipmiCmdSetChassisCapabilities:
		log.Info("      ipmi CHASSIS: Command = IPMI_CMD_SET_CHASSIS_CAPABILITIES", map[string]interface{}{})
	case ipmiCmdSetPowerRestorePolicy:
		log.Info("      ipmi CHASSIS: Command = IPMI_CMD_SET_POWER_RESTORE_POLICY", map[string]interface{}{})
	case ipmiCmdGetSystemRestartCause:
		log.Info("      ipmi CHASSIS: Command = IPMI_CMD_GET_SYSTEM_RESTART_CAUSE", map[string]interface{}{})
	case ipmiCmdSetSystemBootOptions:
		log.Info("      ipmi CHASSIS: Command = IPMI_CMD_SET_SYSTEM_BOOT_OPTIONS", map[string]interface{}{})
	case ipmiCmdGetSystemBootOptions:
		log.Info("      ipmi CHASSIS: Command = IPMI_CMD_GET_SYSTEM_BOOT_OPTIONS", map[string]interface{}{})
	case ipmiCmdGetPOHCounter:
		log.Info("      ipmi CHASSIS: Command = IPMI_CMD_GET_POH_COUNTER", map[string]interface{}{})
	}

	return nil, fmt.Errorf("unsupported Chassis command: %x", message.Command)
}

func (i *ipmi) handleIPMIGetChassisStatus() ([]byte, error) {
	response := ipmiGetChassisStatusResponse{}
	powerStatus := i.machine.PowerStatus()
	if powerStatus == PowerStatusOn || powerStatus == PowerStatusPoweringOn {
		response.CurrentPowerState |= chassisPowerStateBitmaskPowerOn
	}
	response.LastPowerEvent = 0
	response.MiscChassisState = 0
	response.FrontPanelButtonCapabilities = 0

	dataBuf := bytes.Buffer{}
	if err := binary.Write(&dataBuf, binary.LittleEndian, response); err != nil {
		return nil, err
	}

	return dataBuf.Bytes(), nil
}

func (i *ipmi) handleIPMIChassisControl(message *ipmiMessage) error {
	buf := bytes.NewBuffer(message.Data)
	request := ipmiChassisControlRequest{}
	if err := binary.Read(buf, binary.LittleEndian, &request); err != nil {
		return err
	}

	switch request.ChassisControl {
	case chassisControlPowerDown:
		powerState := i.machine.PowerStatus()
		if powerState == PowerStatusOff || powerState == PowerStatusPoweringOff {
			return errors.New("server is already powered off")
		}
		return i.machine.PowerOff()
	case chassisControlPowerUp:
		powerState := i.machine.PowerStatus()
		if powerState == PowerStatusOn || powerState == PowerStatusPoweringOn {
			return errors.New("server is already powered on")
		}
		return i.machine.PowerOn()
	case chassisControlPowerCycle:
		powerState := i.machine.PowerStatus()
		if powerState == PowerStatusOff || powerState == PowerStatusPoweringOff {
			return errors.New("server is already powered off")
		}
		if err := i.machine.PowerOff(); err != nil {
			return err
		}
		return i.machine.PowerOn()
	case chassisControlHardReset:
		powerState := i.machine.PowerStatus()
		if powerState == PowerStatusOff || powerState == PowerStatusPoweringOff {
			return errors.New("server is already powered off")
		}
		if err := i.machine.PowerOff(); err != nil {
			return err
		}
		return i.machine.PowerOn()
	case chassisControlPulse:
		// do nothing
	case chassisControlPowerSoft:
		// do nothing
	}

	return nil
}
