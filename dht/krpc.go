package dht

// krpc消息
type message struct {
	// 事务id，发送方生成事务id，发送请求。响应方收到请求后，会把请求里的事务id带上，发送响应回去。一般使用2个字符
	t string `bencode:"t"`
	// 消息的类型，y的可选值：q，r，e，q表示这是一个请求，r表示这是一个正常的响应，r表示这是一个错误的响应
	y string `bencode:"y"`
	// 请求消息的类型，q的可选值: "ping", "find_node", "get_peer", "announce_peer"
	q string `bencode:"q"`
	// 请求 payload, y="q"
	a interface{} `bencode:"a"`
	// 响应 payload, y="r"
	r interface{} `bencode:"r"`
	// 错误 payload, y="e"
	e interface{} `bencode:"e"`
}

type kRpc struct {
}
