package btclient

import (
	"github.com/anacrolix/torrent"
	"github.com/anacrolix/torrent/metainfo"
	"github.com/vstorm/bt_crawler/msq"
	"log"
	"time"
)

const maxConcurrentGoroutines = 10 // 最大运行并发的协程数

type BtClient struct {
	client               *torrent.Client
	concurrentGoroutines chan struct{}
	done                 chan bool
}

func NewClient() (bc *BtClient, err error) {
	client, err := torrent.NewClient(nil)
	if err != nil {
		log.Fatalf("create client error: %v", err)
	}
	bc = &BtClient{}
	bc.client = client
	bc.concurrentGoroutines = make(chan struct{}, maxConcurrentGoroutines)
	for i := 0; i < maxConcurrentGoroutines; i++ {
		bc.concurrentGoroutines <- struct{}{}
	}
	bc.done = make(chan bool)
	return
}

// getMetaData 将磁力链转成种子
func (bc *BtClient) getMetaData(infoHash string) *metainfo.Info {
	t, _ := bc.client.AddMagnet("magnet:?xt=urn:btih:" + infoHash)
	select {
	case <-t.GotInfo():
		info := t.Info()
		log.Print(info.Name)
		return info
	case <-time.After(4 * time.Minute): // 4分钟内都获取不了种子信息就放弃获取
		return nil
	}
}

func (bc *BtClient) Start() {
	go func() {
		for {
			<-bc.done
			bc.concurrentGoroutines <- struct{}{}
		}
	}()

	for {
		msg, err := msq.Consumer.ReadMessage(-1)

		if err == nil {
			<-bc.concurrentGoroutines
			infoHash := string(msg.Value)
			go func() {
				info := bc.getMetaData(infoHash)
				if info != nil {
					// 将种子信息保存起来
					
				}
				bc.done <- true
			}()
		} else {
			// The client will automatically try to recover from all errors.
			log.Printf("Consumer error: %v (%v)\n", err, msg)
		}
	}

}
