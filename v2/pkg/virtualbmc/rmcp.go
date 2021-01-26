package virtualbmc

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

type remoteManagementControlProtocolHeader struct {
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

func handleRMCPRequest(buf io.Reader, machine Machine, session *rmcpPlusSessionHolder, bmcUser *bmcUserHolder) ([]byte, error) {
	rmcp, err := newRMCP(buf)
	if err != nil {
		return nil, err
	}
	res, err := rmcp.handle(buf, machine, session, bmcUser)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func newRMCP(buf io.Reader) (*remoteManagementControlProtocolHeader, error) {
	header, err := deserializeRMCPHeader(buf)
	if err != nil {
		return nil, err
	}

	return header, nil
}

func (r *remoteManagementControlProtocolHeader) handle(buf io.Reader, machine Machine, session *rmcpPlusSessionHolder, bmcUser *bmcUserHolder) ([]byte, error) {
	var class string
	switch r.Class {
	case RmcpClassIpmi:
		ipmiSession, err := newIPMISession(buf)
		if err != nil {
			return nil, err
		}
		res, err := ipmiSession.handle(buf, machine, session, bmcUser)
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

func deserializeRMCPHeader(buf io.Reader) (*remoteManagementControlProtocolHeader, error) {
	header := &remoteManagementControlProtocolHeader{}
	if err := binary.Read(buf, binary.LittleEndian, header); err != nil {
		return nil, err
	}

	return header, nil
}

func serializeRMCP(buf *bytes.Buffer, header remoteManagementControlProtocolHeader) error {
	if err := binary.Write(buf, binary.LittleEndian, header); err != nil {
		return err
	}

	return nil
}

func buildUpRMCPForIPMI() (rmcp remoteManagementControlProtocolHeader) {
	rmcp.Version = RmcpVersion1
	rmcp.Reserved = 0x00
	rmcp.Sequence = 0xff
	rmcp.Class = RmcpClassIpmi

	return rmcp
}
