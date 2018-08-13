package mtest

import (
	"bytes"
	"errors"
	"io/ioutil"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"golang.org/x/crypto/ssh"
)

const sshTimeout = 3 * time.Minute

var (
	sshClients      = make(map[string]*ssh.Client)
	placematSession *gexec.Session
)

func runPlacemt(args ...string) {
	args = append([]string{placemat}, args...)
	args = append(args, clusterYaml)
	command := exec.Command("sudo", args...)
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	session, err := gexec.Start(command, ginkgo.GinkgoWriter, ginkgo.GinkgoWriter)
	gomega.Expect(err).To(gomega.Succeed())
	placematSession = session
}

func terminatePlacemat() {
	pid := placematSession.Command.Process.Pid
	exec.Command("sudo", "kill", "-TERM", strconv.Itoa(-pid)).Run()
}

func killPlacemat() {
	pid := placematSession.Command.Process.Pid
	exec.Command("sudo", "kill", "-KILL", strconv.Itoa(-pid)).Run()
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
	gomega.ExpectWithOffset(1, err).To(gomega.Succeed())
	return string(stdout)
}
