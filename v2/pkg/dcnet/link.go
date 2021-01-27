package dcnet

import (
	"crypto/rand"
	"fmt"
	"strings"

	"github.com/cybozu-go/log"
	"github.com/vishvananda/netlink"
)

type LinkType string

const (
	LinkTypeVeth = LinkType("veth")
	LinkTypeTap  = LinkType("tap")
)

const prefix = "pm_"

// RandomLinkName generates a random link name
func RandomLinkName(typ LinkType) (string, error) {
	entropy := make([]byte, 4)
	_, err := rand.Reader.Read(entropy)
	if err != nil {
		return "", fmt.Errorf("failed to generate random link name: %v", err)
	}

	return fmt.Sprintf("%s%s%x", prefix, typ, entropy), nil
}

// CleanupAllLinks removes all links placemat added
func CleanupAllLinks() {
	links, err := netlink.LinkList()
	if err != nil {
		log.Warn("failed to list links", map[string]interface{}{
			log.FnError: err,
		})
		return
	}

	for _, link := range links {
		if strings.HasPrefix(link.Attrs().Name, prefix) {
			if err := netlink.LinkDel(link); err != nil {
				log.Warn("failed to delete the link", map[string]interface{}{
					log.FnError: err,
					"name":      link.Attrs().Name,
				})
			}
		}
	}
}
