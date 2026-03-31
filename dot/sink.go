package dot

import (
	"fmt"
	"github.com/anxiwuyanzu/openscraper-framework/v4/util/serde"
)

type ISinker interface {
	Sink(string, interface{}, ...string)
	SinkString(string, string, ...string)
	SinkBytes(string, []byte, []byte)
	Close()
}

func NewDebugSinker(sinker string) ISinker {
	return &DebugSinker{
		sinker: sinker,
	}
}

type DebugSinker struct {
	sinker string
}

func (s *DebugSinker) Sink(topic string, data interface{}, key ...string) {
	if s.sinker == "debug" {
		var msgKey []byte
		if len(key) == 1 {
			msgKey = []byte(key[0])
		}
		buf, _ := serde.Json.MarshalIndent(data, "", "  ")
		fmt.Println(topic, string(buf), string(msgKey))
	}
}

func (s *DebugSinker) SinkString(topic string, data string, key ...string) {
	if s.sinker == "debug" {
		fmt.Println(topic, data)
	}
}

func (s *DebugSinker) SinkBytes(topic string, data, key []byte) {
	if s.sinker == "debug" {
		fmt.Println(topic, string(data))
	}
}

func (s *DebugSinker) Close() {

}
