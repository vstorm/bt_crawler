package main

import (
	"flag"
	"github.com/vstorm/bt_crawler/btclient"
	"github.com/vstorm/bt_crawler/btdht"
	"log"
)

func startBtDht() {
	server, err := btdht.NewKServer()
	if err != nil {
		log.Panic(err)
	}

	go server.SendFindNodeForever()
	go server.ReBootstrap()
	server.ReceiveForever()
}

func startBtClient() {
	client, err := btclient.NewClient()
	if err != nil {
		log.Panic(err)
	}
	client.Start()
}

func main() {
	var m string
	flag.StringVar(&m, "m", "dht", "运行模式")
	flag.Parse()

	switch m {
	case "dht":
		startBtDht()
	case "client":
		startBtClient()
	}
}
