package virtualbmc

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

type authenticationType uint8

const (
	authTypeNone                = authenticationType(0x00)
	authTypeMD2                 = authenticationType(0x01)
	authTypeMD5                 = authenticationType(0x02)
	authTypeStraightPasswordKey = authenticationType(0x04)
	authTypeOEM                 = authenticationType(0x05)
	authTypeRMCPPlus            = authenticationType(0x06)
)

type ipmiSession struct {
	authType authenticationType
}

// ipmiSessionWrapper represents ipmi RMCPPlusSession header v1.5 format
// BMCs that support ipmi v2.0/RMCP+ must support the Get Channel Authentication Capabilities command in both the ipmi v1.5 and v2.0 packet formats
// It is recommended that a remote console uses the ipmi v1.5 formats until it has confirmed ipmi v2.0 support
type ipmiSessionWrapper struct {
	AuthenticationType authenticationType
	SequenceNumber     uint32
	SessionId          uint32
	AuthenticationCode [16]byte
	MessageLen         uint8
}

func newIPMISession(buf io.Reader) (*ipmiSession, error) {
	var authType authenticationType
	if err := binary.Read(buf, binary.LittleEndian, &authType); err != nil {
		return nil, fmt.Errorf("failed to read authentication type: %w", err)
	}

	return &ipmiSession{authType: authType}, nil
}

// handle handles both ipmi v1.5 and v2.0 packet formats and dispatches layers below
func (r *ipmiSession) handle(buf io.Reader, machine Machine, session *rmcpPlusSessionHolder, bmcUser *bmcUserHolder) ([]byte, error) {
	rmcpPlus, err := isRMCPPlusFormat(r.authType)
	if err != nil {
		return nil, err
	}

	if rmcpPlus {
		rmcpPlus, err := newRMCPPlus(buf, r.authType, session, bmcUser)
		if err != nil {
			return nil, err
		}
		return rmcpPlus.handle(buf, machine)
	}

	wrapper, err := deserializeIPMISessionWrapper(buf, r.authType)
	if err != nil {
		return nil, err
	}

	ipmi, err := newIPMI(buf, int(wrapper.MessageLen), machine, session)
	if err != nil {
		return nil, err
	}
	res, err := ipmi.handle()
	if err != nil {
		return nil, err
	}

	responseWrapper := ipmiSessionWrapper{
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
		return nil, fmt.Errorf("failed to write ipmi response body: %w", err)
	}
	return obuf.Bytes(), nil
}

func isRMCPPlusFormat(authType authenticationType) (bool, error) {
	switch authType {
	case authTypeRMCPPlus:
		return true, nil
	case authTypeNone:
		return false, nil
	case authTypeMD2:
	case authTypeMD5:
	case authTypeStraightPasswordKey:
	case authTypeOEM:
	default:
	}

	return false, fmt.Errorf("unsupported authentication type %d", authType)
}

func deserializeIPMISessionWrapper(buf io.Reader, authType authenticationType) (*ipmiSessionWrapper, error) {
	wrapper := &ipmiSessionWrapper{}
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

func serializeIPMISessionWrapper(buf *bytes.Buffer, wrapper ipmiSessionWrapper) error {
	if err := binary.Write(buf, binary.LittleEndian, wrapper.AuthenticationType); err != nil {
		return fmt.Errorf("failed to write authenticationType: %w", err)
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
