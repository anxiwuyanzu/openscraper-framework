package dot

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/anxiwuyanzu/openscraper-framework/spider-common-go/v4/util/serde"
	"github.com/tidwall/gjson"
	"io"
	"io/ioutil"
	"net/http"
	"time"
)

type Mq struct {
	mqUrl string
}

func NewMq(mqUrl string) *Mq {
	return &Mq{
		mqUrl: mqUrl,
	}
}

func (m *Mq) Get(topic string, size int) []Item {
	start := time.Now()
	uri := fmt.Sprintf("%s/get/%s/%d", m.mqUrl, topic, size)
	resp, err := http.Get(uri)
	if err != nil {
		std.Logger().WithField("url", uri).WithError(err).Error("failed to get mq")
		return []Item{}
	}
	defer resp.Body.Close()
	if took := time.Now().Sub(start).Seconds(); took > 10 {
		std.Logger().WithField("took", took).Warn("wait for get items from mq")
	}

	body, _ := ioutil.ReadAll(resp.Body)
	var items wrapItems
	err = serde.Unmarshal(body, &items)
	if err != nil {
		std.Logger().WithField("b", string(body)).Error("failed to parse json")
	}
	//if len(items.Data) == 0 {
	//	log.Info("mq size is zero")
	//}

	return items.Data
}

var ErrTooManyItems = errors.New("too many items")
var ErrIllegalPriority = errors.New("illegal priority")
var ErrIllegalTopic = errors.New("illegal topic")

func (m *Mq) Put(topic, source string, priority int, data []Item) error {
	if len(topic) == 0 {
		return ErrIllegalTopic
	}
	if priority < 0 || priority > 5 {
		return ErrIllegalPriority
	}
	if len(data) > 3000 {
		return ErrTooManyItems
	}

	uri := fmt.Sprintf("%s/put/%s/%d?source=%s", m.mqUrl, topic, priority, source)

	body := serde.Marshal(data)
	req, err := http.NewRequest("POST", uri, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Close = true

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ = io.ReadAll(resp.Body)
	src := gjson.ParseBytes(body)
	if src.Get("code").Int() != 0 {
		return errors.New(src.Get("msg").String())
	}
	return nil
}

type wrapItems struct {
	Code int    `json:"code"`
	Data []Item `json:"data"`
	Msg  string `json:"msg"`
}

type MqWriter struct {
	mq         *Mq
	topic      string
	source     string
	priority   int
	operations chan Item
}

func NewMqWriter(url string, topic, source string, priority int) *MqWriter {
	w := &MqWriter{
		mq:         NewMq(url),
		topic:      topic,
		source:     source,
		priority:   priority,
		operations: make(chan Item, 3000),
	}
	go w.flushLoop()
	return w
}

func (w *MqWriter) Add(data Item) {
	w.operations <- data
}

func (w *MqWriter) flushLoop() {
	ticker := time.NewTicker(10 * time.Second)
	var data []Item
	for {
		select {
		case item := <-w.operations:
			data = append(data, item)
			if len(data) >= 2000 {
				w.mq.Put(w.topic, w.source, w.priority, data)
				data = data[:0]
			}
		case <-ticker.C:
			if len(data) > 0 {
				w.mq.Put(w.topic, w.source, w.priority, data)
				data = data[:0]
			}
		case <-std.Context().Done():
			if len(data) > 0 {
				w.mq.Put(w.topic, w.source, w.priority, data)
				data = data[:0]
			}
			return
		}
	}
}
