package virtualbmc

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

type AuthenticationType uint8

const (
	AuthTypeNone                AuthenticationType = 0x00
	AuthTypeMD2                 AuthenticationType = 0x01
	AuthTypeMD5                 AuthenticationType = 0x02
	AuthTypeStraightPasswordKey AuthenticationType = 0x04
	AuthTypeOEM                 AuthenticationType = 0x05
	AuthTypeRMCPPlus            AuthenticationType = 0x06
)

// IPMISession represents IPMI RMCPPlusSession
type IPMISession struct {
	authType AuthenticationType
}

// IPMISessionWrapper represents IPMI RMCPPlusSession header v1.5 format
// BMCs that support IPMI v2.0/RMCP+ must support the Get Channel Authentication Capabilities command in both the IPMI v1.5 and v2.0 packet formats
// It is recommended that a remote console uses the IPMI v1.5 formats until it has confirmed IPMI v2.0 support
type IPMISessionWrapper struct {
	AuthenticationType AuthenticationType
	SequenceNumber     uint32
	SessionId          uint32
	AuthenticationCode [16]byte
	MessageLen         uint8
}

// NewIPMISession creates IPMISession
func NewIPMISession(buf io.Reader) (*IPMISession, error) {
	var authType AuthenticationType
	if err := binary.Read(buf, binary.LittleEndian, &authType); err != nil {
		return nil, fmt.Errorf("failed to read authentication type: %w", err)
	}

	return &IPMISession{authType: authType}, nil
}

// Handle handles both IPMI v1.5 and v2.0 packet formats and dispatches layers below
func (r *IPMISession) Handle(buf io.Reader, machine Machine, session *RMCPPlusSessionHolder, bmcUser *BMCUserHolder) ([]byte, error) {
	rmcpPlus, err := isRMCPPlusFormat(r.authType)
	if err != nil {
		return nil, err
	}

	if rmcpPlus {
		rmcpPlus, err := NewRMCPPlus(buf, r.authType, session, bmcUser)
		if err != nil {
			return nil, err
		}
		return rmcpPlus.Handle(buf, machine)
	}

	wrapper, err := deserializeIPMISessionWrapper(buf, r.authType)
	if err != nil {
		return nil, err
	}

	ipmi, err := NewIPMI(buf, int(wrapper.MessageLen), machine, session)
	if err != nil {
		return nil, err
	}
	res, err := ipmi.Handle()
	if err != nil {
		return nil, err
	}

	responseWrapper := IPMISessionWrapper{
		AuthenticationType: r.authType,
		SequenceNumber:     wrapper.SequenceNumber,
		SessionId:          wrapper.SessionId,
		MessageLen:         uint8(len(res)),
	}

	obuf := bytes.Buffer{}
	if err := serializeIPMISessionWrapper(&obuf, responseWrapper); err != nil {
		return nil, err
	}
	if err := binary.Write(&obuf, binary.LittleEndian, res); err != nil {
		return nil, fmt.Errorf("failed to write IPMI response body: %w", err)
	}
	return obuf.Bytes(), nil
}

func isRMCPPlusFormat(authType AuthenticationType) (bool, error) {
	switch authType {
	case AuthTypeRMCPPlus:
		return true, nil
	case AuthTypeNone:
		return false, nil
	case AuthTypeMD2:
	case AuthTypeMD5:
	case AuthTypeStraightPasswordKey:
	case AuthTypeOEM:
	default:
	}

	return false, fmt.Errorf("unsupported authentication type %d", authType)
}

func deserializeIPMISessionWrapper(buf io.Reader, authType AuthenticationType) (*IPMISessionWrapper, error) {
	wrapper := &IPMISessionWrapper{}
	wrapper.AuthenticationType = authType
	if err := binary.Read(buf, binary.LittleEndian, &wrapper.SequenceNumber); err != nil {
		return nil, fmt.Errorf("failed to read SequenceNumber: %w", err)
	}
	if err := binary.Read(buf, binary.LittleEndian, &wrapper.SessionId); err != nil {
		return nil, fmt.Errorf("failed to read SessionId: %w", err)
	}
	if wrapper.SessionId != 0x00 {
		if err := binary.Read(buf, binary.LittleEndian, &wrapper.AuthenticationCode); err != nil {
			return nil, fmt.Errorf("failed to read AuthenticationCode: %w", err)
		}
	}
	if err := binary.Read(buf, binary.LittleEndian, &wrapper.MessageLen); err != nil {
		return nil, fmt.Errorf("failed to read MessageLen: %w", err)
	}
	return wrapper, nil
}

func serializeIPMISessionWrapper(buf *bytes.Buffer, wrapper IPMISessionWrapper) error {
	if err := binary.Write(buf, binary.LittleEndian, wrapper.AuthenticationType); err != nil {
		return fmt.Errorf("failed to write AuthenticationType: %w", err)
	}
	if err := binary.Write(buf, binary.LittleEndian, wrapper.SequenceNumber); err != nil {
		return fmt.Errorf("failed to write SequenceNumber: %w", err)
	}
	if err := binary.Write(buf, binary.LittleEndian, wrapper.SessionId); err != nil {
		return fmt.Errorf("failed to write SessionId: %w", err)
	}
	if wrapper.SessionId != 0 {
		if err := binary.Write(buf, binary.LittleEndian, wrapper.AuthenticationCode); err != nil {
			return fmt.Errorf("failed to write AuthenticationCode: %w", err)
		}
	}
	if err := binary.Write(buf, binary.LittleEndian, wrapper.MessageLen); err != nil {
		return fmt.Errorf("failed to write MessageLen: %w", err)
	}

	return nil
}
