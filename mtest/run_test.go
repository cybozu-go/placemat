package mtest

import (
	"bytes"
	"errors"
	"io/ioutil"
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

const sshTimeout = 2 * time.Minute

var (
	sshClients = make(map[string]*ssh.Client)
	httpClient = &well.HTTPClient{Client: &http.Client{}}
)

func runPlacemt(cluster string, args ...string) *gexec.Session {
	cleanupPlacemat()

	args = append([]string{placemat}, args...)
	args = append(args, cluster)
	command := exec.Command("sudo", args...)
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	session, err := gexec.Start(command, nil, nil)
	Expect(err).To(Succeed())
	return session
}

func terminatePlacemat(session *gexec.Session) {
	pid := session.Command.Process.Pid
	exec.Command("sudo", "kill", "-TERM", strconv.Itoa(-pid)).Run()
}

func killPlacemat(session *gexec.Session) {
	pid := session.Command.Process.Pid
	exec.Command("sudo", "kill", "-KILL", strconv.Itoa(-pid)).Run()
}

func cleanupPlacemat() {
	exec.Command("sudo", "rm", "-rf", "/var/scratch/placemat/volumes/node1").Run()
	exec.Command("sudo", "rm", "-rf", "/var/scratch/placemat/volumes/node2").Run()
}

func sshTo(address string, sshKey ssh.Signer) (*ssh.Client, error) {
	config := &ssh.ClientConfig{
		User: "cybozu",
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(sshKey),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         5 * time.Second,
	}
	return ssh.Dial("tcp", address+":22", config)
}

func parsePrivateKey() (ssh.Signer, error) {
	f, err := os.Open(os.Getenv("SSH_PRIVKEY"))
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

func prepareSSHClients(addresses ...string) error {
	sshKey, err := parsePrivateKey()
	if err != nil {
		return err
	}

	ch := time.After(sshTimeout)
	for _, a := range addresses {
	RETRY:
		select {
		case <-ch:
			return errors.New("timed out")
		default:
		}
		client, err := sshTo(a, sshKey)
		if err != nil {
			time.Sleep(5 * time.Second)
			goto RETRY
		}
		sshClients[a] = client
	}

	return nil
}

func syncSSHKeys() {
	// sync VM root filesystem to store newly generated SSH host keys.
	for h := range sshClients {
		execSafeAt(h, "sync")
	}
}

func destroySSHClients() {
	for key, client := range sshClients {
		client.Close()
		delete(sshClients, key)
	}
}

func execAt(host string, args ...string) (stdout, stderr []byte, e error) {
	client := sshClients[host]
	sess, err := client.NewSession()
	if err != nil {
		return nil, nil, err
	}
	defer sess.Close()

	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	sess.Stdout = outBuf
	sess.Stderr = errBuf
	err = sess.Run(strings.Join(args, " "))
	return outBuf.Bytes(), errBuf.Bytes(), err
}

func execSafeAt(host string, args ...string) string {
	stdout, _, err := execAt(host, args...)
	ExpectWithOffset(1, err).To(Succeed())
	return string(stdout)
}

func pmctl(args ...string) ([]byte, error) {
	var stdout bytes.Buffer
	command := exec.Command(pmctlPath, args...)
	command.Stdout = &stdout
	command.Stderr = GinkgoWriter
	err := command.Run()
	if err != nil {
		return nil, err
	}
	return stdout.Bytes(), nil
}

func rkt(args ...string) ([]byte, error) {
	var stdout bytes.Buffer
	command := exec.Command("rkt", args...)
	command.Stdout = &stdout
	command.Stderr = GinkgoWriter
	err := command.Run()
	if err != nil {
		return nil, err
	}
	return stdout.Bytes(), nil
}
