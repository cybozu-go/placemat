package virtualbmc

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"io"
)

const (
	payloadTypeRMCPPlusOpenSessionRequest  = 0x10
	payloadTypeRMCPPlusOpenSessionResponse = 0x11
	payloadTypeRAKPMessage1                = 0x12
	payloadTypeRAKPMessage2                = 0x13
	payloadTypeRAKPMessage3                = 0x14
	payloadTypeRAKPMessage4                = 0x15
)

type maximumPrivilegeLevel uint8

// 0x01: Callback, 0x02: User, 0x03: Operator. 0x04: Administrator, 0x05: OEM
// We only use Administrator
const maximumPrivilegeLevelAdministrator maximumPrivilegeLevel = 0x04

type rmcpStatus uint8

const (
	rmcpPlusStatusNoErrors rmcpStatus = 0x00
)

type authenticationAlgorithm uint8

// 0x00: RKAP-SHA1, 0x01: RKAP-HMAC-SHA1, 0x02: RKAP-HMAC-MD5, 0x03: RKAP-HMAC-SHA256
// We only support RKAP-HMAC-SHA1 which is default authentication algorithm
const authenticationAlgorithmRKAPHMACSHA1 authenticationAlgorithm = 0x01

type integrityAlgorithm uint8

// 0x01: HMAC-SHA1-196, 0x02: HMAC-MD5-128, 0x03: MD5-128, 0x04: HMAC-SHA-256-128
// We only support HMAC-SHA1-196 which is default integrity algorithm
const integrityAlgorithmHMACSHA1196 integrityAlgorithm = 0x01

type confidentialityAlgorithm uint8

// 0x01: AES-CBC-128, 0x02: XRC-4128, 0x03: XRC-440
// We only support AES-CBC-128 which is default confidentiality algorithm
const confidentialityAlgorithmAESCBC128 confidentialityAlgorithm = 0x01

const authenticationPayloadTypeAuthenticationAlgorithm = 0x00
const integrityPayloadTypeIntegrityAlgorithm = 0x01
const confidentialityPayloadTypeConfidentialityAlgorithm = 0x02

// rmcpPlus represents RMCP+
type rmcpPlus struct {
	header  *rmcpPlusSessionHeader
	session *rmcpPlusSessionHolder
	bmcUser *bmcUserHolder
}

// rmcpPlusSessionHeader represents RMCP+ RMCPPlusSession Header
type rmcpPlusSessionHeader struct {
	AuthenticationType authenticationType
	// encrypted(1b) + authenticated(1b) + payloadType(6b)
	PayloadType           uint8
	SessionId             uint32
	SessionSequenceNumber uint32
	IpmiPayloadLen        uint16
}

// openSessionRequestPayload represents Open RMCPPlusSession Request payload
type openSessionRequestPayload struct {
	MessageTag                      uint8
	RequestedMaximumPrivilegeLevel  maximumPrivilegeLevel
	Reserved2                       [2]byte
	RemoteConsoleSessionId          uint32
	AuthenticationPayloadType       uint8
	Reserved3                       [2]byte
	AuthenticationPayloadLength     uint8
	AuthenticationPayloadAlgorithm  authenticationAlgorithm
	Reserved5                       [3]byte
	IntegrityPayloadType            uint8
	Reserved6                       [2]byte
	IntegrityPayloadLength          uint8
	IntegrityPayloadAlgorithm       integrityAlgorithm
	Reserved8                       [3]byte
	ConfidentialityPayloadType      uint8
	Reserved9                       [2]byte
	ConfidentialityPayloadLength    uint8
	ConfidentialityPayloadAlgorithm confidentialityAlgorithm
	Reserved11                      [3]byte
}

// openSessionResponsePayload represents Open RMCPPlusSession Response payload
type openSessionResponsePayload struct {
	MessageTag                      uint8
	RmcpPlusStatusCode              rmcpStatus
	MaximumPrivilegeLevel           maximumPrivilegeLevel
	Reserved2                       [1]byte
	RemoteConsoleSessionId          uint32
	ManagedSystemSessionId          uint32
	AuthenticationPayloadType       uint8
	Reserved3                       [2]byte
	AuthenticationPayloadLength     uint8
	AuthenticationPayloadAlgorithm  authenticationAlgorithm
	Reserved5                       [3]byte
	IntegrityPayloadType            uint8
	Reserved6                       [2]byte
	IntegrityPayloadLength          uint8
	IntegrityPayloadAlgorithm       integrityAlgorithm
	Reserved8                       [3]byte
	ConfidentialityPayloadType      uint8
	Reserved9                       [2]byte
	ConfidentialityPayloadLength    uint8
	ConfidentialityPayloadAlgorithm confidentialityAlgorithm
	Reserved11                      [3]byte
}

// rakpMessage1Request represents RAKP Message1 request
type rakpMessage1Request struct {
	MessageTag                uint8
	Reserved1                 [3]byte
	ManagedSystemSessionId    uint32
	RemoteConsoleRandomNumber [16]byte
	// reserved2(3b) + name_only_lookup(1b) + requested_maximum_privilege_level(4b)
	RequestedMaximumPrivilegeLevelAndNameOnlyLookup uint8
	Reserved3                                       [2]byte
	UserNameLength                                  uint8
	UserName                                        [20]byte
}

// rakpMessage2Response represents RAKP Message2 response
type rakpMessage2Response struct {
	MessageTag                    uint8
	RmcpPlusStatusCode            rmcpStatus
	Reserved1                     [2]byte
	RemoteConsoleSessionId        uint32
	ManagedSystemRandomNumber     [16]byte
	ManagedSystemGuid             [16]byte
	KeyExchangeAuthenticationCode [20]byte
}

// rakpMessage3Request represents RAKP Message3 request
type rakpMessage3Request struct {
	MessageTag                    uint8
	RmcpPlusStatusCode            rmcpStatus
	Reserved1                     [2]byte
	ManagedSystemSessionId        uint32
	KeyExchangeAuthenticationCode [20]byte
}

// rakpMessage4Response represents RAKP Message4 response
type rakpMessage4Response struct {
	MessageTag             uint8
	RmcpPlusStatusCode     rmcpStatus
	Reserved1              [2]byte
	RemoteConsoleSessionId uint32
	IntegrityCheckValue    [12]byte
}

// ipmiSessionTrailer represents ipmi RMCPPlusSession Trailer added to the end of the encrypted packet
type ipmiSessionTrailer struct {
	IntegrityPad [2]byte
	PadLength    uint8
	NextHeader   uint8
}

func newRMCPPlus(buf io.Reader, authType authenticationType, session *rmcpPlusSessionHolder, bmcUser *bmcUserHolder) (*rmcpPlus, error) {
	header, err := deserializeRMCPPlusSessionHeader(buf, authType)
	if err != nil {
		return nil, err
	}

	return &rmcpPlus{
		header:  header,
		session: session,
		bmcUser: bmcUser,
	}, nil
}

// handle handles RMCP+ format request
func (r *rmcpPlus) handle(buf io.Reader, machine Machine) ([]byte, error) {
	payloadType := r.header.PayloadType & 0x3f
	authenticated := (r.header.PayloadType & 0x40) >> 6
	encrypted := (r.header.PayloadType & 0x80) >> 7

	if authenticated == 1 && encrypted == 1 {
		return r.handleEncryptedRequest(buf, machine)
	}

	switch payloadType {
	case payloadTypeRMCPPlusOpenSessionRequest:
		return r.handleOpenSessionRequest(buf)
	case payloadTypeRAKPMessage1:
		return r.handleRAKPMessage1Request(buf)
	case payloadTypeRAKPMessage3:
		return r.handleRAKPMessage3Request(buf)
	}

	return nil, errors.New("unsupported payload type")
}

func deserializeRMCPPlusSessionHeader(buf io.Reader, authType authenticationType) (*rmcpPlusSessionHeader, error) {
	header := &rmcpPlusSessionHeader{}
	header.AuthenticationType = authType

	if err := binary.Read(buf, binary.LittleEndian, &header.PayloadType); err != nil {
		return nil, err
	}

	if err := binary.Read(buf, binary.LittleEndian, &header.SessionId); err != nil {
		return nil, err
	}

	if err := binary.Read(buf, binary.LittleEndian, &header.SessionSequenceNumber); err != nil {
		return nil, err
	}

	if err := binary.Read(buf, binary.LittleEndian, &header.IpmiPayloadLen); err != nil {
		return nil, err
	}

	return header, nil
}

func (r *rmcpPlus) handleOpenSessionRequest(buf io.Reader) ([]byte, error) {
	payload, err := deserializeOpenSessionRequestPayload(buf)
	if err != nil {
		return nil, err
	}

	// Serialize RMCP+ session header
	obuf := bytes.Buffer{}
	rmcpPlus := &rmcpPlusSessionHeader{
		AuthenticationType:    authTypeRMCPPlus,
		PayloadType:           payloadTypeRMCPPlusOpenSessionResponse,
		SessionId:             0,
		SessionSequenceNumber: 0,
		IpmiPayloadLen:        36,
	}
	if err := binary.Write(&obuf, binary.LittleEndian, rmcpPlus); err != nil {
		return nil, err
	}

	// Serialize Open RMCPPlusSession Response Payload
	session, err := r.session.getNewRMCPPlusSession(payload.RemoteConsoleSessionId)
	if err != nil {
		return nil, err
	}
	response := &openSessionResponsePayload{
		MessageTag:                      payload.MessageTag,
		RmcpPlusStatusCode:              rmcpPlusStatusNoErrors,
		MaximumPrivilegeLevel:           maximumPrivilegeLevelAdministrator,
		Reserved2:                       [1]byte{},
		RemoteConsoleSessionId:          payload.RemoteConsoleSessionId,
		ManagedSystemSessionId:          session.ManagedSystemSessionId,
		AuthenticationPayloadType:       authenticationPayloadTypeAuthenticationAlgorithm,
		Reserved3:                       [2]byte{},
		AuthenticationPayloadLength:     0x08,
		AuthenticationPayloadAlgorithm:  authenticationAlgorithmRKAPHMACSHA1,
		Reserved5:                       [3]byte{},
		IntegrityPayloadType:            integrityPayloadTypeIntegrityAlgorithm,
		Reserved6:                       [2]byte{},
		IntegrityPayloadLength:          0x08,
		IntegrityPayloadAlgorithm:       integrityAlgorithmHMACSHA1196,
		Reserved8:                       [3]byte{},
		ConfidentialityPayloadType:      confidentialityPayloadTypeConfidentialityAlgorithm,
		Reserved9:                       [2]byte{},
		ConfidentialityPayloadLength:    0x08,
		ConfidentialityPayloadAlgorithm: confidentialityAlgorithmAESCBC128,
		Reserved11:                      [3]byte{},
	}
	if err := binary.Write(&obuf, binary.LittleEndian, response); err != nil {
		return nil, err
	}

	return obuf.Bytes(), nil
}

func deserializeOpenSessionRequestPayload(buf io.Reader) (*openSessionRequestPayload, error) {
	payload := &openSessionRequestPayload{}
	if err := binary.Read(buf, binary.LittleEndian, payload); err != nil {
		return nil, err
	}

	return payload, nil
}

func (r *rmcpPlus) handleRAKPMessage1Request(buf io.Reader) ([]byte, error) {
	payload, err := deserializeRAKPMessage1RequestPayload(buf)
	if err != nil {
		return nil, err
	}

	// Check if the session has been activated
	session, ok := r.session.getRMCPPlusSession(payload.ManagedSystemSessionId)
	if !ok {
		return nil, errors.New("session hasn't been activated")
	}

	// Serialize RMCP+ session header
	obuf := bytes.Buffer{}
	rmcpPlus := &rmcpPlusSessionHeader{
		AuthenticationType:    authTypeRMCPPlus,
		PayloadType:           payloadTypeRAKPMessage2,
		SessionId:             0,
		SessionSequenceNumber: 0,
		IpmiPayloadLen:        60,
	}
	if err := binary.Write(&obuf, binary.LittleEndian, rmcpPlus); err != nil {
		return nil, err
	}

	userName := payload.UserName[:payload.UserNameLength]
	user, ok := r.bmcUser.getBMCUser(string(userName))
	if !ok {
		return nil, errors.New("user not found")
	}

	managedSystemRandomNumber, err := generateRandomNumber()
	if err != nil {
		return nil, err
	}
	managedSystemGuid, err := generateRandomNumber()
	if err != nil {
		return nil, err
	}

	session.RemoteConsoleRandomNumber = payload.RemoteConsoleRandomNumber
	session.ManagedSystemRandomNumber = managedSystemRandomNumber
	session.ManagedSystemGuid = managedSystemGuid
	session.RequestedPrivilegeLevel = payload.RequestedMaximumPrivilegeLevelAndNameOnlyLookup
	session.UserNameLength = payload.UserNameLength
	session.UserName = userName

	// Generate Authentication Code with specified Authentication algorithm
	authCode, err := generateAuthCode(session.RemoteConsoleSessionId, session.ManagedSystemSessionId, payload.RemoteConsoleRandomNumber, managedSystemRandomNumber,
		managedSystemGuid, payload.RequestedMaximumPrivilegeLevelAndNameOnlyLookup, payload.UserNameLength, userName, user.Password)
	if err != nil {
		return nil, err
	}

	// Serialize RAKP Message2 Payload
	fixedAuthCode := [20]byte{}
	copy(fixedAuthCode[:], authCode)
	response := &rakpMessage2Response{
		MessageTag:                    payload.MessageTag,
		RmcpPlusStatusCode:            rmcpPlusStatusNoErrors,
		Reserved1:                     [2]byte{},
		RemoteConsoleSessionId:        session.RemoteConsoleSessionId,
		ManagedSystemRandomNumber:     managedSystemRandomNumber,
		ManagedSystemGuid:             managedSystemGuid,
		KeyExchangeAuthenticationCode: fixedAuthCode,
	}
	if err := binary.Write(&obuf, binary.LittleEndian, response); err != nil {
		return nil, err
	}

	return obuf.Bytes(), nil
}

func deserializeRAKPMessage1RequestPayload(buf io.Reader) (*rakpMessage1Request, error) {
	payload := &rakpMessage1Request{}
	if err := binary.Read(buf, binary.LittleEndian, payload); err != nil {
		return nil, err
	}

	return payload, nil
}

func generateAuthCode(remoteConsoleSessionId, managedSystemSessionId uint32, remoteConsoleRandomNumber, managedSystemRandomNumber, managedSystemGuid [16]byte,
	requestedPrivilegeLevel, usernameLength uint8, userName []byte, password string) ([]byte, error) {
	buf := bytes.Buffer{}
	if err := binary.Write(&buf, binary.LittleEndian, remoteConsoleSessionId); err != nil {
		return nil, err
	}
	if err := binary.Write(&buf, binary.LittleEndian, managedSystemSessionId); err != nil {
		return nil, err
	}
	if err := binary.Write(&buf, binary.LittleEndian, remoteConsoleRandomNumber); err != nil {
		return nil, err
	}
	if err := binary.Write(&buf, binary.LittleEndian, managedSystemRandomNumber); err != nil {
		return nil, err
	}
	if err := binary.Write(&buf, binary.LittleEndian, managedSystemGuid); err != nil {
		return nil, err
	}
	if err := binary.Write(&buf, binary.LittleEndian, requestedPrivilegeLevel); err != nil {
		return nil, err
	}
	if err := binary.Write(&buf, binary.LittleEndian, usernameLength); err != nil {
		return nil, err
	}
	if err := binary.Write(&buf, binary.LittleEndian, userName); err != nil {
		return nil, err
	}

	mac := hmac.New(sha1.New, []byte(password))
	mac.Write(buf.Bytes())
	return mac.Sum(nil), nil
}

func (r *rmcpPlus) handleRAKPMessage3Request(buf io.Reader) ([]byte, error) {
	payload, err := deserializeRAKPMessage3RequestPayload(buf)
	if err != nil {
		return nil, err
	}

	session, ok := r.session.getRMCPPlusSession(payload.ManagedSystemSessionId)
	if !ok {
		return nil, errors.New("session hasn't been activated")
	}

	user, ok := r.bmcUser.getBMCUser(string(session.UserName))
	if !ok {
		return nil, errors.New("user not found")
	}

	ok, err = validateAuthCode(payload.KeyExchangeAuthenticationCode, session, user.Password)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errors.New("authentication failed")
	}

	// Serialize RMCP+ session header
	obuf := bytes.Buffer{}
	rmcpPlus := &rmcpPlusSessionHeader{
		AuthenticationType:    authTypeRMCPPlus,
		PayloadType:           payloadTypeRAKPMessage4,
		SessionId:             0,
		SessionSequenceNumber: 0,
		IpmiPayloadLen:        20,
	}
	if err := binary.Write(&obuf, binary.LittleEndian, rmcpPlus); err != nil {
		return nil, err
	}

	sik, err := generateSessionIntegrityKey(session, user.Password)
	if err != nil {
		return nil, err
	}
	session.SessionIntegrityKey = sik
	session.IntegrityKey = generateK1(sik)
	session.ConfidentialityKey = generateK2(sik)

	checkValue, err := generateSessionIntegrityCheckValue(session, sik)
	if err != nil {
		return nil, err
	}

	// Serialize RAKP Message4 Payload
	fixedCheckValue := [12]byte{}
	copy(fixedCheckValue[:], checkValue)
	response := &rakpMessage4Response{
		MessageTag:             payload.MessageTag,
		RmcpPlusStatusCode:     rmcpPlusStatusNoErrors,
		Reserved1:              [2]byte{},
		RemoteConsoleSessionId: session.RemoteConsoleSessionId,
		IntegrityCheckValue:    fixedCheckValue,
	}
	if err := binary.Write(&obuf, binary.LittleEndian, response); err != nil {
		return nil, err
	}

	return obuf.Bytes(), nil
}

func deserializeRAKPMessage3RequestPayload(buf io.Reader) (*rakpMessage3Request, error) {
	payload := &rakpMessage3Request{}
	if err := binary.Read(buf, binary.LittleEndian, payload); err != nil {
		return nil, err
	}

	return payload, nil
}

func validateAuthCode(authCode [20]byte, session *rmcpPlusSession, password string) (bool, error) {
	buf := bytes.Buffer{}
	if err := binary.Write(&buf, binary.LittleEndian, session.ManagedSystemRandomNumber); err != nil {
		return false, err
	}
	if err := binary.Write(&buf, binary.LittleEndian, session.RemoteConsoleSessionId); err != nil {
		return false, err
	}
	if err := binary.Write(&buf, binary.LittleEndian, session.RequestedPrivilegeLevel); err != nil {
		return false, err
	}
	if err := binary.Write(&buf, binary.LittleEndian, session.UserNameLength); err != nil {
		return false, err
	}
	if err := binary.Write(&buf, binary.LittleEndian, session.UserName); err != nil {
		return false, err
	}

	mac := hmac.New(sha1.New, []byte(password))
	mac.Write(buf.Bytes())
	code := mac.Sum(nil)
	return bytes.Equal(authCode[:], code[:]), nil
}

func generateSessionIntegrityKey(session *rmcpPlusSession, password string) ([]byte, error) {
	buf := bytes.Buffer{}
	if err := binary.Write(&buf, binary.LittleEndian, session.RemoteConsoleRandomNumber); err != nil {
		return nil, err
	}
	if err := binary.Write(&buf, binary.LittleEndian, session.ManagedSystemRandomNumber); err != nil {
		return nil, err
	}
	if err := binary.Write(&buf, binary.LittleEndian, session.RequestedPrivilegeLevel); err != nil {
		return nil, err
	}
	if err := binary.Write(&buf, binary.LittleEndian, session.UserNameLength); err != nil {
		return nil, err
	}
	if err := binary.Write(&buf, binary.LittleEndian, session.UserName); err != nil {
		return nil, err
	}

	mac := hmac.New(sha1.New, []byte(password))
	mac.Write(buf.Bytes())
	return mac.Sum(nil), nil
}

func generateK1(sessionIntegrityKey []byte) []byte {
	return generateAdditionalKeyingMaterials(sessionIntegrityKey, 0x01)
}

func generateK2(sessionIntegrityKey []byte) []byte {
	return generateAdditionalKeyingMaterials(sessionIntegrityKey, 0x02)
}

func generateAdditionalKeyingMaterials(sessionIntegrityKey []byte, b byte) []byte {
	mac := hmac.New(sha1.New, sessionIntegrityKey)
	var cons []byte
	for i := 0; i < sha1.Size; i++ {
		cons = append(cons, b)
	}
	mac.Write(cons)
	return mac.Sum(nil)
}

func generateSessionIntegrityCheckValue(session *rmcpPlusSession, sessionIntegrityKey []byte) ([]byte, error) {
	buf := bytes.Buffer{}
	if err := binary.Write(&buf, binary.LittleEndian, session.RemoteConsoleRandomNumber); err != nil {
		return nil, err
	}
	if err := binary.Write(&buf, binary.LittleEndian, session.ManagedSystemSessionId); err != nil {
		return nil, err
	}
	if err := binary.Write(&buf, binary.LittleEndian, session.ManagedSystemGuid); err != nil {
		return nil, err
	}

	mac := hmac.New(sha1.New, sessionIntegrityKey)
	mac.Write(buf.Bytes())
	// HMAC-SHA1-96
	return mac.Sum(nil)[:12], nil
}

func (r *rmcpPlus) handleEncryptedRequest(buf io.Reader, machine Machine) ([]byte, error) {
	session, ok := r.session.getRMCPPlusSession(r.header.SessionId)
	if !ok {
		return nil, errors.New("session hasn't been activated")
	}

	_, ok = r.bmcUser.getBMCUser(string(session.UserName))
	if !ok {
		return nil, errors.New("user not found")
	}

	payload := make([]byte, r.header.IpmiPayloadLen)
	if err := binary.Read(buf, binary.LittleEndian, payload); err != nil {
		return nil, err
	}

	plain, err := decryptByCBCMode(session.ConfidentialityKey, payload)
	if err != nil {
		return nil, err
	}

	ipmi, err := newIPMI(bytes.NewBuffer(plain), len(plain), machine, r.session)
	if err != nil {
		return nil, err
	}
	ipmiResponse, err := ipmi.handle()
	if err != nil {
		return nil, err
	}
	ciphered, err := encryptByCBCMode(session.ConfidentialityKey, ipmiResponse)
	if err != nil {
		return nil, err
	}

	// Serialize RMCP+ session header
	obuf := bytes.Buffer{}
	rmcpPlus := &rmcpPlusSessionHeader{
		AuthenticationType:    authTypeRMCPPlus,
		PayloadType:           0xc0, // payload_type.encrypted(1b) + payload_type.authenticated(1b) + payload_type(6b) (11000000)
		SessionId:             session.RemoteConsoleSessionId,
		SessionSequenceNumber: r.header.SessionSequenceNumber,
		IpmiPayloadLen:        uint16(len(ciphered)),
	}
	if err := binary.Write(&obuf, binary.LittleEndian, rmcpPlus); err != nil {
		return nil, err
	}
	if err := binary.Write(&obuf, binary.LittleEndian, ciphered); err != nil {
		return nil, err
	}

	// IntegrityPad is fixed at size 2 since we only support HMAC-SHA1-96
	trailer := &ipmiSessionTrailer{
		IntegrityPad: [2]byte{0xff, 0xff},
		PadLength:    0x02,
		NextHeader:   0x07,
	}
	if err := binary.Write(&obuf, binary.LittleEndian, trailer); err != nil {
		return nil, err
	}

	ret := obuf.Bytes()
	// Generate Integrity data using fields from RMCP+ header up to and including the field that immediately precedes the AuthCode itself
	authCode, err := generateIntegrityData(ret, session.IntegrityKey)
	if err != nil {
		return nil, err
	}
	ret = append(ret, authCode...)

	return ret, nil
}

func generateIntegrityData(data, integrityKey []byte) ([]byte, error) {
	mac := hmac.New(sha1.New, integrityKey)
	mac.Write(data)
	// HMAC-SHA1-96
	return mac.Sum(nil)[:12], nil
}

func decryptByCBCMode(key []byte, payload []byte) ([]byte, error) {
	if len(payload) < aes.BlockSize {
		return nil, errors.New("payload must be longer that block size")
	} else if len(payload)%aes.BlockSize != 0 {
		return nil, errors.New("payload must be multiple of block size")
	}

	block, err := aes.NewCipher(key[:aes.BlockSize])
	if err != nil {
		return nil, err
	}

	iv := payload[:aes.BlockSize]
	payload = payload[aes.BlockSize:]
	plain := make([]byte, len(payload))

	cbc := cipher.NewCBCDecrypter(block, iv)
	cbc.CryptBlocks(plain, payload)

	padLength := plain[len(plain)-1]
	return plain[:aes.BlockSize-(padLength+1)], nil
}

func encryptByCBCMode(key []byte, plain []byte) ([]byte, error) {
	block, err := aes.NewCipher(key[:aes.BlockSize])
	if err != nil {
		return nil, err
	}

	paddedPlaintext := padPKCS7(plain)
	iv, err := generateRandomNumber()
	if err != nil {
		return nil, err
	}
	cbc := cipher.NewCBCEncrypter(block, iv[:])
	ciphered := make([]byte, aes.BlockSize)
	cbc.CryptBlocks(ciphered, paddedPlaintext)
	return append(iv[:], ciphered...), nil
}

func padPKCS7(data []byte) []byte {
	padSize := aes.BlockSize - (len(data) % aes.BlockSize)
	pad := make([]byte, padSize)
	for i := range pad {
		pad[i] = byte(i + 1)
	}
	pad[padSize-1] = byte(padSize - 1)
	return append(data, pad...)
}

const randomNumberSize = 16

func generateRandomNumber() ([randomNumberSize]byte, error) {
	b := make([]byte, randomNumberSize)
	_, err := rand.Read(b)
	if err != nil {
		return [randomNumberSize]byte{}, err
	}

	encoded := make([]byte, hex.EncodedLen(randomNumberSize))
	hex.Encode(
		encoded, b)

	fixedBytes := [randomNumberSize]byte{}
	copy(fixedBytes[:],
		encoded)
	return fixedBytes, nil
}
