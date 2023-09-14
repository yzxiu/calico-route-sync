package route

import (
	"errors"
	"fmt"
	"github.com/vishvananda/netlink"
	"github.com/yzxiu/calico-route-sync/pkg/types"
	"github.com/yzxiu/calico-route-sync/pkg/util"
	"k8s.io/klog/v2"
	"net"
)

type netlinkHandle struct {
	netlink.Handle
}

func NewNetLinkHandle() NetLinkHandle {
	return &netlinkHandle{netlink.Handle{}}
}

func (n netlinkHandle) CalicoRoutes(pools []net.IPNet) []netlink.Route {
	routes, err := n.RouteList(nil, netlink.FAMILY_V4)
	var calicoRoutes []netlink.Route
	if err != nil {
		klog.Error("get routes err: %v", err)
		return nil
	}
	for _, r := range routes {
		for _, pool := range pools {
			if r.Dst != nil && util.ContainsCIDR(&pool, r.Dst) {
				calicoRoutes = append(calicoRoutes, r)
			}
		}
	}
	return calicoRoutes
}

func (n netlinkHandle) RouteCheckAndDel(localNetworks []types.LocalNetwork, route *types.Route) error {
	if !n.RouteExist(localNetworks, route) {
		return nil
	}
	return n.RouteDel(route)
}

func (n netlinkHandle) RouteEnsure(localNetworks []types.LocalNetwork, route *types.Route) error {
	var err error
	if n.RouteExist(localNetworks, route) {
		return nil
	}
	if n.RouteConflict(localNetworks, route) {
		err = n.RouteDel(route)
		if err != nil {
			return err
		}
	}
	return n.RouteAdd(localNetworks, route)
}

// RouteAdd TODO Compatible with other networks through tun tunnel
func (n netlinkHandle) RouteAdd(localNetworks []types.LocalNetwork, dr *types.Route) error {
	r := &netlink.Route{
		Dst: dr.DstNet,
		Gw:  dr.GwIP,
	}
	if !gwContains(localNetworks, dr) {
		return errors.New(dr.GwIP.String() + " is not included in the local network")
	}
	var err error
	for _, network := range localNetworks {
		for _, ip4 := range network.LocalIp4 {
			if ip4.Net.Contains(dr.GwIP) {
				err = n.Handle.RouteAdd(r)
				if err != nil {
					klog.Errorf("add route: [%s] err: %v", r.Dst.String(), err)
					continue
				}
				klog.Infof("add route: [%s] success, with interface [%s]", r.Dst.String(), network.LinkName)
				return nil
			}
		}
	}
	return errors.New(fmt.Sprintf("add route: %s err: %v", r.Dst.String(), err))
}

func gwContains(localNetworks []types.LocalNetwork, dr *types.Route) bool {
	for _, network := range localNetworks {
		for _, ip4 := range network.LocalIp4 {
			if ip4.Net.Contains(dr.GwIP) {
				return true
			}
		}
	}
	return false
}

func (n netlinkHandle) RouteDel(route *types.Route) error {
	err := n.Handle.RouteDel(&netlink.Route{
		Dst: route.DstNet,
	})
	if err != nil {
		klog.Error("del route err: %v", err)
	} else {
		klog.Infof("del route: %+v", route.DstNet)
	}
	return err
}

func (n netlinkHandle) RouteDelNet(net *net.IPNet) error {
	routes, err := n.RouteList(nil, netlink.FAMILY_V4)
	if err != nil {
		klog.Error("get routes err: %v", err)
		return err
	}
	for _, r := range routes {
		if r.Dst != nil && net.Contains(r.Dst.IP) {
			err := n.RouteDel(&types.Route{
				DstNet: r.Dst,
				GwIP:   nil,
			})
			if err != nil {
				klog.Errorf("del route: %s err: %v", r.Dst.String(), err)
			}
		}
	}
	return nil
}

func (n netlinkHandle) RouteExist(localNetworks []types.LocalNetwork, dr *types.Route) bool {
	linkName := getViaLinkName(localNetworks, dr)
	if len(linkName) == 0 {
		return false
	}
	routes, err := n.RouteList(nil, netlink.FAMILY_V4)
	if err != nil {
		klog.Error("get routes err: %v", err)
		return false
	}
	return n.routeExist(routes, dr, linkName)
}

func (n netlinkHandle) RouteConflict(localNetworks []types.LocalNetwork, dr *types.Route) bool {
	linkName := getViaLinkName(localNetworks, dr)
	routes, err := n.RouteList(nil, netlink.FAMILY_V4)
	if err != nil {
		klog.Error("get routes err: %v", err)
		return false
	}
	return n.routeConflict(routes, dr, linkName)
}

func (n netlinkHandle) routeConflict(localRoutes []netlink.Route, r *types.Route, name string) bool {
	for _, localRoute := range localRoutes {
		if equalIPNet(localRoute.Dst, r.DstNet) {
			link, err := n.LinkByIndex(localRoute.LinkIndex)
			if err != nil {
				return true
			}
			if !localRoute.Gw.Equal(r.GwIP) || link.Attrs().Name != name {
				return true
			}
		}
	}
	return false
}

func (n netlinkHandle) routeExist(localRoutes []netlink.Route, r *types.Route, name string) bool {
	for _, localRoute := range localRoutes {
		if equalIPNet(localRoute.Dst, r.DstNet) &&
			localRoute.Gw.Equal(r.GwIP) {
			link, err := n.LinkByIndex(localRoute.LinkIndex)
			if err != nil {
				continue
			}
			if link.Attrs().Name == name {
				klog.Infof("route [%s] already exist, with interface [%s]", r.DstNet, name)
				return true
			}
		}
	}
	return false
}
func getViaLinkName(localNetworks []types.LocalNetwork, dr *types.Route) string {
	for _, network := range localNetworks {
		for _, ip4 := range network.LocalIp4 {
			if ip4.Net.Contains(dr.GwIP) {
				return network.LinkName
			}
		}
	}
	return ""
}

func equalIPNet(net1, net2 *net.IPNet) bool {
	if net1 == nil || net2 == nil {
		return false
	}
	if net1.String() == net2.String() {
		return true
	}
	return false
}
