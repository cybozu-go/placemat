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
	ipmiCmdGetDeviceID                = 0x01
	ipmiCmdBroadcastGetDeviceID       = 0x01
	ipmiCmdColdReset                  = 0x02
	ipmiCmdWarmReset                  = 0x03
	ipmiCmdGetSelfTestResults         = 0x04
	ipmiCmdManufacturingTestOn        = 0x05
	ipmiCmdSetACPIPowerState          = 0x06
	ipmiCmdGetACPIPowerState          = 0x07
	ipmiCmdGetDeviceGUID              = 0x08
	ipmiCmdResetWatchdogTimer         = 0x22
	ipmiCmdSetWatchdogTimer           = 0x24
	ipmiCmdGetWatchdogTimer           = 0x25
	ipmiCmdSetBMCGlobalEnables        = 0x2e
	ipmiCmdGetBMCGlobalEnables        = 0x2f
	ipmiCmdClearMSGFlags              = 0x30
	ipmiCmdGetMSGFlags                = 0x31
	ipmiCmdEnableMessageChannelRCV    = 0x32
	ipmiCmdGetMSG                     = 0x33
	ipmiCmdSendMSG                    = 0x34
	ipmiCmdReadEventMSGBuffer         = 0x35
	ipmiCmdGetBTInterfaceCapabilities = 0x36
	ipmiCmdGetSystemGUID              = 0x37
	ipmiCmdGetChannelAuthCapabilities = 0x38
	ipmiCmdGetSessionChallenge        = 0x39
	ipmiCmdActivateSession            = 0x3a
	ipmiCmdSetSessionPrivilege        = 0x3b
	ipmiCmdCloseSession               = 0x3c
	ipmiCmdGetSessionInfo             = 0x3d

	ipmiCmdGetAuthCode                = 0x3f
	ipmiCmdSetChannelAccess           = 0x40
	ipmiCmdGetChannelAccess           = 0x41
	ipmiCmdGetChannelInfo             = 0x42
	ipmiCmdSetUserAccess              = 0x43
	ipmiCmdGetUserAccess              = 0x44
	ipmiCmdSetUserName                = 0x45
	ipmiCmdGetUserName                = 0x46
	ipmiCmdSetUserPassword            = 0x47
	ipmiCmdActivatePayload            = 0x48
	ipmiCmdDeactivatePayload          = 0x49
	ipmiCmdGetPayloadActivationStatus = 0x4a
	ipmiCmdGetPayloadInstanceInfo     = 0x4b
	ipmiCmdSetUserPayloadAccess       = 0x4c
	ipmiCmdGetUserPayloadAccess       = 0x4d
	ipmiCmdGetChannelPayloadSupport   = 0x4e
	ipmiCmdGetChannelPayloadVersion   = 0x4f
	ipmiCmdGetChannelOEMPayloadInfo   = 0x50

	ipmiCmdMasterReadWrite = 0x52

	ipmiCmdGetChannelCipherSuites         = 0x54
	ipmiCmdSuspendResumePayloadEncryption = 0x55
	ipmiCmdSetChannelSecurityKey          = 0x56
	ipmiCmdGetSystemInterfaceCapabilities = 0x57
)

const (
	authBitmaskNone        = 0x01
	authBitmaskMD2         = 0x02
	authBitmaskMD5         = 0x04
	authBitmaskStraightKey = 0x10
	authBitmaskOEM         = 0x20
	authBitmaskIPMIV2      = 0x80
)

const (
	authStatusAnonymous   = 0x01
	authStatusNullUser    = 0x02
	authStatusNonNullUser = 0x04
	authStatusUserLevel   = 0x08
	authStatusPerMessage  = 0x10
	authStatusKG          = 0x20
)

const (
	extendedCapabilitiesChannel15 = 0x01
	extendedCapabilitiesChannel20 = 0x02
)

type ipmiAuthenticationCapabilitiesRequest struct {
	AutnticationTypeSupport uint8
	RequestedPrivilegeLevel uint8
}

type ipmiAuthenticationCapabilitiesResponse struct {
	Channel                   uint8
	AuthenticationTypeSupport uint8
	AuthenticationStatus      uint8
	ExtCapabilities           uint8 // In ipmi v1.5, 0 is always put here. (Reserved)
	OEMID                     [3]uint8
	OEMAuxiliaryData          uint8
}

type ipmiSetSessionPrivilegeLevelRequest struct {
	RequestPrivilegeLevel uint8
}

type ipmiSetSessionPrivilegeLevelResponse struct {
	NewPrivilegeLevel uint8
}

type ipmiCloseSessionRequest struct {
	SessionID uint32
}

func (i *ipmi) handleIPMIApp(message *ipmiMessage) ([]byte, error) {
	switch message.Command {
	case ipmiCmdGetChannelAuthCapabilities:
		log.Info("      ipmi APP: Command = IPMI_CMD_GET_CHANNEL_AUTH_CAPABILITIES", map[string]interface{}{})
		return handleIPMIAuthenticationCapabilities(message)
	case ipmiCmdSetSessionPrivilege:
		log.Info("      ipmi APP: Command = IPMI_CMD_SET_SESSION_PRIVILEGE", map[string]interface{}{})
		return handleIPMISetSessionPrivilegeLevel(message)
	case ipmiCmdCloseSession:
		log.Info("      ipmi APP: Command = IPMI_CMD_CLOSE_SESSION", map[string]interface{}{})
		return nil, i.handleIPMICloseSession(message)
	case ipmiCmdGetDeviceID:
		log.Info("      ipmi APP: Command = IPMI_CMD_GET_DEVICE_ID", map[string]interface{}{})
	case ipmiCmdColdReset:
		log.Info("      ipmi APP: Command = IPMI_CMD_COLD_RESET", map[string]interface{}{})
	case ipmiCmdWarmReset:
		log.Info("      ipmi APP: Command = IPMI_CMD_WARM_RESET", map[string]interface{}{})
	case ipmiCmdGetSelfTestResults:
		log.Info("      ipmi APP: Command = IPMI_CMD_GET_SELF_TEST_RESULTS", map[string]interface{}{})
	case ipmiCmdManufacturingTestOn:
		log.Info("      ipmi APP: Command = IPMI_CMD_MANUFACTURING_TEST_ON", map[string]interface{}{})
	case ipmiCmdSetACPIPowerState:
		log.Info("      ipmi APP: Command = IPMI_CMD_SET_ACPI_POWER_STATE", map[string]interface{}{})
	case ipmiCmdGetACPIPowerState:
		log.Info("      ipmi APP: Command = IPMI_CMD_GET_ACPI_POWER_STATE", map[string]interface{}{})
	case ipmiCmdGetDeviceGUID:
		log.Info("      ipmi APP: Command = IPMI_CMD_GET_DEVICE_GUID", map[string]interface{}{})
	case ipmiCmdResetWatchdogTimer:
		log.Info("      ipmi APP: Command = IPMI_CMD_RESET_WATCHDOG_TIMER", map[string]interface{}{})
	case ipmiCmdSetWatchdogTimer:
		log.Info("      ipmi APP: Command = IPMI_CMD_SET_WATCHDOG_TIMER", map[string]interface{}{})
	case ipmiCmdGetWatchdogTimer:
		log.Info("      ipmi APP: Command = IPMI_CMD_GET_WATCHDOG_TIMER", map[string]interface{}{})
	case ipmiCmdSetBMCGlobalEnables:
		log.Info("      ipmi APP: Command = IPMI_CMD_SET_BMC_GLOBAL_ENABLES", map[string]interface{}{})
	case ipmiCmdGetBMCGlobalEnables:
		log.Info("      ipmi APP: Command = IPMI_CMD_GET_BMC_GLOBAL_ENABLES", map[string]interface{}{})
	case ipmiCmdClearMSGFlags:
		log.Info("      ipmi APP: Command =IPMI_CMD_CLEAR_MSG_FLAGS", map[string]interface{}{})
	case ipmiCmdGetMSGFlags:
		log.Info("      ipmi APP: Command = IPMI_CMD_GET_MSG_FLAGS", map[string]interface{}{})
	case ipmiCmdEnableMessageChannelRCV:
		log.Info("      ipmi APP: Command = IPMI_CMD_ENABLE_MESSAGE_CHANNEL_RCV", map[string]interface{}{})
	case ipmiCmdGetMSG:
		log.Info("      ipmi APP: Command = IPMI_CMD_GET_MSG", map[string]interface{}{})
	case ipmiCmdSendMSG:
		log.Info("      ipmi APP: Command = IPMI_CMD_SEND_MSG", map[string]interface{}{})
	case ipmiCmdReadEventMSGBuffer:
		log.Info("      ipmi APP: Command = IPMI_CMD_READ_EVENT_MSG_BUFFER", map[string]interface{}{})
	case ipmiCmdGetBTInterfaceCapabilities:
		log.Info("      ipmi APP: Command = IPMI_CMD_GET_BT_INTERFACE_CAPABILITIES", map[string]interface{}{})
	case ipmiCmdGetSystemGUID:
		log.Info("      ipmi APP: Command = IPMI_CMD_GET_SYSTEM_GUID", map[string]interface{}{})
	case ipmiCmdGetSessionChallenge:
		log.Info("      ipmi APP: Command = IPMI_CMD_GET_SESSION_CHALLENGE", map[string]interface{}{})
	case ipmiCmdActivateSession:
		log.Info("      ipmi APP: Command = IPMI_CMD_ACTIVATE_SESSION", map[string]interface{}{})
	case ipmiCmdGetSessionInfo:
		log.Info("      ipmi APP: Command = IPMI_CMD_GET_SESSION_INFO", map[string]interface{}{})
	case ipmiCmdGetAuthCode:
		log.Info("      ipmi APP: Command = IPMI_CMD_GET_AUTHCODE", map[string]interface{}{})
	case ipmiCmdSetChannelAccess:
		log.Info("      ipmi APP: Command = IPMI_CMD_SET_CHANNEL_ACCESS", map[string]interface{}{})
	case ipmiCmdGetChannelAccess:
		log.Info("      ipmi APP: Command =IPMI_CMD_GET_CHANNEL_ACCESS", map[string]interface{}{})
	case ipmiCmdGetChannelInfo:
		log.Info("      ipmi APP: Command = IPMI_CMD_GET_CHANNEL_INFO", map[string]interface{}{})
	case ipmiCmdSetUserAccess:
		log.Info("      ipmi APP: Command = IPMI_CMD_SET_USER_ACCESS", map[string]interface{}{})
	case ipmiCmdGetUserAccess:
		log.Info("      ipmi APP: Command = IPMI_CMD_GET_USER_ACCESS", map[string]interface{}{})
	case ipmiCmdSetUserName:
		log.Info("      ipmi APP: Command = IPMI_CMD_SET_USER_NAME", map[string]interface{}{})
	case ipmiCmdGetUserName:
		log.Info("      ipmi APP: Command = IPMI_CMD_GET_USER_NAME", map[string]interface{}{})
	case ipmiCmdSetUserPassword:
		log.Info("      ipmi APP: Command = IPMI_CMD_SET_USER_PASSWORD", map[string]interface{}{})
	case ipmiCmdActivatePayload:
		log.Info("      ipmi APP: Command = IPMI_CMD_ACTIVATE_PAYLOAD", map[string]interface{}{})
	case ipmiCmdDeactivatePayload:
		log.Info("      ipmi APP: Command = IPMI_CMD_DEACTIVATE_PAYLOAD", map[string]interface{}{})
	case ipmiCmdGetPayloadActivationStatus:
		log.Info("      ipmi APP: Command = IPMI_CMD_GET_PAYLOAD_ACTIVATION_STATUS", map[string]interface{}{})
	case ipmiCmdGetPayloadInstanceInfo:
		log.Info("      ipmi APP: Command = IPMI_CMD_GET_PAYLOAD_INSTANCE_INFO", map[string]interface{}{})
	case ipmiCmdSetUserPayloadAccess:
		log.Info("      ipmi APP: Command = IPMI_CMD_SET_USER_PAYLOAD_ACCESS", map[string]interface{}{})
	case ipmiCmdGetUserPayloadAccess:
		log.Info("      ipmi APP: Command = IPMI_CMD_GET_USER_PAYLOAD_ACCESS", map[string]interface{}{})
	case ipmiCmdGetChannelPayloadSupport:
		log.Info("      ipmi APP: Command = IPMI_CMD_GET_CHANNEL_PAYLOAD_SUPPORT", map[string]interface{}{})
	case ipmiCmdGetChannelPayloadVersion:
		log.Info("      ipmi APP: Command = IPMI_CMD_GET_CHANNEL_PAYLOAD_VERSION", map[string]interface{}{})
	case ipmiCmdGetChannelOEMPayloadInfo:
		log.Info("      ipmi APP: Command = IPMI_CMD_GET_CHANNEL_OEM_PAYLOAD_INFO", map[string]interface{}{})
	case ipmiCmdMasterReadWrite:
		log.Info("      ipmi APP: Command = IPMI_CMD_MASTER_READ_WRITE", map[string]interface{}{})
	case ipmiCmdGetChannelCipherSuites:
		log.Info("      ipmi APP: Command = IPMI_CMD_GET_CHANNEL_CIPHER_SUITES", map[string]interface{}{})
	case ipmiCmdSuspendResumePayloadEncryption:
		log.Info("      ipmi APP: Command = IPMI_CMD_SUSPEND_RESUME_PAYLOAD_ENCRYPTION", map[string]interface{}{})
	case ipmiCmdSetChannelSecurityKey:
		log.Info("      ipmi APP: Command = IPMI_CMD_SET_CHANNEL_SECURITY_KEY", map[string]interface{}{})
	case ipmiCmdGetSystemInterfaceCapabilities:
		log.Info("      ipmi APP: Command = IPMI_CMD_GET_SYSTEM_INTERFACE_CAPABILITIES", map[string]interface{}{})
	}

	return nil, fmt.Errorf("unsupported Command: %x", message.Command)
}

func handleIPMISetSessionPrivilegeLevel(message *ipmiMessage) ([]byte, error) {
	buf := bytes.NewBuffer(message.Data)
	request := ipmiSetSessionPrivilegeLevelRequest{}
	if err := binary.Read(buf, binary.LittleEndian, &request); err != nil {
		return nil, fmt.Errorf("failed to read ipmiSetSessionPrivilegeLevelRequest: %w", err)
	}

	response := ipmiSetSessionPrivilegeLevelResponse{}
	response.NewPrivilegeLevel = request.RequestPrivilegeLevel

	dataBuf := bytes.Buffer{}
	if err := binary.Write(&dataBuf, binary.LittleEndian, response); err != nil {
		return nil, err
	}

	return dataBuf.Bytes(), nil
}

func handleIPMIAuthenticationCapabilities(message *ipmiMessage) ([]byte, error) {
	buf := bytes.NewBuffer(message.Data)
	request := ipmiAuthenticationCapabilitiesRequest{}
	if err := binary.Read(buf, binary.LittleEndian, &request); err != nil {
		return nil, fmt.Errorf("failed to read ipmiAuthenticationCapabilitiesRequest: %w", err)
	}

	// prepare for response data
	// We don't simulate OEM related behavior
	response := ipmiAuthenticationCapabilitiesResponse{}
	response.Channel = 1
	response.AuthenticationTypeSupport = authBitmaskIPMIV2 | authBitmaskMD5 | authBitmaskMD2 | authBitmaskNone
	response.AuthenticationStatus = authStatusNonNullUser | authStatusNullUser
	response.ExtCapabilities = extendedCapabilitiesChannel20
	response.OEMAuxiliaryData = 0

	dataBuf := bytes.Buffer{}
	if err := binary.Write(&dataBuf, binary.LittleEndian, response); err != nil {
		return nil, fmt.Errorf("failed to write ipmiAuthenticationCapabilitiesResponse: %w", err)
	}

	return dataBuf.Bytes(), nil
}

func (i *ipmi) handleIPMICloseSession(message *ipmiMessage) error {
	buf := bytes.NewBuffer(message.Data)
	request := ipmiCloseSessionRequest{}
	if err := binary.Read(buf, binary.LittleEndian, &request); err != nil {
		return err
	}

	i.session.removeRMCPPlusSession(request.SessionID)
	return nil
}
