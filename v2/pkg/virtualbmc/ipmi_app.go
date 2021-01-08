package virtualbmc

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/cybozu-go/log"
)

// port from OpenIPMI
// App Network Function
const (
	IPMICmdGetDeviceID                = 0x01
	IPMICmdBroadcastGetDeviceID       = 0x01
	IPMICmdColdReset                  = 0x02
	IPMICmdWarmReset                  = 0x03
	IPMICmdGetSelfTestResults         = 0x04
	IPMICmdManufacturingTestOn        = 0x05
	IPMICmdSetACPIPowerState          = 0x06
	IPMICmdGetACPIPowerState          = 0x07
	IPMICmdGetDeviceGUID              = 0x08
	IPMICmdResetWatchdogTimer         = 0x22
	IPMICmdSetWatchdogTimer           = 0x24
	IPMICmdGetWatchdogTimer           = 0x25
	IPMICmdSetBMCGlobalEnables        = 0x2e
	IPMICmdGetBMCGlobalEnables        = 0x2f
	IPMICmdClearMSGFlags              = 0x30
	IPMICmdGetMSGFlags                = 0x31
	IPMICmdEnableMessageChannelRCV    = 0x32
	IPMICmdGetMSG                     = 0x33
	IPMICmdSendMSG                    = 0x34
	IPMICmdReadEventMSGBuffer         = 0x35
	IPMICmdGetBTInterfaceCapabilities = 0x36
	IPMICmdGetSystemGUID              = 0x37
	IPMICmdGetChannelAuthCapabilities = 0x38
	IPMICmdGetSessionChallenge        = 0x39
	IPMICmdActivateSession            = 0x3a
	IPMICmdSetSessionPrivilege        = 0x3b
	IPMICmdCloseSession               = 0x3c
	IPMICmdGetSessionInfo             = 0x3d

	IPMICmdGetAuthCode                = 0x3f
	IPMICmdSetChannelAccess           = 0x40
	IPMICmdGetChannelAccess           = 0x41
	IPMICmdGetChannelInfo             = 0x42
	IPMICmdSetUserAccess              = 0x43
	IPMICmdGetUserAccess              = 0x44
	IPMICmdSetUserName                = 0x45
	IPMICmdGetUserName                = 0x46
	IPMICmdSetUserPassword            = 0x47
	IPMICmdActivatePayload            = 0x48
	IPMICmdDeactivatePayload          = 0x49
	IPMICmdGetPayloadActivationStatus = 0x4a
	IPMICmdGetPayloadInstanceInfo     = 0x4b
	IPMICmdSetUserPayloadAccess       = 0x4c
	IPMICmdGetUserPayloadAccess       = 0x4d
	IPMICmdGetChannelPayloadSupport   = 0x4e
	IPMICmdGetChannelPayloadVersion   = 0x4f
	IPMICmdGetChannelOEMPayloadInfo   = 0x50

	IPMICmdMasterReadWrite = 0x52

	IPMICmdGetChannelCipherSuites         = 0x54
	IPMICmdSuspendResumePayloadEncryption = 0x55
	IPMICmdSetChannelSecurityKey          = 0x56
	IPMICmdGetSystemInterfaceCapabilities = 0x57
)

const (
	AuthBitmaskNone        = 0x01
	AuthBitmaskMD2         = 0x02
	AuthBitmaskMD5         = 0x04
	AuthBitmaskStraightKey = 0x10
	AuthBitmaskOEM         = 0x20
	AuthBitmaskIPMIV2      = 0x80
)

const (
	AuthStatusAnonymous   = 0x01
	AuthStatusNullUser    = 0x02
	AuthStatusNonNullUser = 0x04
	AuthStatusUserLevel   = 0x08
	AuthStatusPerMessage  = 0x10
	AuthStatusKG          = 0x20
)

const (
	ExtendedCapabilitiesChannel15 = 0x01
	ExtendedCapabilitiesChannel20 = 0x02
)

// IPMIAuthenticationCapabilitiesRequest represents Get Authentication Capabilities request
type IPMIAuthenticationCapabilitiesRequest struct {
	AutnticationTypeSupport uint8
	RequestedPrivilegeLevel uint8
}

// IPMIAuthenticationCapabilitiesResponse represents Get Authentication Capabilities response
type IPMIAuthenticationCapabilitiesResponse struct {
	Channel                   uint8
	AuthenticationTypeSupport uint8
	AuthenticationStatus      uint8
	ExtCapabilities           uint8 // In IPMI v1.5, 0 is always put here. (Reserved)
	OEMID                     [3]uint8
	OEMAuxiliaryData          uint8
}

// IPMISetSessionPrivilegeLevelRequest represents Set RMCPPlusSession Privilege Level request
type IPMISetSessionPrivilegeLevelRequest struct {
	RequestPrivilegeLevel uint8
}

// IPMISetSessionPrivilegeLevelResponse represents Set RMCPPlusSession Privilege Level response
type IPMISetSessionPrivilegeLevelResponse struct {
	NewPrivilegeLevel uint8
}

// IPMICloseSessionRequest represents Close RMCPPlusSession request
type IPMICloseSessionRequest struct {
	SessionID uint32
}

func (i *IPMI) handleIPMIApp(message *IPMIMessage) ([]byte, error) {
	switch message.Command {
	case IPMICmdGetChannelAuthCapabilities:
		log.Info("      IPMI APP: Command = IPMI_CMD_GET_CHANNEL_AUTH_CAPABILITIES", map[string]interface{}{})
		return handleIPMIAuthenticationCapabilities(message)
	case IPMICmdSetSessionPrivilege:
		log.Info("      IPMI APP: Command = IPMI_CMD_SET_SESSION_PRIVILEGE", map[string]interface{}{})
		return handleIPMISetSessionPrivilegeLevel(message)
	case IPMICmdCloseSession:
		log.Info("      IPMI APP: Command = IPMI_CMD_CLOSE_SESSION", map[string]interface{}{})
		return nil, i.handleIPMICloseSession(message)
	case IPMICmdGetDeviceID:
		log.Info("      IPMI APP: Command = IPMI_CMD_GET_DEVICE_ID", map[string]interface{}{})
	case IPMICmdColdReset:
		log.Info("      IPMI APP: Command = IPMI_CMD_COLD_RESET", map[string]interface{}{})
	case IPMICmdWarmReset:
		log.Info("      IPMI APP: Command = IPMI_CMD_WARM_RESET", map[string]interface{}{})
	case IPMICmdGetSelfTestResults:
		log.Info("      IPMI APP: Command = IPMI_CMD_GET_SELF_TEST_RESULTS", map[string]interface{}{})
	case IPMICmdManufacturingTestOn:
		log.Info("      IPMI APP: Command = IPMI_CMD_MANUFACTURING_TEST_ON", map[string]interface{}{})
	case IPMICmdSetACPIPowerState:
		log.Info("      IPMI APP: Command = IPMI_CMD_SET_ACPI_POWER_STATE", map[string]interface{}{})
	case IPMICmdGetACPIPowerState:
		log.Info("      IPMI APP: Command = IPMI_CMD_GET_ACPI_POWER_STATE", map[string]interface{}{})
	case IPMICmdGetDeviceGUID:
		log.Info("      IPMI APP: Command = IPMI_CMD_GET_DEVICE_GUID", map[string]interface{}{})
	case IPMICmdResetWatchdogTimer:
		log.Info("      IPMI APP: Command = IPMI_CMD_RESET_WATCHDOG_TIMER", map[string]interface{}{})
	case IPMICmdSetWatchdogTimer:
		log.Info("      IPMI APP: Command = IPMI_CMD_SET_WATCHDOG_TIMER", map[string]interface{}{})
	case IPMICmdGetWatchdogTimer:
		log.Info("      IPMI APP: Command = IPMI_CMD_GET_WATCHDOG_TIMER", map[string]interface{}{})
	case IPMICmdSetBMCGlobalEnables:
		log.Info("      IPMI APP: Command = IPMI_CMD_SET_BMC_GLOBAL_ENABLES", map[string]interface{}{})
	case IPMICmdGetBMCGlobalEnables:
		log.Info("      IPMI APP: Command = IPMI_CMD_GET_BMC_GLOBAL_ENABLES", map[string]interface{}{})
	case IPMICmdClearMSGFlags:
		log.Info("      IPMI APP: Command =IPMI_CMD_CLEAR_MSG_FLAGS", map[string]interface{}{})
	case IPMICmdGetMSGFlags:
		log.Info("      IPMI APP: Command = IPMI_CMD_GET_MSG_FLAGS", map[string]interface{}{})
	case IPMICmdEnableMessageChannelRCV:
		log.Info("      IPMI APP: Command = IPMI_CMD_ENABLE_MESSAGE_CHANNEL_RCV", map[string]interface{}{})
	case IPMICmdGetMSG:
		log.Info("      IPMI APP: Command = IPMI_CMD_GET_MSG", map[string]interface{}{})
	case IPMICmdSendMSG:
		log.Info("      IPMI APP: Command = IPMI_CMD_SEND_MSG", map[string]interface{}{})
	case IPMICmdReadEventMSGBuffer:
		log.Info("      IPMI APP: Command = IPMI_CMD_READ_EVENT_MSG_BUFFER", map[string]interface{}{})
	case IPMICmdGetBTInterfaceCapabilities:
		log.Info("      IPMI APP: Command = IPMI_CMD_GET_BT_INTERFACE_CAPABILITIES", map[string]interface{}{})
	case IPMICmdGetSystemGUID:
		log.Info("      IPMI APP: Command = IPMI_CMD_GET_SYSTEM_GUID", map[string]interface{}{})
	case IPMICmdGetSessionChallenge:
		log.Info("      IPMI APP: Command = IPMI_CMD_GET_SESSION_CHALLENGE", map[string]interface{}{})
	case IPMICmdActivateSession:
		log.Info("      IPMI APP: Command = IPMI_CMD_ACTIVATE_SESSION", map[string]interface{}{})
	case IPMICmdGetSessionInfo:
		log.Info("      IPMI APP: Command = IPMI_CMD_GET_SESSION_INFO", map[string]interface{}{})
	case IPMICmdGetAuthCode:
		log.Info("      IPMI APP: Command = IPMI_CMD_GET_AUTHCODE", map[string]interface{}{})
	case IPMICmdSetChannelAccess:
		log.Info("      IPMI APP: Command = IPMI_CMD_SET_CHANNEL_ACCESS", map[string]interface{}{})
	case IPMICmdGetChannelAccess:
		log.Info("      IPMI APP: Command =IPMI_CMD_GET_CHANNEL_ACCESS", map[string]interface{}{})
	case IPMICmdGetChannelInfo:
		log.Info("      IPMI APP: Command = IPMI_CMD_GET_CHANNEL_INFO", map[string]interface{}{})
	case IPMICmdSetUserAccess:
		log.Info("      IPMI APP: Command = IPMI_CMD_SET_USER_ACCESS", map[string]interface{}{})
	case IPMICmdGetUserAccess:
		log.Info("      IPMI APP: Command = IPMI_CMD_GET_USER_ACCESS", map[string]interface{}{})
	case IPMICmdSetUserName:
		log.Info("      IPMI APP: Command = IPMI_CMD_SET_USER_NAME", map[string]interface{}{})
	case IPMICmdGetUserName:
		log.Info("      IPMI APP: Command = IPMI_CMD_GET_USER_NAME", map[string]interface{}{})
	case IPMICmdSetUserPassword:
		log.Info("      IPMI APP: Command = IPMI_CMD_SET_USER_PASSWORD", map[string]interface{}{})
	case IPMICmdActivatePayload:
		log.Info("      IPMI APP: Command = IPMI_CMD_ACTIVATE_PAYLOAD", map[string]interface{}{})
	case IPMICmdDeactivatePayload:
		log.Info("      IPMI APP: Command = IPMI_CMD_DEACTIVATE_PAYLOAD", map[string]interface{}{})
	case IPMICmdGetPayloadActivationStatus:
		log.Info("      IPMI APP: Command = IPMI_CMD_GET_PAYLOAD_ACTIVATION_STATUS", map[string]interface{}{})
	case IPMICmdGetPayloadInstanceInfo:
		log.Info("      IPMI APP: Command = IPMI_CMD_GET_PAYLOAD_INSTANCE_INFO", map[string]interface{}{})
	case IPMICmdSetUserPayloadAccess:
		log.Info("      IPMI APP: Command = IPMI_CMD_SET_USER_PAYLOAD_ACCESS", map[string]interface{}{})
	case IPMICmdGetUserPayloadAccess:
		log.Info("      IPMI APP: Command = IPMI_CMD_GET_USER_PAYLOAD_ACCESS", map[string]interface{}{})
	case IPMICmdGetChannelPayloadSupport:
		log.Info("      IPMI APP: Command = IPMI_CMD_GET_CHANNEL_PAYLOAD_SUPPORT", map[string]interface{}{})
	case IPMICmdGetChannelPayloadVersion:
		log.Info("      IPMI APP: Command = IPMI_CMD_GET_CHANNEL_PAYLOAD_VERSION", map[string]interface{}{})
	case IPMICmdGetChannelOEMPayloadInfo:
		log.Info("      IPMI APP: Command = IPMI_CMD_GET_CHANNEL_OEM_PAYLOAD_INFO", map[string]interface{}{})
	case IPMICmdMasterReadWrite:
		log.Info("      IPMI APP: Command = IPMI_CMD_MASTER_READ_WRITE", map[string]interface{}{})
	case IPMICmdGetChannelCipherSuites:
		log.Info("      IPMI APP: Command = IPMI_CMD_GET_CHANNEL_CIPHER_SUITES", map[string]interface{}{})
	case IPMICmdSuspendResumePayloadEncryption:
		log.Info("      IPMI APP: Command = IPMI_CMD_SUSPEND_RESUME_PAYLOAD_ENCRYPTION", map[string]interface{}{})
	case IPMICmdSetChannelSecurityKey:
		log.Info("      IPMI APP: Command = IPMI_CMD_SET_CHANNEL_SECURITY_KEY", map[string]interface{}{})
	case IPMICmdGetSystemInterfaceCapabilities:
		log.Info("      IPMI APP: Command = IPMI_CMD_GET_SYSTEM_INTERFACE_CAPABILITIES", map[string]interface{}{})
	}

	return nil, fmt.Errorf("unsupported Command: %x", message.Command)
}

func handleIPMISetSessionPrivilegeLevel(message *IPMIMessage) ([]byte, error) {
	buf := bytes.NewBuffer(message.Data)
	request := IPMISetSessionPrivilegeLevelRequest{}
	if err := binary.Read(buf, binary.LittleEndian, &request); err != nil {
		return nil, fmt.Errorf("failed to read IPMISetSessionPrivilegeLevelRequest: %w", err)
	}

	response := IPMISetSessionPrivilegeLevelResponse{}
	response.NewPrivilegeLevel = request.RequestPrivilegeLevel

	dataBuf := bytes.Buffer{}
	if err := binary.Write(&dataBuf, binary.LittleEndian, response); err != nil {
		return nil, err
	}

	return dataBuf.Bytes(), nil
}

func handleIPMIAuthenticationCapabilities(message *IPMIMessage) ([]byte, error) {
	buf := bytes.NewBuffer(message.Data)
	request := IPMIAuthenticationCapabilitiesRequest{}
	if err := binary.Read(buf, binary.LittleEndian, &request); err != nil {
		return nil, fmt.Errorf("failed to read IPMIAuthenticationCapabilitiesRequest: %w", err)
	}

	// prepare for response data
	// We don't simulate OEM related behavior
	response := IPMIAuthenticationCapabilitiesResponse{}
	response.Channel = 1
	response.AuthenticationTypeSupport = AuthBitmaskIPMIV2 | AuthBitmaskMD5 | AuthBitmaskMD2 | AuthBitmaskNone
	response.AuthenticationStatus = AuthStatusNonNullUser | AuthStatusNullUser
	response.ExtCapabilities = ExtendedCapabilitiesChannel20
	response.OEMAuxiliaryData = 0

	dataBuf := bytes.Buffer{}
	if err := binary.Write(&dataBuf, binary.LittleEndian, response); err != nil {
		return nil, fmt.Errorf("failed to write IPMIAuthenticationCapabilitiesResponse: %w", err)
	}

	return dataBuf.Bytes(), nil
}

func (i *IPMI) handleIPMICloseSession(message *IPMIMessage) error {
	buf := bytes.NewBuffer(message.Data)
	request := IPMICloseSessionRequest{}
	if err := binary.Read(buf, binary.LittleEndian, &request); err != nil {
		return err
	}

	i.session.RemoveRMCPPlusSession(request.SessionID)
	return nil
}
