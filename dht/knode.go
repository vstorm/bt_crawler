package dht

import "net"

type kNode struct {
	nodeId string
	addr *net.UDPAddr
}
