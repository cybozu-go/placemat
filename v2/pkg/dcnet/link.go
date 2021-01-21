package dcnet

import (
	"crypto/rand"
	"fmt"
	"strings"

	"github.com/vishvananda/netlink"
)

type LinkType string

const (
	LinkTypeVeth = LinkType("veth")
	LinkTypeTap  = LinkType("tap")
)

const prefix = "pm_"

func RandomLinkName(typ LinkType) (string, error) {
	entropy := make([]byte, 4)
	_, err := rand.Reader.Read(entropy)
	if err != nil {
		return "", fmt.Errorf("failed to generate random link name: %v", err)
	}

	return fmt.Sprintf("%s%s%x", prefix, typ, entropy), nil
}

func CleanupAllLinks() error {
	links, err := netlink.LinkList()
	if err != nil {
		return fmt.Errorf("failed to list links: %w", err)
	}

	for _, link := range links {
		if strings.HasPrefix(link.Attrs().Name, prefix) {
			if err := netlink.LinkDel(link); err != nil {
				return fmt.Errorf("failed to delete the link %s: %w", link.Attrs().Name, err)
			}
		}
	}

	return nil
}
