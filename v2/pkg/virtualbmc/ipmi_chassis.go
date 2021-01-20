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
	IPMICmdGetChassisCapabilities = 0x00
	IPMICmdGetChassisStatus       = 0x01
	IPMICmdChassisControl         = 0x02
	IPMICmdChassisReset           = 0x03
	IPMICmdChassisIdentify        = 0x04
	IPMICmdSetChassisCapabilities = 0x05
	IPMICmdSetPowerRestorePolicy  = 0x06
	IPMICmdGetSystemRestartCause  = 0x07
	IPMICmdSetSystemBootOptions   = 0x08
	IPMICmdGetSystemBootOptions   = 0x09
	IPMICmdGetPOHCounter          = 0x0f
)

const (
	ChassisControlPowerDown  = 0x00
	ChassisControlPowerUp    = 0x01
	ChassisControlPowerCycle = 0x02
	ChassisControlHardReset  = 0x03
	ChassisControlPulse      = 0x04
	ChassisControlPowerSoft  = 0x05
)

const ChassisPowerStateBitmaskPowerOn = 0x01

// IPMIChassisControlRequest represents Chassis Control request
type IPMIChassisControlRequest struct {
	ChassisControl uint8
}

// IPMIGetChassisStatusResponse represents Chassis Status response
type IPMIGetChassisStatusResponse struct {
	CurrentPowerState            uint8
	LastPowerEvent               uint8
	MiscChassisState             uint8
	FrontPanelButtonCapabilities uint8
}

func (i *IPMI) handleIPMIChassis(message *IPMIMessage) ([]byte, error) {
	switch message.Command {
	case IPMICmdGetChassisStatus:
		log.Info("      IPMI CHASSIS: Command = IPMI_CMD_GET_CHASSIS_STATUS", map[string]interface{}{})
		return i.handleIPMIGetChassisStatus()
	case IPMICmdChassisControl:
		log.Info("      IPMI CHASSIS: Command = IPMI_CMD_CHASSIS_CONTROL", map[string]interface{}{})
		return nil, i.handleIPMIChassisControl(message)
	case IPMICmdGetChassisCapabilities:
		log.Info("      IPMI CHASSIS: Command = IPMI_CMD_GET_CHASSIS_CAPABILITIES", map[string]interface{}{})
	case IPMICmdChassisReset:
		log.Info("      IPMI CHASSIS: Command = IPMI_CMD_CHASSIS_RESET", map[string]interface{}{})
	case IPMICmdChassisIdentify:
		log.Info("      IPMI CHASSIS: Command = IPMI_CMD_CHASSIS_IDENTIFY", map[string]interface{}{})
	case IPMICmdSetChassisCapabilities:
		log.Info("      IPMI CHASSIS: Command = IPMI_CMD_SET_CHASSIS_CAPABILITIES", map[string]interface{}{})
	case IPMICmdSetPowerRestorePolicy:
		log.Info("      IPMI CHASSIS: Command = IPMI_CMD_SET_POWER_RESTORE_POLICY", map[string]interface{}{})
	case IPMICmdGetSystemRestartCause:
		log.Info("      IPMI CHASSIS: Command = IPMI_CMD_GET_SYSTEM_RESTART_CAUSE", map[string]interface{}{})
	case IPMICmdSetSystemBootOptions:
		log.Info("      IPMI CHASSIS: Command = IPMI_CMD_SET_SYSTEM_BOOT_OPTIONS", map[string]interface{}{})
	case IPMICmdGetSystemBootOptions:
		log.Info("      IPMI CHASSIS: Command = IPMI_CMD_GET_SYSTEM_BOOT_OPTIONS", map[string]interface{}{})
	case IPMICmdGetPOHCounter:
		log.Info("      IPMI CHASSIS: Command = IPMI_CMD_GET_POH_COUNTER", map[string]interface{}{})
	}

	return nil, fmt.Errorf("unsupported Chassis command: %x", message.Command)
}

func (i *IPMI) handleIPMIGetChassisStatus() ([]byte, error) {
	response := IPMIGetChassisStatusResponse{}
	powerStatus := i.machine.PowerStatus()
	if powerStatus == PowerStatusOn || powerStatus == PowerStatusPoweringOn {
		response.CurrentPowerState |= ChassisPowerStateBitmaskPowerOn
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

func (i *IPMI) handleIPMIChassisControl(message *IPMIMessage) error {
	buf := bytes.NewBuffer(message.Data)
	request := IPMIChassisControlRequest{}
	if err := binary.Read(buf, binary.LittleEndian, &request); err != nil {
		return err
	}

	switch request.ChassisControl {
	case ChassisControlPowerDown:
		powerState := i.machine.PowerStatus()
		if powerState == PowerStatusOff || powerState == PowerStatusPoweringOff {
			return errors.New("server is already powered off")
		}
		return i.machine.PowerOff()
	case ChassisControlPowerUp:
		powerState := i.machine.PowerStatus()
		if powerState == PowerStatusOn || powerState == PowerStatusPoweringOn {
			return errors.New("server is already powered on")
		}
		return i.machine.PowerOn()
	case ChassisControlPowerCycle:
		powerState := i.machine.PowerStatus()
		if powerState == PowerStatusOff || powerState == PowerStatusPoweringOff {
			return errors.New("server is already powered off")
		}
		if err := i.machine.PowerOff(); err != nil {
			return err
		}
		return i.machine.PowerOn()
	case ChassisControlHardReset:
		powerState := i.machine.PowerStatus()
		if powerState == PowerStatusOff || powerState == PowerStatusPoweringOff {
			return errors.New("server is already powered off")
		}
		if err := i.machine.PowerOff(); err != nil {
			return err
		}
		return i.machine.PowerOn()
	case ChassisControlPulse:
		// do nothing
	case ChassisControlPowerSoft:
		// do nothing
	}

	return nil
}
