package dcnet

import (
	"fmt"
	"net"

	"github.com/vishvananda/netlink"
)

func detectByRouteGet() (int, error) {
	// 192.0.0.10 is a special globally reachable IPv4 address described in RFC8155.
	// Likewise, 2001:1::2/128 is a special IPv6 address described in the same RFC.
	// https://tools.ietf.org/html/rfc8155
	routes, err := netlink.RouteGet(net.ParseIP("192.0.0.10"))
	if len(routes) == 0 {
		routes, err = netlink.RouteGet(net.ParseIP("2001:1::2/128"))
	}
	if err != nil {
		return 0, err
	}

	mtu := 0
	for _, r := range routes {
		if r.LinkIndex == 0 {
			continue
		}

		link, err := netlink.LinkByIndex(r.LinkIndex)
		if err != nil {
			return 0, err
		}

		lmtu := link.Attrs().MTU
		if lmtu == 0 {
			continue
		}

		if mtu == 0 {
			mtu = lmtu
			continue
		}

		if lmtu < mtu {
			mtu = lmtu
		}
	}

	return mtu, nil
}

func detectFromPhysLinks() (int, error) {
	links, err := netlink.LinkList()
	if err != nil {
		return 0, fmt.Errorf("netlink: failed to list links: %w", err)
	}

	mtu := 0
	for _, link := range links {
		dev, ok := link.(*netlink.Device)
		if !ok {
			continue
		}

		if dev.Attrs().OperState != netlink.OperUp {
			continue
		}

		if dev.MTU == 0 {
			continue
		}

		if mtu == 0 {
			mtu = dev.MTU
			continue
		}

		if dev.MTU < mtu {
			mtu = dev.MTU
		}
	}

	return mtu, nil
}

// DetectMTU returns the right MTU value for communications to the Internet.
// This may return zero if it fails to detect MTU.
func DetectMTU() (int, error) {
	mtu, err := detectByRouteGet()
	if mtu == 0 {
		mtu, err = detectFromPhysLinks()
	}
	return mtu, err
}
