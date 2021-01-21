package mtest

//import (
//	"context"
//	"fmt"
//	"io/ioutil"
//	"net"
//	"os"
//	"path/filepath"
//	"strings"
//	"time"
//
//	"github.com/containernetworking/plugins/pkg/ns"
//	"github.com/cybozu-go/placemat/v2/pkg/dcnet"
//	"github.com/cybozu-go/placemat/v2/pkg/types"
//	"github.com/cybozu-go/placemat/v2/pkg/vm"
//	. "github.com/onsi/ginkgo"
//	. "github.com/onsi/gomega"
//	"golang.org/x/crypto/ssh"
//)
//
//// TODO Run on GCP
//var _ = Describe("Node", func() {
//	BeforeEach(func() {
//		Expect(dcnet.CreateNatRules()).ToNot(HaveOccurred())
//	})
//
//	AfterEach(func() {
//		dcnet.CleanupNatRules()
//	})
//
//	It("should setup a node as a QEMU process", func() {
//		// Set up runtime
//		vm.LoadModules()
//		cur, err := os.Getwd()
//		Expect(err).NotTo(HaveOccurred())
//		temp := filepath.Join(cur, "temp")
//		Expect(os.Mkdir(temp, 0755)).NotTo(HaveOccurred())
//		defer os.RemoveAll(temp)
//		r, err := vm.NewRuntime(false, false, filepath.Join(temp, "run"), filepath.Join(temp, "data"),
//			filepath.Join(temp, "cache"), "127.0.0.1:10808")
//		Expect(err).NotTo(HaveOccurred())
//
//		// Prepare config files and directories
//		Expect(os.Mkdir("temp/run", 0755)).NotTo(HaveOccurred())
//		Expect(os.Mkdir("temp/shared-dir", 0755)).NotTo(HaveOccurred())
//		userData := `
//#cloud-config
//hostname: boot-0
//users:
// - name: cybozu
//   sudo: ALL=(ALL) NOPASSWD:ALL
//   primary-group: cybozu
//   groups: users, admin, systemd-journal
//   lock_passwd: false
//   # below passwd is hashed string of "cybozu"
//   passwd: $6$rounds=4096$m3AVOWeB$EPystoHozf.eJNCm4tWyRHpJzgTDymYuGOONWxRN8uk4amLvxwB4Pc7.tEkZdeXewoVEBEX5ujUon9wSpEf1N.
//   shell: /bin/bash
//   ssh_authorized_keys:
//   - ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQCoYNNrwXDSpa5D/vG+xN0V8/SiqCldTGXwWk4VaklZNQz1mEk2J0F+CVucABDXj/sl+9NQcBCBDtfSKHwgnZnpUMYZn2SvU3jaI3n/XvIwJnCAaBFvC2+P79fiUVRrTNUd792cvGQFDJXaE6+Us78Tt9R5XLvQy3/U12Vm0jXmXUlf/6kklVJb5hovtAXhfhphp349JBTmNFAHkox+FNJrK4AwMlz8UJhwCuqEe8L96HqVvK5DLdaiQjWn5dpFvWCLJt8VbfnKZ9VPcSwYFmOSmyBkYIx+dDkf7Gv0mIi28sTvIB2cFl6/HkPIqasL3m2+MqIMZJQt3yPgiIC+WwAv mtest
//`
//		Expect(ioutil.WriteFile("temp/user-data.yml", []byte(userData), 0600)).NotTo(HaveOccurred())
//		networkConfig := `
//version: 2
//ethernets:
//  ens4:
//    addresses:
//      - 10.0.0.1/24
//`
//		Expect(ioutil.WriteFile("temp/network.yml", []byte(networkConfig), 0600)).NotTo(HaveOccurred())
//		sshkey := `
//-----BEGIN RSA PRIVATE KEY-----
//MIIEowIBAAKCAQEAqGDTa8Fw0qWuQ/7xvsTdFfP0oqgpXUxl8FpOFWpJWTUM9ZhJ
//NidBfglbnAAQ14/7JfvTUHAQgQ7X0ih8IJ2Z6VDGGZ9kr1N42iN5/17yMCZwgGgR
//bwtvj+/X4lFUa0zVHe/dnLxkBQyV2hOvlLO/E7fUeVy70Mt/1NdlZtI15l1JX/+p
//JJVSW+YaL7QF4X4aYad+PSQU5jRQB5KMfhTSayuAMDJc/FCYcArqhHvC/eh6lbyu
//Qy3WokI1p+XaRb1giybfFW35ymfVT3EsGBZjkpsgZGCMfnQ5H+xr9JiItvLE7yAd
//nBZevx5DyKmrC95tvjKiDGSULd8j4IiAvlsALwIDAQABAoIBACQJJPZo3gaXIua2
//h3J2m4J5RaASMVggY6i/CvsWVkBbVDyzrOeEG0YoJo0KjpAz5mJItP8AHOgiDxqR
//Q4+Pa0M94EfXjyreyHyXHyMCZP7dGzLAEwsa/XNmt2NeWJzmQq43icxjnVxfRyr3
//D5rZpUlJDJY0vJWBGAirWK5ayuJUN9SFfsJWqEk4CDNQvONWNK1gvxazbppdCu93
//FuuQvNkutosx8tmyl9eCev6sIugB6pp/YRf57JLRKJ0BwG7qn3gRNpyQOhGrF1MX
//+0I9Ldi42OluLKP1X7n6MOux7Alxh5KuIq28d4mrE0iKUGU3yBt9R61UUGgynWc/
//98QUQ/ECgYEA11Oj2fizzNnvEWn8nO1apYohtG+fjga8Tt472EpDjwwvhFVAX59f
//2VoTJZct/oCkgffeut+bTB9FIYMRPoO1OH7Vd5lqsa+GCO+vTDM2mezFdfItxPoe
//8h8u4brBy+x0aPyiNLEuYIjUh0ymUoviFGB4jP/J2QNzJvhM1nu12BsCgYEAyC7w
//nHiMmkfPEslG1DyKsD2rmPiVHLldjVzYSOgBcL8bPGU2SYQedRdQBpzK6OF9TqXv
//QsvO6HVgq8bmZVr2e0zhZhCak+NyxczObOdP2i+M2QUIXGBXG7ivCBexSiUH0DUd
//xV2LEWkXA+3WuJ9gKY9GBBBdTOD+jqssiLZvIX0CgYEAtlHgo9g8TZCeJy2Jskoa
///Z2nCkOVYsl7OoBbRbkj2QRlW3RfzFeC7eOh4KtQS3UbVdzN34cj1GGJxGVY/YjB
//sfNaxijFuWu4XuqrkCaw7cYYL9T+QhHSkAotRP4/x24P5zE6GsmHTj+tTF5vWeeN
//ZtmEWUbf3vtXzkBhtx4Ki88CgYAaliFepqQF2YOm+xRtG51PyuD/cARdzECghbQz
//+pw2XStA2jBbkzB4XKBEQI6yX0BFMcSVGnxgYzZzmfb/fxU9SviklY/yFEMqAglo
//bVAtqiMKr6BspF7tT5nveTYSothmzqclj0bpCQwFeZEK9B/RZTXnVEUP8NHeIN3J
//SnF4AQKBgCXupLs3AqbEWg2iUs+Eqeru0rEWopuTUiLJOvoT6X5NQlUIlpv5Ye+Z
//tsChz55NjCxNEpn4NvGyeGgJrBEGwAPbx/X2v2BWFxWPNWh6byHi9ZxELa0Utlc8
//B29lX8k9dqD0HitCL6ibsw0DqsU6FC3fd179rH8Bik83FuukuxvD
//-----END RSA PRIVATE KEY-----
//`
//		Expect(ioutil.WriteFile("temp/sshkey", []byte(sshkey), 0600)).NotTo(HaveOccurred())
//
//		clusterYaml := `
//kind: Node
//name: boot-0
//cpu: 8
//memory: 2G
//interfaces:
//- op-to-boot-0
//volumes:
//- kind: image
//  name: root
//  image: custom-ubuntu-image
//  cache: writeback
//  copy-on-write: true
//- kind: localds
//  name: seed
//  network-config: temp/network.yml
//  user-data: temp/user-data.yml
//- kind: hostPath
//  name: sabakan
//  path: temp/shared-dir
//smbios:
// serial: fb8f2417d0b4db30050719c31ce02a2e8141bbd8
//---
//kind: Image
//name: custom-ubuntu-image
//file: assets/cybozu-ubuntu-18.04-server-cloudimg-amd64.img
//---
//kind: Network
//name: op-to-boot-0
//type: internal
//---
//kind: NetworkNamespace
//name: operation
//interfaces:
//- addresses:
//  - 10.0.0.2/24
//  network: op-to-boot-0
//`
//		cluster, err := types.Parse(strings.NewReader(clusterYaml))
//		Expect(err).NotTo(HaveOccurred())
//
//		// Create bridge networks
//		var networks []*dcnet.Network
//		for _, network := range cluster.Networks {
//			network, err := dcnet.NewNetwork(network)
//			Expect(err).NotTo(HaveOccurred())
//			Expect(network.Create(1460)).NotTo(HaveOccurred())
//			networks = append(networks, network)
//		}
//		defer func() {
//			for _, network := range networks {
//				network.Cleanup()
//			}
//		}()
//
//		// Create a network namespace
//		var netnss []*dcnet.NetNS
//		ctx, cancel := context.WithCancel(context.Background())
//		for _, netnsSpec := range cluster.NetNSs {
//			netns, err := dcnet.NewNetNS(netnsSpec)
//			Expect(err).NotTo(HaveOccurred())
//			Expect(netns.Setup(ctx, 1460)).NotTo(HaveOccurred())
//			netnss = append(netnss, netns)
//		}
//		defer func() {
//			for _, netns := range netnss {
//				netns.Cleanup()
//			}
//		}()
//
//		// Set up a node
//		node, err := vm.newNode(cluster.Nodes[0], cluster.Images)
//		Expect(err).NotTo(HaveOccurred())
//		nodeVm, err := node.Setup(ctx, r, 1460)
//		Expect(err).NotTo(HaveOccurred())
//		defer node.Cleanup()
//
//		opNs, err := ns.GetNS(filepath.Join(dcnet.GetNsRunDir(), "operation"))
//		Expect(err).NotTo(HaveOccurred())
//		sshKey, err := parsePrivateKey("temp/sshkey")
//		Expect(err).NotTo(HaveOccurred())
//		Eventually(func() error {
//			err := opNs.Do(func(hostNS ns.NetNS) error {
//				_, err := sshTo("10.0.0.1", sshKey, "cybozu")
//				if err != nil {
//					return fmt.Errorf("failed ssh to the node: %w", err)
//				}
//				return nil
//			})
//			if err != nil {
//				return fmt.Errorf("failed to exec ssh inside the netns operation: %w", err)
//			}
//
//			return nil
//		}).Should(Succeed())
//
//		cancel()
//		defer nodeVm.Cleanup()
//	})
//})
//
//const (
//	defaultDialTimeout = 30 * time.Second
//	defaultKeepAlive   = 5 * time.Second
//)
//
//var agentDialer = &net.Dialer{
//	Timeout:   defaultDialTimeout,
//	KeepAlive: defaultKeepAlive,
//}
//
//type sshAgent struct {
//	client *ssh.Client
//	conn   net.Conn
//}
//
//func sshTo(address string, sshKey ssh.Signer, userName string) (*sshAgent, error) {
//	conn, err := agentDialer.Dial("tcp", address+":22")
//	if err != nil {
//		fmt.Printf("failed to dial: %s\n", address)
//		return nil, err
//	}
//	config := &ssh.ClientConfig{
//		User: userName,
//		Auth: []ssh.AuthMethod{
//			ssh.PublicKeys(sshKey),
//		},
//		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
//		Timeout:         5 * time.Second,
//	}
//	err = conn.SetDeadline(time.Now().Add(defaultDialTimeout))
//	if err != nil {
//		conn.Close()
//		return nil, err
//	}
//	clientConn, channelCh, reqCh, err := ssh.NewClientConn(conn, "tcp", config)
//	if err != nil {
//		// conn was already closed in ssh.NewClientConn
//		return nil, err
//	}
//	err = conn.SetDeadline(time.Time{})
//	if err != nil {
//		clientConn.Close()
//		return nil, err
//	}
//	a := sshAgent{
//		client: ssh.NewClient(clientConn, channelCh, reqCh),
//		conn:   conn,
//	}
//	return &a, nil
//}
//
//func parsePrivateKey(keyPath string) (ssh.Signer, error) {
//	f, err := os.Open(keyPath)
//	if err != nil {
//		return nil, err
//	}
//	defer f.Close()
//
//	data, err := ioutil.ReadAll(f)
//	if err != nil {
//		return nil, err
//	}
//
//	return ssh.ParsePrivateKey(data)
//}
