package msq

import (
	"gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"
	"log"
	"os"
)

var Producer *kafka.Producer = nil

func init() {
	//https://github.com/edenhill/librdkafka/blob/master/CONFIGURATION.md
	var err error
	Producer, err = kafka.NewProducer(producerConfig)

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