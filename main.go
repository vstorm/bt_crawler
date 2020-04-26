package main

import (
	"github.com/vstorm/bt_crawler/btdht"
	"log"
)

func main() {
	server, err := btdht.NewKServer()
	if err != nil {
		log.Panic(err)
	}

	go server.SendFindNodeForever()
	go server.ReBootstrap()
	server.ReceiveForever()
}