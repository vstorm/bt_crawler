package msq

import "gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"

const broker = "0.0.0.0:9093"
const InfoHashTopic = "info-hash"
const consumerGroupId = "info-hash"

var producerConfig = &kafka.ConfigMap{
	"bootstrap.servers": broker,
	"retries":           5,  // 重试次数
	"acks":              -1, // 确认模式，ack=all
	//"partitioner": "consistent_random"  // 默认分区器
}

var consumerConfig = &kafka.ConfigMap{
	"bootstrap.servers": broker,
	// Avoid connecting to IPv6 brokers:
	// This is needed for the ErrAllBrokersDown show-case below
	// when using localhost brokers on OSX, since the OSX resolver
	// will return the IPv6 addresses first.
	// You typically don't need to specify this configuration property.
	"broker.address.family": "v4",
	"group.id":              consumerGroupId,
	"session.timeout.ms":    6000,
	"auto.offset.reset":     "earliest"}
