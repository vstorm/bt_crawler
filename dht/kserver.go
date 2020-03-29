package dht

import (
	"encoding/hex"
	"github.com/zeebo/bencode"
	"log"
	"net"
	"strconv"
)

const udpRecvBufferSize = 65535
const listenUdpPort = "11111"
const chSize = 10000

var bootstrapNodes = []string{
	"router.utorrent.com:6881",
	"router.bittorrent.com:6881",
	"dht.transmissionbt.com:6881",
	"dht.aelitis.com:6881",
	"router.silotis.us:6881",
	"dht.libtorrent.org:25401",
}

type KServer struct {
	udp    net.PacketConn
	nodeId string
	// 使用通道来将 find_node 响应中的节点发送到 SendFindNodeForever，循环继续发起 find_node 请求
	ch chan kNode
}

func NewKServer() (s *KServer, err error) {
	s = &KServer{}
	s.udp, err = net.ListenPacket("udp", ":" + listenUdpPort)
	if err != nil {
		log.Panic(err)
	}
	s.nodeId = randomNodeId()
	s.ch = make(chan kNode, chSize)
	return
}

// 加入 dht 网络
func (s *KServer) bootstrap() {
	log.Print("bootstrap")
	for _, node := range bootstrapNodes {
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
			})
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
	msg := message{t: tid,
		y: "e",
		e: "",	// todo
	}
	s.sendRpc(msg, addr)
}

// 发送 find_node 请求
func (s *KServer) sendFindNode(addr net.Addr) {
	log.Print("sendFindNode")
	var nodeId string
	nodeId = randomNodeId()
	targetNodeId := randomNodeId()
	msg := message{
		t: randomTid(),
		y: "q",
		q: "find_node",
		a: map[string]string{
			"id":     nodeId,
			"target": targetNodeId,
		},
	}
	s.sendRpc(msg, addr)
}

// 分发接收到请求和响应信息到对应的函数处理
func (s *KServer) onMessage(msg message, addr net.Addr) {
	switch msg.y {
	case "r": // 响应消息
		// 因为我们只发送了 find_node 请求，所以我们只处理 find_node 响应
		if r, ok := msg.r.(map[string][]byte); ok {
			if _, ok := r["nodes"]; ok {
				s.onFindNodeResponse(msg)
			}
		}
	case "q": // 请求消息
		switch msg.q {
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
	if r, ok := msg.r.(map[string][]byte); ok {
		if nodes, ok := r["nodes"]; ok {
			for _, node := range nodesInfo(nodes) {
				s.ch <- node
			}
		}
	}
}

// 处理 ping 请求，虽然 ping 请求里，没有 info hash, 但是及时地去响应 ping 请求，可以不被对方从路由表里淘汰
// ping Query = {"t":"aa", "y":"q", "q":"ping", "a":{"id":"abcdefghij0123456789"}}
// Response = {"t":"aa", "y":"r", "r": {"id":"mnopqrstuvwxyz123456"}}
func (s *KServer) onPingRequest(msg message, addr net.Addr) {
	tid := msg.t
	msg = message{
		t:tid,
		r:map[string]string {
			"id": randomNodeId(),
		},
	}
	s.sendRpc(msg, addr)
}

// 处理 find_node 请求，虽然 find_node 请求里，没有 info hash, 但是及时地去响应 find_node 请求，可以不被对方从路由表里淘汰
// find_node Query = {"t":"aa", "y":"q", "q":"find_node", "a": {"id":"abcdefghij0123456789", "target":"mnopqrstuvwxyz123456"}}
// Response = {"t":"aa", "y":"r", "r": {"id":"0123456789abcdefghij", "nodes": "def456..."}}
func (s *KServer) onFindNodeRequest(msg message, addr net.Addr) {
	tid := msg.t
	msg = message{t:tid,r:map[string]string {
		"id": randomNodeId(),
		"nodes": "",
	}}
	s.sendRpc(msg, addr)
}

// 处理 get_peers 请求, 获取 info hash
// get_peers Query = {"t":"aa", "y":"q", "q":"get_peers", "a": {"id":"abcdefghij0123456789", "info_hash":"mnopqrstuvwxyz123456"}}
// Response with peers = {"t":"aa", "y":"r", "r": {"id":"abcdefghij0123456789", "token":"aoeusnth", "values": ["axje.u", "idhtnm"]}}
// Response with closest nodes = {"t":"aa", "y":"r", "r": {"id":"abcdefghij0123456789", "token":"aoeusnth", "nodes": "def456..."}}
func (s *KServer) onGetPeersRequest(msg message, addr net.Addr) {
	tid := msg.t
	if a, ok := msg.a.(map[string][]byte); ok {
		if infoHash, ok := a["info_hash"]; ok {
			s.saveInfoHash(infoHash)
			msg := message{t:tid,
				y: "r",
				r: map[string]string{
					"id": randomNodeId(),
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
	tid := msg.t
	if a, ok := msg.a.(map[string][]byte); ok {
		if infoHash, ok := a["info_hash"]; ok {
			s.saveInfoHash(infoHash)
			msg := message{t:tid,
				y: "r",
				r: map[string]string{
					"id": randomNodeId(),
				}}
			s.sendRpc(msg, addr)
			return
		}
	}
	s.sendError(tid, addr)
}

// 保存 info hash
func (s *KServer) saveInfoHash(infoHash []byte) {
	infoHashHex := hex.EncodeToString(infoHash)
	print(infoHashHex)
}

// 循环发送 find_node 请求，让自己加入到更多节点的路由表中
func (s *KServer) SendFindNodeForever() {
	log.Print("循环发送 find_node 请求")
	select {
	case node := <-s.ch:
		s.sendFindNode(node.addr)
	default:
		s.bootstrap()
	}
}

// 循环接收其他节点发送过来的请求和响应
func (s *KServer) ReceiveForever() {
	log.Print("循环接收其他节点发送过来的请求和响应")
	s.bootstrap()
	for {
		buf := make([]byte, udpRecvBufferSize)
		_, addr, err := s.udp.ReadFrom(buf)
		log.Print(buf)
		if err != nil {
			continue
		}

		msg := message{}
		// 接收到信息使用bencode解码
		err = bencode.DecodeBytes(buf, msg)
		if err != nil {
			continue
		}
		// 开启协程处理消息
		go s.onMessage(msg, addr)
	}
}
