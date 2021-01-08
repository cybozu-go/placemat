package virtualbmc

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

// RemoteManagementControlProtocolHeader represents RMCP header
type RemoteManagementControlProtocolHeader struct {
	Version  uint8
	Reserved uint8
	Sequence uint8
	Class    uint8
}

const (
	RmcpVersion1 = 0x06
)

const (
	RmcpClassAsf  = 0x06
	RmcpClassIpmi = 0x07
	RmcpClassOem  = 0x08
)

func HandleRMCPRequest(buf io.Reader, vm VM, session *RMCPPlusSessionHolder, bmcUser *BMCUserHolder) ([]byte, error) {
	rmcp, err := newRMCP(buf)
	if err != nil {
		return nil, err
	}
	res, err := rmcp.handle(buf, vm, session, bmcUser)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func newRMCP(buf io.Reader) (*RemoteManagementControlProtocolHeader, error) {
	header, err := deserializeRMCPHeader(buf)
	if err != nil {
		return nil, err
	}

	return header, nil
}

func (r *RemoteManagementControlProtocolHeader) handle(buf io.Reader, vm VM, session *RMCPPlusSessionHolder, bmcUser *BMCUserHolder) ([]byte, error) {
	var class string
	switch r.Class {
	case RmcpClassIpmi:
		ipmiSession, err := NewIPMISession(buf)
		if err != nil {
			return nil, err
		}
		res, err := ipmiSession.Handle(buf, vm, session, bmcUser)
		if err != nil {
			return nil, err
		}

		return appendRMCPHeader(res)
	case RmcpClassAsf:
		class = "ASF"
	case RmcpClassOem:
		class = "OEM"
	}

	return nil, fmt.Errorf("unsupported Class: %s %d", class, r.Class)
}

func appendRMCPHeader(response []byte) ([]byte, error) {
	obuf := bytes.Buffer{}
	rmcp := buildUpRMCPForIPMI()
	if err := serializeRMCP(&obuf, rmcp); err != nil {
		return nil, err
	}
	if err := binary.Write(&obuf, binary.LittleEndian, response); err != nil {
		return nil, err
	}

	return obuf.Bytes(), nil
}

func deserializeRMCPHeader(buf io.Reader) (*RemoteManagementControlProtocolHeader, error) {
	header := &RemoteManagementControlProtocolHeader{}
	if err := binary.Read(buf, binary.LittleEndian, header); err != nil {
		return nil, err
	}

	return header, nil
}

func serializeRMCP(buf *bytes.Buffer, header RemoteManagementControlProtocolHeader) error {
	if err := binary.Write(buf, binary.LittleEndian, header); err != nil {
		return err
	}

	return nil
}

func buildUpRMCPForIPMI() (rmcp RemoteManagementControlProtocolHeader) {
	rmcp.Version = RmcpVersion1
	rmcp.Reserved = 0x00
	rmcp.Sequence = 0xff
	rmcp.Class = RmcpClassIpmi

	return rmcp
}
