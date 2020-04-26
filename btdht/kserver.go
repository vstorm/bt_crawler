package btdht

import (
	"github.com/zeebo/bencode"
	"log"
	"net"
	"strconv"
	"time"
)

const udpRecvBufferSize = 65535
const listenUdpPort = "11115"
const chSize = 10000
const sleepTime = time.Millisecond * 50
const reBootstrapTime = time.Second * 10

var bootstrapNodes = []string{
	"router.utorrent.com:6881",
	"router.bittorrent.com:6881",
	"btdht.transmissionbt.com:6881",
	"btdht.aelitis.com:6881",
	"router.silotis.us:6881",
	"btdht.libtorrent.org:25401",
}

type KServer struct {
	udp    net.PacketConn
	nodeId string
	// 使用通道来将 find_node 响应中的节点发送到 SendFindNodeForever，循环继续发起 find_node 请求
	ch chan kNode
}

func NewKServer() (s *KServer, err error) {
	s = &KServer{}
	s.udp, err = net.ListenPacket("udp", ":"+listenUdpPort)
	if err != nil {
		log.Panic(err)
	}
	s.nodeId = randomNodeId()
	s.ch = make(chan kNode, chSize)
	return
}

// 加入 btdht 网络
func (s *KServer) bootstrap() {
	for _, node := range bootstrapNodes {
		//log.Printf("start bootstrap to %v", node)
		host, port, err := net.SplitHostPort(node)
		if err != nil {
			log.Panic(err)
		}
		addrs, err := net.LookupHost(host)
		if err != nil {
			log.Print(err)
		}
		p, _ := strconv.Atoi(port)
		for _, addr := range addrs {
			s.sendFindNode(&net.UDPAddr{
				IP:   net.ParseIP(addr),
				Port: p,
			}, "")
		}
	}
}

// 发送 krpc 消息
func (s *KServer) sendRpc(msg message, addr net.Addr) {
	// msg 需要经过 bencode 编码
	msgBytes, err := bencode.EncodeBytes(msg)
	if err != nil {
		log.Print(err)
	}
	_, err = s.udp.WriteTo(msgBytes, addr)
	if err != nil {
		log.Print(err)
	}
}

// 响应 krpc 错误
func (s *KServer) sendError(tid string, addr net.Addr) {
	e := make([]interface{}, 2)
	e = append(e, 202, "Server Error")
	msg := message{T: tid,
		Y: "e",
		E: e,
	}
	s.sendRpc(msg, addr)
}

// 发送 find_node 请求
func (s *KServer) sendFindNode(addr net.Addr, nId string) {
	//log.Printf("sendFindNode to %v", addr.String())
	var nodeId string

	if nId == "" {
		nodeId = s.nodeId
	} else {
		//刚开始时，使用随机的节点id，能接收到get_peers请求很少
		//使用和被请求的节点target的节点id临近的节点id，这样当target节点收到其他客户端的get_peers请求后，
		//更有可能将我们的节点信息包含在get_peers响应中
		//随后，其他客户端就会发送get_peers请求给我们
		//nodeId = randomNodeId()
		nodeId = getNeighbor(nId)
	}
	targetNodeId := randomNodeId()
	msg := message{
		T: randomTid(),
		Y: "q",
		Q: "find_node",
		A: map[string]string{
			"id":     nodeId,
			"target": targetNodeId,
		},
	}
	s.sendRpc(msg, addr)
}

// 分发接收到请求和响应信息到对应的函数处理
func (s *KServer) onMessage(msg message, addr net.Addr) {
	switch msg.Y {
	case "r": // 响应消息
		// 因为我们只发送了 find_node 请求，所以我们只处理 find_node 响应
		s.onFindNodeResponse(msg)
	case "q": // 请求消息
		switch msg.Q {
		case "ping":
			s.onPingRequest(msg, addr)
		case "find_node":
			s.onFindNodeRequest(msg, addr)
		case "get_peers":
			s.onGetPeersRequest(msg, addr)
		case "announce_peer":
			s.onAnnouncePeerRequest(msg, addr)
		default:

		}
	}
}

// 处理 find_node 响应
func (s *KServer) onFindNodeResponse(msg message) {
	//log.Print("onFindNodeResponse")
	if r, ok := msg.R.(map[string]interface{}); ok {
		if nodes, ok := r["nodes"].(string); ok {
			for _, node := range nodesInfo([]byte(nodes)) {
				s.ch <- node
			}
		}
	}
}

// 处理 ping 请求，虽然 ping 请求里，没有 info hash, 但是及时地去响应 ping 请求，可以不被对方从路由表里淘汰
// ping Query = {"t":"aa", "y":"q", "q":"ping", "a":{"id":"abcdefghij0123456789"}}
// Response = {"t":"aa", "y":"r", "r": {"id":"mnopqrstuvwxyz123456"}}
func (s *KServer) onPingRequest(msg message, addr net.Addr) {
	log.Printf("ping request from %v", addr.String())
	tid := msg.T
	msg = message{
		T: tid,
		R: map[string]string{
			"id": randomNodeId(),
		},
	}
	s.sendRpc(msg, addr)
}

// 处理 find_node 请求，虽然 find_node 请求里，没有 info hash, 但是及时地去响应 find_node 请求，可以不被对方从路由表里淘汰
// find_node Query = {"t":"aa", "y":"q", "q":"find_node", "a": {"id":"abcdefghij0123456789", "target":"mnopqrstuvwxyz123456"}}
// Response = {"t":"aa", "y":"r", "r": {"id":"0123456789abcdefghij", "nodes": "def456..."}}
func (s *KServer) onFindNodeRequest(msg message, addr net.Addr) {
	log.Printf("find_node request from %v", addr.String())
	tid := msg.T
	msg = message{T: tid, R: map[string]string{
		"id":    randomNodeId(),
		"nodes": "",
	}}
	s.sendRpc(msg, addr)
}

// 处理 get_peers 请求, 获取 info hash
// get_peers Query = {"t":"aa", "y":"q", "q":"get_peers", "a": {"id":"abcdefghij0123456789", "info_hash":"mnopqrstuvwxyz123456"}}
// Response with peers = {"t":"aa", "y":"r", "r": {"id":"abcdefghij0123456789", "token":"aoeusnth", "values": ["axje.u", "idhtnm"]}}
// Response with closest nodes = {"t":"aa", "y":"r", "r": {"id":"abcdefghij0123456789", "token":"aoeusnth", "nodes": "def456..."}}
func (s *KServer) onGetPeersRequest(msg message, addr net.Addr) {
	log.Printf("get_peers request from %v", addr.String())
	tid := msg.T
	if a, ok := msg.A.(map[string]interface{}); ok {
		if infoHash, ok := a["info_hash"].(string); ok {
			saveInfoHash([]byte(infoHash))
			msg := message{
				T: tid,
				Y: "r",
				R: map[string]string{
					"id":    randomNodeId(),
					"token": randomToken(),
					"nodes": "",
				}}
			s.sendRpc(msg, addr)
			return
		}
	}
	s.sendError(tid, addr)
}

// 处理 announce_peer 请求, 获取 info hash
// announce_peers Query = {"t":"aa", "y":"q", "q":"announce_peer", "a": {"id":"abcdefghij0123456789", "implied_port": 1, "info_hash":"mnopqrstuvwxyz123456", "port": 6881, "token": "aoeusnth"}}
// Response = {"t":"aa", "y":"r", "r": {"id":"mnopqrstuvwxyz123456"}}
func (s *KServer) onAnnouncePeerRequest(msg message, addr net.Addr) {
	log.Printf("announce_peers request from %v", addr.String())
	tid := msg.T
	if a, ok := msg.A.(map[string]interface{}); ok {
		if infoHash, ok := a["info_hash"].(string); ok {
			saveInfoHash([]byte(infoHash))
			msg := message{
				T: tid,
				Y: "r",
				R: map[string]string{
					"id": randomNodeId(),
				}}
			s.sendRpc(msg, addr)
			return
		}
	}
	s.sendError(tid, addr)
}

// 定时执行 bootstrap
func (s *KServer) ReBootstrap() {
	for {
		time.Sleep(reBootstrapTime)
		s.bootstrap()
	}
}

// 循环发送 find_node 请求，让自己加入到更多节点的路由表中
func (s *KServer) SendFindNodeForever() {
	log.Print("循环发送 find_node 请求")
	for {
		select {
		case node := <-s.ch:
			s.sendFindNode(node.addr, node.nodeId)
			time.Sleep(sleepTime)	// 降低下发送find_node请求的速度，之前发送地太快，CPU占用很高
		default:
			s.bootstrap()
		}
	}
}

// 循环接收其他节点发送过来的请求和响应
func (s *KServer) ReceiveForever() {
	log.Print("循环接收其他节点发送过来的请求和响应")
	s.bootstrap()
	for {
		buf := make([]byte, udpRecvBufferSize)
		n, addr, err := s.udp.ReadFrom(buf)
		//log.Printf("receive response from %v", addr.String())
		if err != nil {
			continue
		}

		msg := message{}
		// 接收到信息使用bencode解码
		err = bencode.DecodeBytes(buf[0:n], &msg)
		if err != nil {
			log.Print(err)
			continue
		}
		// 开启协程处理消息
		go s.onMessage(msg, addr)
	}
}
