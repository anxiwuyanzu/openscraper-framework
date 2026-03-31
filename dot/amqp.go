package dot

import (
	log "github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
	"sync"
	"time"
)

type AmqpClient struct {
	sync.RWMutex
	conn    *amqp.Connection
	channel *amqp.Channel
}

func NewAmqpClient(url string) *AmqpClient {
	client := &AmqpClient{}
	err := client.createAmqpConn(url)
	if err != nil {
		panic(err)
	}

	go func() {
		for {
			reason := <-client.channel.NotifyClose(make(chan *amqp.Error))
			log.Errorf("channel closed, reason: %s", reason.Error())
			// reconnect if not closed by developer
			for {
				err := client.createAmqpConn(url)
				if err == nil {
					break
				}
				// wait 1s for connection reconnect
				time.Sleep(1 * time.Second)
			}
		}

	}()

	return client
}

func (a *AmqpClient) createAmqpConn(url string) error {
	a.Lock()
	defer a.Unlock()
	if a.conn != nil {
		a.conn.Close()
	}

	conn, err := amqp.Dial(url)
	if err != nil {
		log.WithError(err).Error("amqp error")
		return err
	}
	ch, err := conn.Channel()
	if err != nil {
		log.WithError(err).Error("amqp error")
		return err
	}
	a.channel = ch
	a.conn = conn
	return nil
}

func (a *AmqpClient) Channel() *amqp.Channel {
	a.RLock()
	defer a.RUnlock()
	return a.channel
}

func (a *AmqpClient) Push(queue string, body []byte) error {
	return a.channel.Publish("", queue, false, false, amqp.Publishing{ContentType: "text/plain", Body: body})
}

func (a *AmqpClient) Get(queue string) (amqp.Delivery, bool, error) {
	return a.channel.Get(queue, true)
}

func (a *AmqpClient) Purge(queue string) (int, error) {
	return a.channel.QueuePurge(queue, false)
}

//func (a *AmqpClient) NewConsumer(queue string) (*AmqpConsumer, error) {
//	msgs, err := a.channel.Consume(queue, "", true, false, false, false, nil)
//	if err != nil {
//		return nil, err
//	}
//	return &AmqpConsumer{channel: a.channel, msgs: msgs}, nil
//}
//
//type AmqpConsumer struct {
//	channel *amqp.Channel
//	msgs    <-chan amqp.Delivery
//}
