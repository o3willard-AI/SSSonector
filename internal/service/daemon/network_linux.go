package daemon

import (
	"fmt"
	"net"
	"os"

	"github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"
)

// configureBridgeNetwork sets up bridge network
func (m *NamespaceManager) configureBridgeNetwork() error {
	// Create bridge interface
	bridge := &netlink.Bridge{
		LinkAttrs: netlink.LinkAttrs{
			Name: m.config.Network.Name,
		},
	}

	// Add bridge interface
	if err := netlink.LinkAdd(bridge); err != nil && !os.IsExist(err) {
		return fmt.Errorf("failed to create bridge interface: %w", err)
	}

	// Get bridge interface
	l, err := netlink.LinkByName(m.config.Network.Name)
	if err != nil {
		return fmt.Errorf("failed to get bridge interface: %w", err)
	}

	// Set bridge up
	if err := netlink.LinkSetUp(l); err != nil {
		return fmt.Errorf("failed to set bridge up: %w", err)
	}

	// Create veth pair
	vethName := fmt.Sprintf("veth%d", os.Getpid())
	veth := &netlink.Veth{
		LinkAttrs: netlink.LinkAttrs{
			Name:        vethName,
			ParentIndex: l.Attrs().Index,
		},
		PeerName: fmt.Sprintf("eth%d", os.Getpid()),
	}

	// Add veth interface
	if err := netlink.LinkAdd(veth); err != nil {
		return fmt.Errorf("failed to create veth pair: %w", err)
	}

	// Get veth interface
	vethLink, err := netlink.LinkByName(vethName)
	if err != nil {
		return fmt.Errorf("failed to get veth interface: %w", err)
	}

	// Set veth master to bridge
	if err := netlink.LinkSetMaster(vethLink, bridge); err != nil {
		return fmt.Errorf("failed to set veth master: %w", err)
	}

	// Set veth up
	if err := netlink.LinkSetUp(vethLink); err != nil {
		return fmt.Errorf("failed to set veth up: %w", err)
	}

	// Get peer interface
	peerLink, err := netlink.LinkByName(veth.PeerName)
	if err != nil {
		return fmt.Errorf("failed to get peer interface: %w", err)
	}

	// Move peer to network namespace
	if err := netlink.LinkSetNsFd(peerLink, int(unix.Getpid())); err != nil {
		return fmt.Errorf("failed to move peer to namespace: %w", err)
	}

	// Set peer up
	if err := netlink.LinkSetUp(peerLink); err != nil {
		return fmt.Errorf("failed to set peer up: %w", err)
	}

	// Configure IP address if specified
	if m.config.Network.IPAddress != "" {
		addr, err := netlink.ParseAddr(m.config.Network.IPAddress)
		if err != nil {
			return fmt.Errorf("failed to parse IP address: %w", err)
		}

		if err := netlink.AddrAdd(peerLink, addr); err != nil {
			return fmt.Errorf("failed to add IP address: %w", err)
		}
	}

	// Configure DNS
	if err := m.setupDNS(); err != nil {
		return fmt.Errorf("failed to setup DNS: %w", err)
	}

	return nil
}

// setupDNS configures DNS settings
func (m *NamespaceManager) setupDNS() error {
	// Create resolv.conf directory if it doesn't exist
	if err := os.MkdirAll("/etc/netns", 0755); err != nil {
		return fmt.Errorf("failed to create netns directory: %w", err)
	}

	// Create resolv.conf file
	resolvPath := fmt.Sprintf("/etc/netns/%s/resolv.conf", m.config.Network.Name)
	f, err := os.Create(resolvPath)
	if err != nil {
		return fmt.Errorf("failed to create resolv.conf: %w", err)
	}
	defer f.Close()

	// Write nameservers
	for _, dns := range m.config.Network.DNS {
		if _, err := fmt.Fprintf(f, "nameserver %s\n", dns); err != nil {
			return fmt.Errorf("failed to write nameserver: %w", err)
		}
	}

	return nil
}

// setupRouting configures network routing
func (m *NamespaceManager) setupRouting() error {
	// Get default gateway interface
	routes, err := netlink.RouteList(nil, netlink.FAMILY_V4)
	if err != nil {
		return fmt.Errorf("failed to get routes: %w", err)
	}

	var defaultGw *netlink.Route
	for _, route := range routes {
		if route.Dst == nil {
			defaultGw = &route
			break
		}
	}

	if defaultGw == nil {
		return fmt.Errorf("no default gateway found")
	}

	// Add default route
	route := netlink.Route{
		LinkIndex: defaultGw.LinkIndex,
		Gw:        net.ParseIP(m.config.Network.Gateway),
		Dst:       nil,
	}

	if err := netlink.RouteAdd(&route); err != nil {
		return fmt.Errorf("failed to add default route: %w", err)
	}

	return nil
}

// setupFirewall configures network firewall rules
func (m *NamespaceManager) setupFirewall() error {
	// Implementation omitted - would use iptables/nftables
	return nil
}
