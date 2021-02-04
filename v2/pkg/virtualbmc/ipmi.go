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
	ipmiNetFNChassis        = 0x00
	ipmiNetFNBridge         = 0x02
	ipmiNetFNSensorEvent    = 0x04
	ipmiNetFNApp            = 0x06
	ipmiNetFNFirmware       = 0x08
	ipmiNetFNStorage        = 0x0a
	ipmiNetFNTransport      = 0x0c
	ipmiNetFNGroupExtension = 0x2c
	ipmiNetFNOEMGroup       = 0x2e

	// Response Bit
	ipmiNetFNResponse = 0x01
)

type completionCode uint8

const (
	completionCodeOK                     = completionCode(0x00)
	completionCodeCouldNotExecuteCommand = completionCode(0xd5)
)

type ipmi struct {
	message *ipmiMessage
	machine Machine
	session *rmcpPlusSessionHolder
}

// Length from TargetAddress to Command
const ipmiMessageHeaderLength = 6

type ipmiMessage struct {
	TargetAddress  uint8
	TargetLun      uint8 // NetFn (6) + Lun (2)
	Checksum       uint8
	SourceAddress  uint8
	SourceLun      uint8 // SequenceNumber (6) + Lun (2)
	Command        uint8
	CompletionCode completionCode
	Data           []uint8
	DataChecksum   uint8
}

func newIPMI(buf io.Reader, ipmiMessageLen int, machine Machine, session *rmcpPlusSessionHolder) (*ipmi, error) {
	message, err := deserializeIPMIMessage(buf, ipmiMessageLen)
	if err != nil {
		return nil, fmt.Errorf("failed to desetialize ipmi message : %w", err)
	}

	return &ipmi{
		message: message,
		machine: machine,
		session: session,
	}, nil
}

func deserializeIPMIMessage(buf io.Reader, ipmiMessageLen int) (*ipmiMessage, error) {
	header := &ipmiMessage{}

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
	dataLen := ipmiMessageLen - ipmiMessageHeaderLength - 1
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

func (i *ipmi) handle() ([]byte, error) {
	netFunction := (i.message.TargetLun & 0xFC) >> 2

	switch netFunction {
	case ipmiNetFNApp:
		log.Info("    ipmi: NetFunction = APP", map[string]interface{}{})
		code := completionCodeOK
		res, err := i.handleIPMIApp(i.message)
		if err != nil {
			code = completionCodeCouldNotExecuteCommand
		}
		return appendIPMIMessageHeader(i.message, res, ipmiNetFNApp|ipmiNetFNResponse, code)
	case ipmiNetFNChassis:
		log.Info("    ipmi: NetFunction = CHASSIS", map[string]interface{}{})
		code := completionCodeOK
		res, err := i.handleIPMIChassis(i.message)
		if err != nil {
			code = completionCodeCouldNotExecuteCommand
		}
		return appendIPMIMessageHeader(i.message, res, ipmiNetFNChassis|ipmiNetFNResponse, code)
	case ipmiNetFNBridge:
		log.Info("    ipmi: NetFunction = BRIDGE", map[string]interface{}{})
	case ipmiNetFNSensorEvent:
		log.Info("    ipmi: NetFunction = SENSOR / EVENT", map[string]interface{}{})
	case ipmiNetFNFirmware:
		log.Info("    ipmi: NetFunction = FIRMWARE", map[string]interface{}{})
	case ipmiNetFNStorage:
		log.Info("    ipmi: NetFunction = STORAGE", map[string]interface{}{})
	case ipmiNetFNTransport:
		log.Info("    ipmi: NetFunction = TRANSPORT", map[string]interface{}{})
	case ipmiNetFNGroupExtension:
		log.Info("    ipmi: NetFunction = GROUP EXTENSION", map[string]interface{}{})
	case ipmiNetFNOEMGroup:
		log.Info("    ipmi: NetFunction = OEM GROUP", map[string]interface{}{})
	default:
		log.Info("    ipmi: NetFunction = Unknown NetFunction", map[string]interface{}{"NetFunction": netFunction})
	}

	return nil, fmt.Errorf("unsupported NetFunction: %x", netFunction)
}

func appendIPMIMessageHeader(request *ipmiMessage, response []byte, netfn uint8, code completionCode) ([]byte, error) {
	responseMessage := buildResponseMessageTemplate(request, netfn, code)
	responseMessage.Data = response

	obuf := bytes.Buffer{}
	if err := serializeIPMI(&obuf, responseMessage); err != nil {
		return nil, err
	}

	return obuf.Bytes(), nil
}

func buildResponseMessageTemplate(requestMessage *ipmiMessage, netfn uint8, code completionCode) ipmiMessage {
	responseMessage := ipmiMessage{}
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

func serializeIPMI(buf *bytes.Buffer, message ipmiMessage) error {
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

	// Calculate ipmi Message Checksum
	sum = uint32(message.TargetAddress) + uint32(message.TargetLun)
	message.Checksum = uint8(0x100 - (sum & 0xff))

	return serializeIPMIMessage(buf, message)
}

func serializeIPMIMessage(buf *bytes.Buffer, message ipmiMessage) error {
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
	return ((targetLun >> 2) & ipmiNetFNResponse) == ipmiNetFNResponse
}
