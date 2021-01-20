package virtualbmc

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/cybozu-go/log"
)

// port from OpenIPMI
// Network Functions
const (
	IPMINetFNChassis        = 0x00
	IPMINetFNBridge         = 0x02
	IPMINetFNSensorEvent    = 0x04
	IPMINetFNApp            = 0x06
	IPMINetFNFirmware       = 0x08
	IPMINetFNStorage        = 0x0a
	IPMINetFNTransport      = 0x0c
	IPMINetFNGroupExtension = 0x2c
	IPMINetFNOEMGroup       = 0x2e

	// Response Bit
	IPMINetFNResponse = 0x01
)

type CompletionCode uint8

const (
	CompletionCodeOK                     = CompletionCode(0x00)
	CompletionCodeCouldNotExecuteCommand = CompletionCode(0xd5)
)

// IPMI represents IPMIMessage and a target Machine
type IPMI struct {
	message *IPMIMessage
	machine Machine
	session *RMCPPlusSessionHolder
}

// Length from TargetAddress to Command
const IPMIMessageHeaderLength = 6

// IPMIMessage represents IPMI Message Header and Command data
type IPMIMessage struct {
	TargetAddress  uint8
	TargetLun      uint8 // NetFn (6) + Lun (2)
	Checksum       uint8
	SourceAddress  uint8
	SourceLun      uint8 // SequenceNumber (6) + Lun (2)
	Command        uint8
	CompletionCode CompletionCode
	Data           []uint8
	DataChecksum   uint8
}

// NewIPMI creates an IPMI
func NewIPMI(buf io.Reader, ipmiMessageLen int, machine Machine, session *RMCPPlusSessionHolder) (*IPMI, error) {
	message, err := deserializeIPMIMessage(buf, ipmiMessageLen)
	if err != nil {
		return nil, fmt.Errorf("failed to desetialize IPMI message : %w", err)
	}

	return &IPMI{
		message: message,
		machine: machine,
		session: session,
	}, nil
}

func deserializeIPMIMessage(buf io.Reader, ipmiMessageLen int) (*IPMIMessage, error) {
	header := &IPMIMessage{}

	if err := binary.Read(buf, binary.LittleEndian, &header.TargetAddress); err != nil {
		return nil, fmt.Errorf("failed to read TargetAddress: %w", err)
	}
	if err := binary.Read(buf, binary.LittleEndian, &header.TargetLun); err != nil {
		return nil, fmt.Errorf("failed to read TargetLun: %w", err)
	}
	if err := binary.Read(buf, binary.LittleEndian, &header.Checksum); err != nil {
		return nil, fmt.Errorf("failed to read Checksum: %w", err)
	}
	if err := binary.Read(buf, binary.LittleEndian, &header.SourceAddress); err != nil {
		return nil, fmt.Errorf("failed to read SourceAddress: %w", err)
	}
	if err := binary.Read(buf, binary.LittleEndian, &header.SourceLun); err != nil {
		return nil, fmt.Errorf("failed to read SourceLun: %w", err)
	}
	if err := binary.Read(buf, binary.LittleEndian, &header.Command); err != nil {
		return nil, fmt.Errorf("failed to read Command: %w", err)
	}
	dataLen := ipmiMessageLen - IPMIMessageHeaderLength - 1
	if dataLen > 0 {
		header.Data = make([]uint8, dataLen)
		if err := binary.Read(buf, binary.LittleEndian, &header.Data); err != nil {
			return nil, fmt.Errorf("failed to read Data: %w", err)
		}
	}
	if err := binary.Read(buf, binary.LittleEndian, &header.DataChecksum); err != nil {
		return nil, fmt.Errorf("failed to read DataChecksum: %w", err)
	}

	return header, nil
}

// Handle handles IPMI Message and run the command specified
func (i *IPMI) Handle() ([]byte, error) {
	netFunction := (i.message.TargetLun & 0xFC) >> 2

	switch netFunction {
	case IPMINetFNApp:
		log.Info("    IPMI: NetFunction = APP", map[string]interface{}{})
		code := CompletionCodeOK
		res, err := i.handleIPMIApp(i.message)
		if err != nil {
			code = CompletionCodeCouldNotExecuteCommand
		}
		return appendIPMIMessageHeader(i.message, res, IPMINetFNApp|IPMINetFNResponse, code)
	case IPMINetFNChassis:
		log.Info("    IPMI: NetFunction = CHASSIS", map[string]interface{}{})
		code := CompletionCodeOK
		res, err := i.handleIPMIChassis(i.message)
		if err != nil {
			code = CompletionCodeCouldNotExecuteCommand
		}
		return appendIPMIMessageHeader(i.message, res, IPMINetFNChassis|IPMINetFNResponse, code)
	case IPMINetFNBridge:
		log.Info("    IPMI: NetFunction = BRIDGE", map[string]interface{}{})
	case IPMINetFNSensorEvent:
		log.Info("    IPMI: NetFunction = SENSOR / EVENT", map[string]interface{}{})
	case IPMINetFNFirmware:
		log.Info("    IPMI: NetFunction = FIRMWARE", map[string]interface{}{})
	case IPMINetFNStorage:
		log.Info("    IPMI: NetFunction = STORAGE", map[string]interface{}{})
	case IPMINetFNTransport:
		log.Info("    IPMI: NetFunction = TRANSPORT", map[string]interface{}{})
	case IPMINetFNGroupExtension:
		log.Info("    IPMI: NetFunction = GROUP EXTENSION", map[string]interface{}{})
	case IPMINetFNOEMGroup:
		log.Info("    IPMI: NetFunction = OEM GROUP", map[string]interface{}{})
	default:
		log.Info("    IPMI: NetFunction = Unknown NetFunction", map[string]interface{}{"NetFunction": netFunction})
	}

	return nil, fmt.Errorf("unsupported NetFunction: %x", netFunction)
}

func appendIPMIMessageHeader(request *IPMIMessage, response []byte, netfn uint8, code CompletionCode) ([]byte, error) {
	responseMessage := buildResponseMessageTemplate(request, netfn, code)
	responseMessage.Data = response

	obuf := bytes.Buffer{}
	if err := serializeIPMI(&obuf, responseMessage); err != nil {
		return nil, err
	}

	return obuf.Bytes(), nil
}

func buildResponseMessageTemplate(requestMessage *IPMIMessage, netfn uint8, code CompletionCode) IPMIMessage {
	responseMessage := IPMIMessage{}
	responseMessage.TargetAddress = requestMessage.SourceAddress
	remoteLun := requestMessage.SourceLun & 0x03
	localLun := requestMessage.TargetLun & 0x03
	responseMessage.TargetLun = remoteLun | (netfn << 2)
	responseMessage.SourceAddress = requestMessage.TargetAddress
	responseMessage.SourceLun = (requestMessage.SourceLun & 0xfc) | localLun
	responseMessage.Command = requestMessage.Command
	responseMessage.CompletionCode = code

	return responseMessage
}

func serializeIPMI(buf *bytes.Buffer, message IPMIMessage) error {
	// Calculate data checksum
	sum := uint32(0)
	sum += uint32(message.SourceAddress)
	sum += uint32(message.SourceLun)
	sum += uint32(message.Command)
	sum += uint32(message.CompletionCode)
	for i := 0; i < len(message.Data); i += 1 {
		sum += uint32(message.Data[i])
	}
	message.DataChecksum = uint8(0x100 - (sum & 0xff))

	// Calculate IPMI Message Checksum
	sum = uint32(message.TargetAddress) + uint32(message.TargetLun)
	message.Checksum = uint8(0x100 - (sum & 0xff))

	return serializeIPMIMessage(buf, message)
}

func serializeIPMIMessage(buf *bytes.Buffer, message IPMIMessage) error {
	if err := binary.Write(buf, binary.LittleEndian, message.TargetAddress); err != nil {
		return fmt.Errorf("failed to write TargetAddress: %w", err)
	}
	if err := binary.Write(buf, binary.LittleEndian, message.TargetLun); err != nil {
		return fmt.Errorf("failed to write TargetLun: %w", err)
	}
	if err := binary.Write(buf, binary.LittleEndian, message.Checksum); err != nil {
		return fmt.Errorf("failed to write Checksum: %w", err)
	}
	if err := binary.Write(buf, binary.LittleEndian, message.SourceAddress); err != nil {
		return fmt.Errorf("failed to write SourceAddress: %w", err)
	}
	if err := binary.Write(buf, binary.LittleEndian, message.SourceLun); err != nil {
		return fmt.Errorf("failed to write SourceLun: %w", err)
	}
	if err := binary.Write(buf, binary.LittleEndian, message.Command); err != nil {
		return fmt.Errorf("failed to write Command: %w", err)
	}
	if isNetFunctionResponse(message.TargetLun) {
		if err := binary.Write(buf, binary.LittleEndian, message.CompletionCode); err != nil {
			return fmt.Errorf("failed to write CompletionCode: %w", err)
		}
	}
	buf.Write(message.Data)
	if err := binary.Write(buf, binary.LittleEndian, message.DataChecksum); err != nil {
		return fmt.Errorf("failed to write DataCheckSum: %w", err)
	}

	return nil
}

func isNetFunctionResponse(targetLun uint8) bool {
	return ((targetLun >> 2) & IPMINetFNResponse) == IPMINetFNResponse
}
