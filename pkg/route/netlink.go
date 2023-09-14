package route

import (
	"github.com/vishvananda/netlink"
	"github.com/yzxiu/calico-route-sync/pkg/types"
	"net"
)

type NetLinkHandle interface {
	CalicoRoutes(nets []net.IPNet) []netlink.Route
	// RouteExist Determine whether the route exists
	RouteExist(localNetworks []types.LocalNetwork, route *types.Route) bool
	// RouteConflict When dst is the same, but gw or linkName is different, we consider routing conflict
	RouteConflict(localNetworks []types.LocalNetwork, route *types.Route) bool
	// RouteDel delete a single route
	RouteDel(route *types.Route) error
	// RouteDelNet Batch delete the routes contained in the net
	RouteDelNet(net *net.IPNet) error
	// RouteAdd add route
	RouteAdd(localNetworks []types.LocalNetwork, route *types.Route) error
	// RouteEnsure Check whether there is a route, create it if it does not exist, and update it if there is a conflict
	RouteEnsure(localNetworks []types.LocalNetwork, route *types.Route) error
	// RouteCheckAndDel Check if the route exists and delete it
	RouteCheckAndDel(localNetworks []types.LocalNetwork, route *types.Route) error
}
