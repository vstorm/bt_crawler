package btclient

import (
	"github.com/anacrolix/torrent"
	"log"
)

// GetMetaData 将磁力链转成种子
func GetMetaData(infoHash string) {
	c, err := torrent.NewClient(nil)
	defer c.Close()
	if err != nil {
		log.Fatalf("create client error: %v", err)
	}
	t, _ := c.AddMagnet("magnet:?xt=urn:btih:" + infoHash)
	<-t.GotInfo()
	info := t.Info()
	log.Print(info.Name)
}