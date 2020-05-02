package msq

import (
	"fmt"
	"gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"
	"log"
	"os"
)

var Consumer *kafka.Consumer = nil


func init() {
	var err error
	Consumer, err = kafka.NewConsumer(consumerConfig)

	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create consumer: %s\n", err)
		os.Exit(1)
	}
	log.Printf("%v", Consumer)
	err = Consumer.Subscribe(InfoHashTopic, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to subscribe topic: %s\n", err)
		os.Exit(1)
	}
}
