package msq

import (
	"gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"
	"log"
	"os"
)

const broker = "192.168.30.174:9003"
const InfoHashTopic = "info-hash"
var Producer *kafka.Producer = nil

func init() {
	//https://github.com/edenhill/librdkafka/blob/master/CONFIGURATION.md
	Producer, err := kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers": broker,
		"retries": 10,	// 重试次数
		"acks": -1,	// 确认模式，ack=all
		//"partitioner": "consistent_random"  // 默认分区器
	})

	if err != nil {
		log.Printf("Failed to create producer: %s\n", err)
		os.Exit(1)
	}
	log.Printf("%v", Producer)
}

func Send(topic, msg string) {
	deliveryChan := make(chan kafka.Event)
	Producer.Produce(&kafka.Message{
		//TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
		TopicPartition: kafka.TopicPartition{Topic: &topic},
		Value:          []byte(msg),
		Key: 			[]byte(msg),
	}, deliveryChan)

	e := <-deliveryChan
	m := e.(*kafka.Message)

	if m.TopicPartition.Error != nil {
		log.Printf("Delivery failed: %v\n", m.TopicPartition.Error)
	} else {
		log.Printf("Delivered message to topic %s [%d] at offset %v\n",
			*m.TopicPartition.Topic, m.TopicPartition.Partition, m.TopicPartition.Offset)
	}

	close(deliveryChan)
}