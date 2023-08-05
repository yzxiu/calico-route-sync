package types

import (
	"net"
)

const (
	IPCmd    = "ip"
	IpIpLink = "tunl0"
)

// Route route info
type Route struct {
	DstNet *net.IPNet
	GwIP   net.IP
}

type IP4 struct {
	Net *net.IPNet
	IP  net.IP
}

// LocalNetwork local network
type LocalNetwork struct {
	LinkName string
	LocalIp4 []IP4
	//LocalIP  net.IP
	//LocalNet *net.IPNet
}
