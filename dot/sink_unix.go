//go:build !windows

package dot

import (
	"github.com/anxiwuyanzu/openscraper-framework/v4/util/serde"
	"time"

	"github.com/anxiwuyanzu/openscraper-framework/v4/util"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	_ "github.com/confluentinc/confluent-kafka-go/v2/kafka/librdkafka_vendor"
	log "github.com/sirupsen/logrus"
)

// NewSinker 通过配置和自定义的 kafkaServers 生成 ISinker; 方便切换kafka
func NewSinker(kafkaServers string) ISinker {
	if Conf().Sinker == "debug" || Conf().Sinker == "dummy" {
		return NewDebugSinker(Conf().Sinker)
	} else if Conf().Sinker == "kafka" {
		return NewKafkaSinker(kafkaServers, nil)
	}
	return nil
}

type KafkaSinker struct {
	kafkaProducer *kafka.Producer
	stopCh        chan bool
	quitCh        chan bool
}

func NewKafkaSinker(servers string, cfg *KafkaConfig) *KafkaSinker {
	if cfg == nil {
		cfg = &Conf().Kafka
	}
	if u, ok := Conf().Kafka.Others[servers]; ok {
		servers = u
	}

	stopCh := make(chan bool, 1)
	quitCh := make(chan bool, 1)

	k := newKafkaProducer(stopCh, quitCh, cfg, servers)
	s := &KafkaSinker{kafkaProducer: k, stopCh: stopCh, quitCh: quitCh}
	EnsureClose(s.Close)
	return s
}

func (s *KafkaSinker) Sink(topic string, data interface{}, key ...string) {
	buf, _ := serde.Json.Marshal(data)
	var msgKey []byte
	if len(key) == 1 {
		msgKey = []byte(key[0])
	}
	err := s.kafkaProducer.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
		Value:          buf,
		Key:            msgKey,
	}, nil)

	if err != nil {
		log.WithField("err", err.Error()).WithField("topic", topic).Error("write kafka error")
	}
}

func (s *KafkaSinker) SinkString(topic string, data string, key ...string) {
	var msgKey []byte
	if len(key) == 1 {
		msgKey = []byte(key[0])
	}
	err := s.kafkaProducer.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
		Value:          []byte(data),
		Key:            msgKey,
	}, nil)

	if err != nil {
		log.WithField("err", err.Error()).WithField("topic", topic).Error("write kafka error")
	}
}

func (s *KafkaSinker) SinkBytes(topic string, data, key []byte) {
	err := s.kafkaProducer.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
		Value:          data,
		Key:            key,
	}, nil)

	if err != nil {
		log.WithField("err", err.Error()).WithField("topic", topic).Error("write kafka error")
	}
}

// Close close kafka producer
func (s *KafkaSinker) Close() {
	util.SafeClose(s.stopCh)

	select {
	case <-s.quitCh:
		return
	}
}

func newKafkaProducer(stopCh chan bool, quitCh chan bool, cfg *KafkaConfig, servers string) *kafka.Producer {
	if len(servers) == 0 {
		servers = cfg.BootstrapServers
	}
	kafkaProducer, err := kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers":            servers,
		"queue.buffering.max.kbytes":   cfg.QueueBufferingMaxKbytes,   // 缓存多大的信息，默认是400MB，最大能提高到2097151KB，也就是大约2GB
		"queue.buffering.max.messages": cfg.QueueBufferingMaxMessages, // 缓存多少个信息
		"linger.ms":                    cfg.LingerMs,                  // 多久发一次, 如果是0表示及时发送
	})
	if err != nil {
		panic(err)
	}

	flushTimeout := cfg.FlushTimeoutSec * 1000

	go func() {
		defer close(quitCh)
		events := kafkaProducer.Events()
		for {
			select {
			case e := <-events:
				switch ev := e.(type) {
				case *kafka.Message:
					if ev.TopicPartition.Error != nil {
						log.Error("Delivery failed")
					}
				}
			case <-std.Context().Done():
				start := time.Now()
				outstanding := kafkaProducer.Flush(flushTimeout)
				log.Infof("kafka stop on context done, un-flushed events %d; took %s", outstanding,
					time.Since(start).String())
				return
			case <-stopCh:
				start := time.Now()
				outstanding := kafkaProducer.Flush(flushTimeout)
				log.Infof("kafka stop on event, un-flushed events %d; took %s", outstanding,
					time.Since(start).String())
				return
			}
		}

	}()
	return kafkaProducer
}
