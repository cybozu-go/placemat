package mtest

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/cybozu-go/well"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"golang.org/x/crypto/ssh"
)

const (
	sshTimeout         = 3 * time.Minute
	defaultDialTimeout = 30 * time.Second
	defaultKeepAlive   = 5 * time.Second

	// DefaultRunTimeout is the timeout value for Agent.Run().
	DefaultRunTimeout = 10 * time.Minute
)

var (
	sshClients = make(map[string]*sshAgent)
	httpClient = &well.HTTPClient{Client: &http.Client{}}

	agentDialer = &net.Dialer{
		Timeout:   defaultDialTimeout,
		KeepAlive: defaultKeepAlive,
	}
)

type sshAgent struct {
	client *ssh.Client
	conn   net.Conn
}

func sshTo(address string, sshKey ssh.Signer, userName string) (*sshAgent, error) {
	conn, err := agentDialer.Dial("tcp", address+":22")
	if err != nil {
		fmt.Printf("failed to dial: %s\n", address)
		return nil, err
	}
	config := &ssh.ClientConfig{
		User: userName,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(sshKey),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         5 * time.Second,
	}
	err = conn.SetDeadline(time.Now().Add(defaultDialTimeout))
	if err != nil {
		conn.Close()
		return nil, err
	}
	clientConn, channelCh, reqCh, err := ssh.NewClientConn(conn, "tcp", config)
	if err != nil {
		// conn was already closed in ssh.NewClientConn
		return nil, err
	}
	err = conn.SetDeadline(time.Time{})
	if err != nil {
		clientConn.Close()
		return nil, err
	}
	a := sshAgent{
		client: ssh.NewClient(clientConn, channelCh, reqCh),
		conn:   conn,
	}
	return &a, nil
}

func prepareSSHClients(addresses ...string) error {
	sshKey, err := parsePrivateKey(sshKeyFile)
	if err != nil {
		return err
	}

	ch := time.After(sshTimeout)
	for _, a := range addresses {
	RETRY:
		select {
		case <-ch:
			return errors.New("prepareSSHClients timed out")
		default:
		}
		agent, err := sshTo(a, sshKey, "cybozu")
		if err != nil {
			time.Sleep(time.Second)
			goto RETRY
		}
		sshClients[a] = agent
	}

	return nil
}

func parsePrivateKey(keyPath string) (ssh.Signer, error) {
	f, err := os.Open(keyPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	data, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}

	return ssh.ParsePrivateKey(data)
}

func syncSSHKeys() {
	// sync VM root filesystem to store newly generated SSH host keys.
	for h := range sshClients {
		execSafeAt(h, "sync")
	}
}

func destroySSHClients() {
	for key, client := range sshClients {
		client.conn.Close()
		delete(sshClients, key)
	}
}

func execAt(host string, args ...string) (stdout, stderr []byte, e error) {
	return execAtWithStream(host, nil, args...)
}

// WARNING: `input` can contain secret data.  Never output `input` to console.
func execAtWithInput(host string, input []byte, args ...string) (stdout, stderr []byte, e error) {
	var r io.Reader
	if input != nil {
		r = bytes.NewReader(input)
	}
	return execAtWithStream(host, r, args...)
}

// WARNING: `input` can contain secret data.  Never output `input` to console.
func execAtWithStream(host string, input io.Reader, args ...string) (stdout, stderr []byte, e error) {
	agent := sshClients[host]
	return doExec(agent, input, args...)
}

// WARNING: `input` can contain secret data.  Never output `input` to console.
func doExec(agent *sshAgent, input io.Reader, args ...string) ([]byte, []byte, error) {
	err := agent.conn.SetDeadline(time.Now().Add(DefaultRunTimeout))
	if err != nil {
		return nil, nil, err
	}
	defer agent.conn.SetDeadline(time.Time{})

	sess, err := agent.client.NewSession()
	if err != nil {
		return nil, nil, err
	}
	defer sess.Close()

	if input != nil {
		sess.Stdin = input
	}
	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	sess.Stdout = outBuf
	sess.Stderr = errBuf
	err = sess.Run(strings.Join(args, " "))
	return outBuf.Bytes(), errBuf.Bytes(), err
}

func execSafeAt(host string, args ...string) []byte {
	stdout, stderr, err := execAt(host, args...)
	ExpectWithOffset(1, err).To(Succeed(), "[%s] %v: %s", host, args, stderr)
	return stdout
}

func execAtLocal(cmd string, args ...string) ([]byte, error) {
	var stdout bytes.Buffer
	command := exec.Command(cmd, args...)
	command.Stdout = &stdout
	command.Stderr = GinkgoWriter
	err := command.Run()
	if err != nil {
		return nil, err
	}
	return stdout.Bytes(), nil
}

func runPlacemat(cluster string, args ...string) *gexec.Session {
	cleanupPlacemat()

	args = append([]string{"env", "GODEBUG=http2server=0", placematPath}, args...)
	args = append(args, cluster)
	command := exec.Command("sudo", args...)
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
	Expect(err).To(Succeed())
	return session
}

func terminatePlacemat(session *gexec.Session) ([]byte, error) {
	pid := session.Command.Process.Pid
	return execAtLocal("sudo", "kill", "-TERM", strconv.Itoa(-pid))
}

func killPlacemat(session *gexec.Session) ([]byte, error) {
	pid := session.Command.Process.Pid
	return execAtLocal("sudo", "kill", "-KILL", strconv.Itoa(-pid))
}

func cleanupPlacemat() {
	_, err := execAtLocal("sudo", "rm", "-rf", "/var/scratch/placemat/volumes/node1")
	Expect(err).NotTo(HaveOccurred())
	_, err = execAtLocal("sudo", "rm", "-rf", "/var/scratch/placemat/volumes/node2")
	Expect(err).NotTo(HaveOccurred())
}

func pmctl(args ...string) ([]byte, error) {
	return execAtLocal(pmctlPath, args...)
}

func rkt(args ...string) ([]byte, error) {
	args = append([]string{"rkt"}, args...)
	return execAtLocal("sudo", args...)
}
