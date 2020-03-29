package main

import (
	"github.com/vstorm/bt_crawler/dht"
	"log"
)

func main() {
	server, err := dht.NewKServer()
	if err != nil {
		log.Panic(err)
	}

	go server.SendFindNodeForever()
	server.ReceiveForever()

}