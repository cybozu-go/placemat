package virtualbmc

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
)

type rmcpPlusSessionHolder struct {
	sessions map[uint32]*rmcpPlusSession
}

type rmcpPlusSession struct {
	RemoteConsoleSessionId    uint32
	ManagedSystemSessionId    uint32
	RemoteConsoleRandomNumber [16]byte
	ManagedSystemRandomNumber [16]byte
	RequestedPrivilegeLevel   uint8
	ManagedSystemGuid         [16]byte
	UserName                  []byte
	UserNameLength            uint8
	SessionIntegrityKey       []byte
	IntegrityKey              []byte
	ConfidentialityKey        []byte
}

func newRMCPPlusSessionHolder() *rmcpPlusSessionHolder {
	return &rmcpPlusSessionHolder{sessions: make(map[uint32]*rmcpPlusSession)}
}

func (r *rmcpPlusSessionHolder) getNewRMCPPlusSession(remoteConsoleSessionId uint32) (*rmcpPlusSession, error) {
	sessionId, err := generateRandomUint32()
	if err != nil {
		return nil, err
	}
	for {
		if _, ok := r.sessions[*sessionId]; ok {
			sessionId, err = generateRandomUint32()
			if err != nil {
				return nil, err
			}
		} else {
			break
		}
	}

	session := &rmcpPlusSession{}
	session.ManagedSystemSessionId = *sessionId
	session.RemoteConsoleSessionId = remoteConsoleSessionId
	r.sessions[*sessionId] = session

	return session, nil
}

func (r *rmcpPlusSessionHolder) getRMCPPlusSession(id uint32) (*rmcpPlusSession, bool) {
	session, ok := r.sessions[id]
	return session, ok
}

func (r *rmcpPlusSessionHolder) removeRMCPPlusSession(id uint32) {
	_, ok := r.sessions[id]
	if ok {
		delete(r.sessions, id)
	}
}

func generateRandomUint32() (*uint32, error) {
	b := make([]byte, 4)
	_, err := rand.Read(b)
	if err != nil {
		return nil, err
	}

	buf := bytes.NewBuffer(b)
	var num uint32
	if err := binary.Read(buf, binary.LittleEndian, &num); err != nil {
		return nil, err
	}

	return &num, err
}
