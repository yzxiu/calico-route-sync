package util

import (
	"errors"
	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
	"github.com/yzxiu/calico-route-sync/pkg/types"
	coreapiv1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
	"net"
)

// LocalNetworks Get the network on the machine
func LocalNetworks() ([]types.LocalNetwork, error) {
	// Execute nh.Delete() to release resources,
	// otherwise the number of open files of the program will continue to increase
	nh, err := netlink.NewHandleAt(netns.None())
	defer nh.Delete()
	if err != nil {
		return nil, err
	}
	links, err := nh.LinkList()
	if err != nil {
		return nil, err
	}
	var networks []types.LocalNetwork
	for _, link := range links {
		n, err := net.InterfaceByName(link.Attrs().Name)
		if err != nil {
			continue
		}
		network, err := parseNetwork(n)
		if err != nil {
			klog.Warningf("[%s] can not find ipv4 address", link.Attrs().Name)
			continue
		}
		networks = append(networks, *network)
	}
	if len(networks) == 0 {
		return nil, errors.New("get local network failed")
	}
	return networks, nil
}

func parseNetwork(inter *net.Interface) (*types.LocalNetwork, error) {
	adds, err := inter.Addrs()
	if err != nil {
		return nil, err
	}
	var ips []types.IP4
	for _, add := range adds {
		ip, ipNet, err := net.ParseCIDR(add.String())
		if err != nil {
			continue
		}
		if ip.To4() != nil {
			ips = append(ips, types.IP4{
				IP:  ip,
				Net: ipNet,
			})
		}
	}
	if len(ips) == 0 {
		return nil, errors.New("can not find ipv4 address")
	}
	return &types.LocalNetwork{
		LinkName: inter.Name,
		LocalIp4: ips,
	}, nil
}

// NodeInternalIP get node ip
func NodeInternalIP(node *coreapiv1.Node) net.IP {
	for _, address := range node.Status.Addresses {
		if address.Type == "InternalIP" {
			return net.ParseIP(address.Address)
		}
	}
	return nil
}

// ParseNet parse IPNet
func ParseNet(nets string) *net.IPNet {
	_, network, err := net.ParseCIDR(nets)
	if err != nil {
		return nil
	}
	return network
}

// ContainsCIDR Whether subnet a contains subnet b
// return true b is a subnet of a
// return false b is not a subnet of a
// https://blog.csdn.net/q1009020096/article/details/115763121
func ContainsCIDR(a, b *net.IPNet) bool {
	ones1, _ := a.Mask.Size()
	ones2, _ := b.Mask.Size()
	return ones1 <= ones2 && a.Contains(b.IP)
}
