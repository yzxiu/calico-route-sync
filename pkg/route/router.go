package route

import (
	"github.com/yzxiu/calico-route-sync/pkg/calico"
	"github.com/yzxiu/calico-route-sync/pkg/types"
	"github.com/yzxiu/calico-route-sync/pkg/util"
	"net"
)

type Interface interface {
}

type Router struct {
	localNetworks []types.LocalNetwork
	netlinkHandle NetLinkHandle
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

	return r.netlinkHandle.RouteEnsure(r.localNetworks, route)
}

// DeleteRoute del route
func (r *Router) DeleteRoute(affinity *calico.BlockAffinity, nodeIP net.IP) error {

	_, podNet, err := net.ParseCIDR(affinity.Spec.CIDR)
	if err != nil {
		return err
	}

	route := &types.Route{
		DstNet: podNet,
		GwIP:   nodeIP,
	}

	return r.netlinkHandle.RouteDel(route)
}

// CleanupLeftovers del cidr route
func CleanupLeftovers(ClusterCIDR string) (encounteredError bool) {
	cleanNet := util.ParseNet(ClusterCIDR)
	if cleanNet == nil {
		return true
	}
	err := NewNetLinkHandle().RouteDelNet(cleanNet)
	if err != nil {
		return true
	}
	return false
}
