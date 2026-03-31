package source

import (
	"github.com/anxiwuyanzu/openscraper-framework/v4/dot"
	"github.com/anxiwuyanzu/openscraper-framework/v4/spider"
	"time"
)

// MqSource 定时从mmq获取任务Item, 写到 itemCh
type MqSource struct {
	// mmq 的 topic
	topic string
	// 每次获取多少条
	size int
	// 间隔时间
	interval time.Duration
	// 如果上次获取到item数为0, 间隔时间
	intervalOnWithoutItems time.Duration
	verbose                bool

	mq     *dot.Mq
	itemCh chan spider.Item
}

func NewMqSource(mmqServer, topic string, size int, itemCh chan spider.Item) *MqSource {
	return NewMqSourceWithConfig(mmqServer, topic, size, itemCh, nil)
}

func NewMqSourceWithConfig(mmqServer, topic string, size int, itemCh chan spider.Item, cfg *dot.MmqConfig) *MqSource {
	if cfg == nil {
		cfg = &dot.Conf().Mmq
	}
	if u, ok := dot.Conf().Mmq.Others[mmqServer]; ok {
		mmqServer = u
	}

	return &MqSource{
		verbose:                cfg.Verbose,
		interval:               cfg.Interval,
		intervalOnWithoutItems: cfg.IntervalOnEmpty,
		topic:                  topic,
		size:                   size,
		mq:                     dot.NewMq(mmqServer),
		itemCh:                 itemCh,
	}
}

func (s *MqSource) Start(workerSize int) {
	minSize := workerSize * 50
	start := time.Now()
	for {
		if s.verbose && time.Now().Sub(start).Seconds() > 180 {
			start = time.Now()
			dot.Logger().WithField("topic", s.topic).WithField("backlog", len(s.itemCh)).Info("items")
		}

		if len(s.itemCh) < minSize {
			items := s.mq.Get(s.topic, s.size)
			if s.verbose {
				dot.Logger().WithField("topic", s.topic).WithField("backlog", len(s.itemCh)).WithField("n", len(items)).Info("get items")
			}
			if len(items) == 0 {
				time.Sleep(s.intervalOnWithoutItems)
			}
			for _, item := range items {
				s.itemCh <- item
			}
		} else {
			time.Sleep(s.interval)
		}
	}
}

func (s *MqSource) ItemCh() chan spider.Item {
	return s.itemCh
}
