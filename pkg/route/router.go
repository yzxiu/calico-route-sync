package route

import (
	"net"
	"sync"

	"github.com/vishvananda/netlink"
	"github.com/yzxiu/calico-route-sync/pkg/calico"
	"github.com/yzxiu/calico-route-sync/pkg/types"
	"github.com/yzxiu/calico-route-sync/pkg/util"
)

type Interface interface {
}

type Router struct {
	localNetworks []types.LocalNetwork
	netlinkHandle NetLinkHandle
	mu            sync.Mutex
}

func NewRouter(localNetworks []types.LocalNetwork) (*Router, error) {
	router := &Router{
		localNetworks: localNetworks,
		netlinkHandle: NewNetLinkHandle(),
	}
	return router, nil
}

// UpdateRoute update route
func (r *Router) UpdateRoute(affinity *calico.BlockAffinity, nodeIP net.IP) error {

	_, podNet, err := net.ParseCIDR(affinity.Spec.CIDR)
	if err != nil {
		return err
	}

	route := &types.Route{
		DstNet: podNet,
		GwIP:   nodeIP,
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	return r.netlinkHandle.RouteEnsure(r.localNetworks, route)
}

// DeleteRoute del route
func (r *Router) DeleteRoute(affinity *calico.BlockAffinity) error {

	_, podNet, err := net.ParseCIDR(affinity.Spec.CIDR)
	if err != nil {
		return err
	}

	route := &types.Route{
		DstNet: podNet,
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	return r.netlinkHandle.RouteDel(route)
}

func (r *Router) CheckRouters(pools []net.IPNet, blocks []net.IPNet) {
	r.mu.Lock()
	defer r.mu.Unlock()
	calicoRoutes := r.netlinkHandle.CalicoRoutes(pools)
	for _, route := range calicoRoutes {
		if !contains(blocks, route) {
			ro := &types.Route{
				DstNet: route.Dst,
			}
			_ = r.netlinkHandle.RouteDel(ro)
		}
	}
}

// CleanRoutes del cidr route
func (r *Router) CleanRoutes(pools []net.IPNet) {
	r.mu.Lock()
	defer r.mu.Unlock()
	calicoRoutes := r.netlinkHandle.CalicoRoutes(pools)
	for _, route := range calicoRoutes {
		ro := &types.Route{
			DstNet: route.Dst,
		}
		_ = r.netlinkHandle.RouteDel(ro)
	}
}

func contains(blocks []net.IPNet, route netlink.Route) bool {
	for _, block := range blocks {
		if util.ContainsCIDR(&block, route.Dst) {
			return true
		}
	}
	return false
}
