//
//
//                  INTEL CORPORATION PROPRIETARY INFORMATION
//     This software is supplied under the terms of a license agreement or
//     nondisclosure agreement with Intel Corporation and may not be copied
//     or disclosed except in accordance with the terms of that agreement.
//          Copyright(c) 2009-2019 Intel Corporation. All Rights Reserved.
//

package foundationcore

import (
	"fmt"
	"os"
	"sync"

	//"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/cornelk/hashmap"
	zmq "github.com/pebbe/zmq4"
)

/*
// Package scoped globals
var kafkaBrokers string
var kafkaStatsInterval int
var kafkaConsumerPollInterval int
var kafkaDebugEnabled bool
*/
const (
	envZmqBrokerUrlForPublisher = "zmq_broker_url_pub"
	envZmqBrokerUrlForSubscriber = "zmq_broker_url_sub"
)

var (
	zmqBrokerUrlPub string
	zmqBrokerUrlSub string
)

// Message type - define an instance message in our pub-sub world
//
//	topic - category of the info
//	payload - content of the info
type Message struct {
	Topic   string
	Payload string
}

// MessageHandler type - defines the callback function interface
type MessageHandler struct {
	Callbk func(*Message)
}

// Consumer type - High level Message Consumption API
type Consumer struct {
	/*
	sub             *kafka.Consumer
	subMap          map[string]map[*MessageHandler]func(*Message) // key is topic
	lock            sync.RWMutex
	listenerStarted bool
	endSignal       chan bool
	pollIntervals   int
	*/
	sub				*zmq.Socket
	subMap          *hashmap.Map[string, string] // key is topic
	lock            sync.RWMutex
	listenerStarted bool
	endSignal       chan bool
	callbk			func(*Message)
	log             *Logger
}

// NewConsumer - Create an instance of Consumer object
func NewConsumer() *Consumer {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println("Recovering panic in NewConsumer. Error:", err)
		}
	}()
	l, lerr := NewDefaultLogger()
	if lerr != nil {
		fmt.Printf("Error initializing logger : %s", lerr)
		panic(lerr)
	}
	/*
	c, cerr := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers":      kafkaBrokers,
		"statistics.interval.ms": kafkaStatsInterval,
		"group.id":               1})
	if cerr != nil {
		l.Panic("Error creating kafka consumer : %s", cerr)
		panic(cerr)
	}
	*/
	var sub *Consumer
	subSock, _ := zmq.NewSocket(zmq.SUB)
	if e := subSock.Connect(zmqBrokerUrlSub); e == nil {
		m := hashmap.New[string,string]()
		sub = &Consumer{sub: subSock, subMap: m, endSignal: make(chan bool), log: l}	
	} else {
		panic(e)
	}
	return sub
}
/*
// NewConsumerWithGroupID - *LB enabling use case: assigning group id to consumer for 1 to many pub-sub
func NewConsumerWithGroupID(groupID interface{}) *Consumer {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println("Recovering panic in NewConsumerWithGroupID. Error:", err)
		}
	}()
	l, lerr := NewDefaultLogger()
	if lerr != nil {
		fmt.Printf("Error initializing logger : %s", lerr)
		panic(lerr)
	}
	c, cerr := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers":      kafkaBrokers,
		"statistics.interval.ms": kafkaStatsInterval,
		"group.id":               groupID})
	if cerr != nil {
		l.Panic("Error creating kafka consumer : %s", cerr)
		panic(cerr)
	}
	m := make(map[string]map[*MessageHandler]func(*Message))
	sub := &Consumer{sub: c, subMap: m, endSignal: make(chan bool), pollIntervals: kafkaConsumerPollInterval, log: l}
	return sub
}
*/
// Subscribe - Perform topic subscriptions
func (o *Consumer) Subscribe(topic string, callbk func(*Message)) {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println("Recovering panic in Consumer::Subscribe. Error:", err)
		}
	}()
	o.log.Info("Unsubscribing to %s\n", topic)
	/*
	if handler != nil {
		subs, sErr := o.sub.Subscription()
		if sErr != nil {
			panic(sErr)
		}
		var foundTopic = false
		for _, v := range subs {
			if v == topic {
				foundTopic = true
				break
			}
		}
		if !foundTopic {
			subs = append(subs, topic)
			handlerMap := make(map[*MessageHandler]func(*Message))
			handlerMap[handler] = callbk
			//o.lock.Lock()
			o.subMap[topic] = handlerMap
			//o.lock.Unlock()
			if err := o.sub.SubscribeTopics(subs, nil); err != nil {
				panic(err)
			}
			if !o.listenerStarted {
				go o.consumerCoroutine()
				o.listenerStarted = true
			}
		}
	} else {
		panic("Null message handler detected !")
	}
	*/
	o.subMap.Set(topic, topic)
	o.callbk = callbk
}

// Unsubscribe - Perform topic unsubscriptions
func (o *Consumer) Unsubscribe(topic string) {
	o.log.Info("Unsubscribing from %s\n", topic)
	/*
	topics, sErr := o.sub.Subscription()
	succeeded := true
	if sErr == nil {
		idx := -1
		for i, s := range topics {
			if s == topic {
				idx = i
				break
			}
		}
		if idx != -1 {
			topics = append(topics[:idx], topics[idx+1:]...)
			if len(topics) >= 1 {
				if err := o.sub.SubscribeTopics(topics, nil); err != nil {
					o.log.Error("Consumer.SubscribeTopics error : %v\n", err)
					succeeded = false
				}
			} else {
				if err := o.sub.Unsubscribe(); err != nil {
					o.log.Error("Consumer.Unsubscribe error : %v\n", err)
					succeeded = false
				}
			}
		} else {
			o.log.Info("Topic %s has not been subscribed before.\n", topic)
		}
	}
	if succeeded {
		o.log.Info("Unsubscribe from %s succeeded.\n", topic)
	} else {
		o.log.Info("Unsubscribe from %s failed.\n", topic)
	}
	*/
	o.subMap.Del(topic)
}

// consumerCoroutine - Go coroutine that handle kafka consumer poll loops
func (o *Consumer) consumerCoroutine() {
	o.log.Info("Consumer Coroutine started.\n")
	keepRunning := true
	for keepRunning == true {
		select {
		case sig := <-o.endSignal:
			if sig == true {
				keepRunning = false
			}
		default:
			/*
			ev := o.sub.Poll(o.pollIntervals)
			if ev == nil {
				continue
			}
			switch e := ev.(type) {
			case *kafka.Message:
				//o.log.Info("ConsumerCoroutine received message on %s : %s\n", e.TopicPartition, string(e.Value))
				o.handleMessage(*e.TopicPartition.Topic, e.Value)
			case *kafka.Error:
				o.log.Error("ConsumerCoroutine error : %v : %v\n", e.Code(), e)
				if e.Code() == kafka.ErrAllBrokersDown {
					keepRunning = false
				}
			default:
			}
			*/
			topic, _ := o.sub.Recv(0)
			msgStr, err := o.sub.Recv(0)
			if err == nil {
				o.log.Info("Watch zmq listener recv message %s/%s, forwarding to event handler", topic, msgStr)
				o.handleMessage(topic, msgStr)
			} else {
				o.log.Error("Watch zmq listener fail to recv message %v", err)
			}
		}
	}
	o.log.Info("Consumer Coroutine exited.\n")
	o.listenerStarted = false
}

// handleMessage - Helper routine that handles delivery of consumed message to consumers
func (o *Consumer) handleMessage(topic string, payload string) {
	/*
	if !kafkaDebugEnabled {
		defer func() {
			if err := recover(); err != nil {
				o.log.Warn("Error caught in handling callback %s\n", topic)
			}
		}()
	}
	m := o.subMap[topic]
	for _, cb := range m {
		p := string(payload)
		msg := &Message{Topic: topic, Payload: p}
		cb(msg)
	}
	*/
	if _, exists := o.subMap.Get(topic); exists {
		go o.callbk(&Message{
			Topic: topic,
			Payload: payload,
		})
	}
}

// Producer - High level Messaging Publishing API
type Producer struct {
	/*
		pub       *kafka.Producer
		endSignal chan bool
	*/
	pub *zmq.Socket
	log *Logger
}

// NewProducer - Create an instance Producer object
func NewProducer() *Producer {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println("Recovering panic in NewProducer. Error:", err)
		}
	}()
	/*
		p, perr := kafka.NewProducer(&kafka.ConfigMap{
			"bootstrap.servers":      kafkaBrokers,
			"statistics.interval.ms": kafkaStatsInterval})
		if perr != nil {
			panic(perr)
		}
	*/
	l, lerr := NewDefaultLogger()
	if lerr != nil {
		panic(lerr)
	}
	pubSock, _ := zmq.NewSocket(zmq.PUB)
	if e:= pubSock.Connect(zmqBrokerUrlPub); e == nil {
		pub := &Producer{ pub: pubSock, log: l}
		return pub
	} else {
		panic(e)
	}
	/*
	pub := &Producer{pub: p, endSignal: make(chan bool), log: l}
	*/
}

// Publish - publish the message
func (o *Producer) Publish(msg *Message) error {
	/*
	err := o.pub.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &msg.Topic, Partition: kafka.PartitionAny},
		Value:          []byte(msg.Payload),
	}, nil)
	if err != nil {
		o.log.Error("publish error -> %s\n", err.Error())
	}
	return err
	*/
	var e error
	if _, e = o.pub.Send(msg.Topic, zmq.SNDMORE); e == nil {
		_, e = o.pub.Send(msg.Payload, 0)
	}
	return e

}

func (o *Producer) PublishAndWait(msg *Message, timeoutMs int) error {
	/*
	err := o.pub.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &msg.Topic, Partition: kafka.PartitionAny},
		Value:          []byte(msg.Payload),
	}, nil)
	if err != nil {
		o.log.Error("publish error -> %s\n", err.Error())
		return err
	}
	if outstanding := o.pub.Flush(timeoutMs); outstanding > 0 {
		err := fmt.Errorf("publish are having internal issue -> %v outstanding messages", outstanding)
		o.log.Error("publish error -> %s\n", err.Error())
		return err
	}
	return nil
	*/
	var e error
	if _, e = o.pub.Send(msg.Topic, zmq.SNDMORE); e == nil {
		_, e = o.pub.Send(msg.Payload, 0)
	}
	return e	
}

func init() {
	/*
	readObjects := GetSystemConfig().GetConfig(ConfigType_PUBSUB).(map[string]interface{})
	kafkaBrokers = readObjects["metadata_broker_list"].(string)
	fmt.Println("kafkaBrokder =", kafkaBrokers)
	kafkaStatsIntervalStr := readObjects["statistics_interval_ms"].(string)
	kafkaStatsInterval, _ = strconv.Atoi(kafkaStatsIntervalStr)
	kafkaConsumerPollInterval = 100
	if readObjects["debug"].(string) == "true" {
		kafkaDebugEnabled = true
	} else {
		kafkaDebugEnabled = false
	}
	*/
	zmqBrokerUrlPub = os.Getenv(envZmqBrokerUrlForPublisher)
	zmqBrokerUrlSub = os.Getenv(envZmqBrokerUrlForSubscriber)
}
