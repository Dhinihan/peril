package pubsub

import (
	"bytes"
	"context"
	"encoding/gob"
	"encoding/json"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

type SimpleQueueType int

const (
	SQTDurable SimpleQueueType = iota
	SQTTransient
)

type Acktype int

const (
	Ack Acktype = iota
	NackRequeue
	NackDiscard
)

func PublishJSON[T any](ch *amqp.Channel, exchange, key string, val T) error {
	jsonData, err := json.Marshal(val)
	if err != nil {
		return err
	}
	return ch.PublishWithContext(
		context.Background(),
		exchange,
		key,
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        jsonData,
		},
	)
}

func DeclareAndBind(
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	queueType SimpleQueueType,
) (*amqp.Channel, amqp.Queue, error) {
	ch, err := conn.Channel()
	if err != nil {
		return nil, amqp.Queue{}, err
	}
	durable := queueType == SQTDurable
	autoDelete := queueType == SQTTransient
	exclusive := queueType == SQTTransient
	tb := amqp.Table{}
	tb["x-dead-letter-exchange"] = "peril_dlx"
	queue, err := ch.QueueDeclare(queueName, durable, autoDelete, exclusive, false, tb)
	if err != nil {
		return nil, amqp.Queue{}, err
	}
	if err := ch.QueueBind(queueName, key, exchange, false, nil); err != nil {
		return nil, amqp.Queue{}, err
	}
	return ch, queue, nil
}

func SubscribeJSON[T any](
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	queueType SimpleQueueType, // an enum to represent "durable" or "transient"
	handler func(T) Acktype,
) error {
	unmarshaller := func(incoming []byte) (T, error) {
		var payload T
		err := json.Unmarshal(incoming, &payload)
		return payload, err
	}
	return subscribe(conn, exchange, queueName, key, queueType, handler, unmarshaller)
}

func SubscribeGob[T any](
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	queueType SimpleQueueType, // an enum to represent "durable" or "transient"
	handler func(T) Acktype,
) error {
	unmarshaller := func(incoming []byte) (T, error) {
		var payload T
		decoder := gob.NewDecoder(bytes.NewReader(incoming))
		err := decoder.Decode(&payload)
		return payload, err
	}
	return subscribe(conn, exchange, queueName, key, queueType, handler, unmarshaller)
}

func subscribe[T any](
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	queueType SimpleQueueType, // an enum to represent "durable" or "transient"
	handler func(T) Acktype,
	unmarshaller func([]byte) (T, error),
) error {
	ch, _, err := DeclareAndBind(conn, exchange, queueName, key, queueType)
	if err != nil {
		return err
	}
	channel, err := ch.Consume(queueName, "", false, false, false, false, nil)
	if err != nil {
		return err
	}
	go func() {
		for msg := range channel {
			payload, err := unmarshaller(msg.Body)
			if err != nil {
				fmt.Printf("Error decoding payload: \n%v\n", err)
			}
			res := handler(payload)
			switch res {
			case Ack:
				fmt.Println("Acknoledged")
				msg.Ack(false)
			case NackRequeue:
				fmt.Println("Not acknoledged, requeued")
				msg.Nack(false, true)
			case NackDiscard:
				fmt.Println("Not acknoledged, discarded")
				msg.Nack(false, false)
			default:
				fmt.Printf("Invalid acktype: %d\n", res)
				msg.Nack(false, false)
			}
		}
	}()

	return nil
}

func PublishGob[T any](ch *amqp.Channel, exchange, key string, val T) error {
	buffer := bytes.NewBuffer([]byte{})
	encoder := gob.NewEncoder(buffer)
	if err := encoder.Encode(val); err != nil {
		return err
	}
	return ch.PublishWithContext(
		context.Background(),
		exchange,
		key,
		false,
		false,
		amqp.Publishing{
			ContentType: "application/gob",
			Body:        buffer.Bytes(),
		},
	)
}
