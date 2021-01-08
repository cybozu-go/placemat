package virtualbmc

import "math/rand"

// RMCPPlusSessionHolder holds RMCP+ sessions
type RMCPPlusSessionHolder struct {
	sessions map[uint32]*RMCPPlusSession
}

// RMCPPlusSession represents RMCP+ session
type RMCPPlusSession struct {
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

// NewRMCPPlusSessionHolder creates a RMCPPlusSessionHolder
func NewRMCPPlusSessionHolder() *RMCPPlusSessionHolder {
	return &RMCPPlusSessionHolder{sessions: make(map[uint32]*RMCPPlusSession)}
}

// GetNewRMCPPlusSession creates a RMCP+ session and saves it to the holder with specified session ID
func (r *RMCPPlusSessionHolder) GetNewRMCPPlusSession(remoteConsoleSessionId uint32) *RMCPPlusSession {
	sessionId := rand.Uint32()
	for {
		if _, ok := r.sessions[sessionId]; ok {
			sessionId = rand.Uint32()
		} else {
			break
		}
	}

	session := &RMCPPlusSession{}
	session.ManagedSystemSessionId = sessionId
	session.RemoteConsoleSessionId = remoteConsoleSessionId
	r.sessions[sessionId] = session

	return session
}

// GetRMCPPlusSession gets the RMCPPlus session specified from the holder
func (r *RMCPPlusSessionHolder) GetRMCPPlusSession(id uint32) (*RMCPPlusSession, bool) {
	session, ok := r.sessions[id]
	return session, ok
}

// RemoveRMCPPlusSession removes the RMCP+ session specified
func (r *RMCPPlusSessionHolder) RemoveRMCPPlusSession(id uint32) {
	_, ok := r.sessions[id]
	if ok {
		delete(r.sessions, id)
	}
}
