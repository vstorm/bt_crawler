package dht

import (
	"encoding/binary"
	"math/rand"
	"net"
	"time"
)

const tIdLength = 2
const tokenLength = 8
const nodeIdLength = 20
const nodeInfoLength = 26
const nodeIdIpLength = 24

// 生成随机的事务id, 2个字节长
func randomTid() string {
	return randomString(tIdLength)
}

// 生成随机的节点 id，20个字节长
func randomNodeId() string {
	return randomString(nodeIdLength)
}

// 生成随机token, 8个字节长
func randomToken() string {
	return randomString(tokenLength)
}

func randomString(l int) string {
	buf := make([]byte, l)
	rand.Seed(time.Now().UnixNano())
	rand.Read(buf)
	node := string(buf)
	return node
}

// 解析 find_node 响应返回的节点信息
func nodesInfo(compactNodes []byte) (nodes []kNode) {
	l := len(compactNodes)
	if l%nodeInfoLength != 0 {
		return []kNode{}
	}

	i := 0
	for i < l {
		nodeId := string(compactNodes[i : i+nodeIdLength])
		ip := compactNodes[i+nodeIdLength : i+nodeIdIpLength]
		port := int(binary.BigEndian.Uint16(compactNodes[i+nodeIdIpLength : i+nodeInfoLength]))
		node := kNode{nodeId: nodeId, addr: &net.UDPAddr{
			IP:   ip,
			Port: port,
		}}
		nodes = append(nodes, node)
		i += nodeInfoLength
	}
	return nodes

}
