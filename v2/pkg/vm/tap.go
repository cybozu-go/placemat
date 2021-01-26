package vm

import (
	"fmt"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/placemat/v2/pkg/dcnet"
	"github.com/vishvananda/netlink"
)

type tap struct {
	bridge  netlink.Link
	tapName string
}

type tapInfo struct {
	tap    string
	bridge string
	mtu    int
}

func newTap(bridgeName string) (*tap, error) {
	bridge, err := netlink.LinkByName(bridgeName)
	if err != nil {
		return nil, fmt.Errorf("failed to find the bridge %s: %w", bridgeName, err)
	}

	return &tap{
		bridge: bridge,
	}, nil
}

func (t *tap) create(mtu int) (*tapInfo, error) {
	la := netlink.NewLinkAttrs()
	name, err := dcnet.RandomLinkName(dcnet.LinkTypeTap)
	if err != nil {
		return nil, fmt.Errorf("failed to generate random tap name: %w", err)
	}
	la.Name = name
	tap := &netlink.Tuntap{
		LinkAttrs: la,
		Mode:      netlink.TUNTAP_MODE_TAP,
	}
	if err := netlink.LinkAdd(tap); err != nil {
		return nil, fmt.Errorf("failed to add the tap %s: %w", name, err)
	}
	if mtu > 0 {
		if err := netlink.LinkSetMTU(tap, mtu); err != nil {
			return nil, err
		}
	}
	if err := netlink.LinkSetUp(tap); err != nil {
		return nil, err
	}
	if err = netlink.LinkSetMaster(tap, t.bridge.(*netlink.Bridge)); err != nil {
		return nil, fmt.Errorf("failed to set %s to bridge %s: %w", tap.Name, t.bridge.Attrs().Name, err)
	}
	t.tapName = tap.Name

	createdTap, err := netlink.LinkByName(tap.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to find the created tap: %w", err)
	}
	return &tapInfo{
		tap:    tap.Name,
		bridge: t.bridge.Attrs().Name,
		mtu:    createdTap.Attrs().MTU,
	}, nil
}

func (t *tap) TapInfo() tapInfo {
	return tapInfo{
		tap:    t.tapName,
		bridge: t.bridge.Attrs().Name,
	}
}

func (t *tap) Cleanup() {
	link, err := netlink.LinkByName(t.tapName)
	if err != nil {
		log.Warn("failed to find the tap", map[string]interface{}{
			log.FnError: err,
			"tap":       t.tapName,
		})
		return
	}

	if err := netlink.LinkDel(link); err != nil {
		log.Warn("failed to delete the tap", map[string]interface{}{
			log.FnError: err,
			"tap":       t.tapName,
		})
	}
}
