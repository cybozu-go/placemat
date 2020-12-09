package dcnet

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"syscall"

	"github.com/containernetworking/plugins/pkg/ip"
	"github.com/containernetworking/plugins/pkg/ns"
	"github.com/cybozu-go/log"
	"github.com/cybozu-go/placemat/v2/pkg/types"
	"github.com/cybozu-go/well"
	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
)

// NetNS represents a pod resource.
type NetNS struct {
	name          string
	initScripts   []string
	interfaces    []iface
	apps          []app
	hostVethNames []string
}

type app struct {
	name    string
	command []string
}

type iface struct {
	network   netlink.Link
	addresses []*netlink.Addr
}

// NewNetNS creates a NetNS from spec.
func NewNetNS(spec *types.NetNSSpec) (*NetNS, error) {
	n := &NetNS{
		name: spec.Name,
	}

	for _, script := range spec.InitScripts {
		script, err := filepath.Abs(script)
		if err != nil {
			return nil, err
		}
		_, err = os.Stat(script)
		if err != nil {
			return nil, err
		}
		n.initScripts = append(n.initScripts, script)
	}

	for _, i := range spec.Interfaces {
		bridge, err := netlink.LinkByName(i.Network)
		if err != nil {
			return nil, fmt.Errorf("failed to find the bridge %s: %w", i.Network, err)
		}

		var addrs []*netlink.Addr
		for _, a := range i.Addresses {
			addr, err := netlink.ParseAddr(a)
			if err != nil {
				return nil, fmt.Errorf("failed to parse the address %s: %w", a, err)
			}
			addrs = append(addrs, addr)
		}

		n.interfaces = append(n.interfaces, iface{
			network:   bridge,
			addresses: addrs,
		})
	}

	for _, a := range spec.Apps {
		n.apps = append(n.apps, app{
			name:    a.Name,
			command: a.Command,
		})
	}

	return n, nil
}

// Setup creates a linux network namespace and runs applications as specified
func (n *NetNS) Setup(ctx context.Context, mtu int) error {
	createdNS, err := n.createNetNS()
	if err != nil {
		return err
	}
	defer createdNS.Close()

	err = createdNS.Do(func(hostNS ns.NetNS) error {
		// Enable IP Forwarding
		if err := ip.EnableIP4Forward(); err != nil {
			return fmt.Errorf("failed to enable IPv4 forwarding: %w", err)
		}
		if err := ip.EnableIP6Forward(); err != nil {
			return fmt.Errorf("failed to enable IPv6 forwarding: %w", err)
		}

		// Create Veth
		for i, iface := range n.interfaces {
			hostVeth, containerVeth, err := ip.SetupVeth(fmt.Sprintf("eth%d", i), mtu, hostNS)
			if err != nil {
				return fmt.Errorf("failed to set up veth: %w", err)
			}
			n.hostVethNames = append(n.hostVethNames, hostVeth.Name)

			containerVethLink, err := netlink.LinkByName(containerVeth.Name)
			if err != nil {
				return fmt.Errorf("failed to find the container veth %s: %w", containerVeth.Name, err)
			}
			for _, addr := range iface.addresses {
				if err = netlink.AddrAdd(containerVethLink, addr); err != nil {
					return fmt.Errorf("failed to add the address %s to %s: %w", addr.String(), containerVethLink.Attrs().Name, err)
				}
			}
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to create veths in the namespace %s: %w", n.name, err)
	}

	for i, hostVethName := range n.hostVethNames {
		hostVethLink, err := netlink.LinkByName(hostVethName)
		if err != nil {
			return fmt.Errorf("failed to find the host veth %s: %w", hostVethName, err)
		}
		bridge := n.interfaces[i].network
		if err = netlink.LinkSetMaster(hostVethLink, bridge); err != nil {
			return fmt.Errorf("failed to set %s to bridge %s: %w", hostVethLink.Attrs().Name, bridge.Attrs().Name, err)
		}
	}

	err = createdNS.Do(func(hostNS ns.NetNS) error {
		// Run InitScripts
		for _, script := range n.initScripts {
			if err := well.CommandContext(ctx, script).Run(); err != nil {
				return fmt.Errorf("failed to run the script %s: %w", script, err)
			}
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to run init scripts in namespace %s: %w", n.name, err)
	}

	// Run Commands
	env := well.NewEnvironment(ctx)
	for _, app := range n.apps {
		env.Go(func(ctx2 context.Context) error {
			err := createdNS.Do(func(hostNS ns.NetNS) error {
				if err := well.CommandContext(ctx, app.command[0], app.command[1:]...).Run(); err != nil {
					return fmt.Errorf("failed to execute the command %v: %w", app.command, err)
				}
				return nil
			})
			if err != nil {
				return fmt.Errorf("failed to run command inside namespace %s: %w", n.name, err)
			}
			return nil
		})
	}
	env.Stop()
	return env.Wait()
}

func (n *NetNS) createNetNS() (ns.NetNS, error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	currentNs, err := netns.Get()
	if err != nil {
		return nil, fmt.Errorf("failed to get the current NetNS: %w", err)
	}

	nsHandle, err := netns.NewNamed(n.name)
	if err != nil {
		return nil, fmt.Errorf("failed to create network namespace %s: %w", n.name, err)
	}
	defer nsHandle.Close()

	if err = netns.Set(currentNs); err != nil {
		return nil, fmt.Errorf("failed to set the original NetNS: %w", err)
	}

	createdNS, err := ns.GetNS(path.Join(GetNsRunDir(), n.name))
	if err != nil {
		return nil, fmt.Errorf("failed to get network namespace %s: %w", n.name, err)
	}

	return createdNS, err
}

// Reference https://github.com/containernetworking/plugins/blob/509d645ee9ccfee0ad90fe29de3133d0598b7305/pkg/testutils/netns_linux.go#L31-L47
func GetNsRunDir() string {
	xdgRuntimeDir := os.Getenv("XDG_RUNTIME_DIR")

	/// If XDG_RUNTIME_DIR is set, check if the current user owns /var/run.  If
	// the owner is different, we are most likely running in a user namespace.
	// In that case use $XDG_RUNTIME_DIR/netns as runtime dir.
	if xdgRuntimeDir != "" {
		if s, err := os.Stat("/var/run"); err == nil {
			st, ok := s.Sys().(*syscall.Stat_t)
			if ok && int(st.Uid) != os.Geteuid() {
				return path.Join(xdgRuntimeDir, "netns")
			}
		}
	}

	return "/var/run/netns"
}

// Cleanup
func (n *NetNS) Cleanup() {
	if err := netns.DeleteNamed(n.name); err != nil {
		log.Warn("failed to delete the network namespace", map[string]interface{}{
			log.FnError: err,
			"netns":     n.name,
		})
	}

	for _, hostVethName := range n.hostVethNames {
		hostVeth, err := netlink.LinkByName(hostVethName)
		if err != nil {
			log.Warn("failed to find the veth", map[string]interface{}{
				log.FnError: err,
				"veth":      hostVeth,
			})
		}
		if err := netlink.LinkDel(hostVeth); err != nil {
			log.Warn("failed to delete the veth", map[string]interface{}{
				log.FnError: err,
				"veth":      hostVeth,
			})
		}
	}
}
