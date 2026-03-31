package serde

import (
	jsoniter "github.com/json-iterator/go"
	"github.com/tidwall/sjson"
	"github.com/valyala/fastjson"
)

var (
	Json = jsoniter.ConfigCompatibleWithStandardLibrary
)

func Marshal(v interface{}) []byte {
	buf, _ := Json.Marshal(v)
	return buf
}

func MarshalToString(v interface{}) string {
	str, _ := Json.MarshalToString(v)
	return str
}

func Unmarshal(data []byte, v interface{}) error {
	return Json.Unmarshal(data, v)
}

var fastjsonPool fastjson.ParserPool

func AcquireFastjsonParser() *fastjson.Parser {
	return fastjsonPool.Get()
}

func ReleaseFastjsonParser(parser *fastjson.Parser) {
	fastjsonPool.Put(parser)
}

type SJson struct {
	inner string
}

func NewSJson() *SJson {
	return &SJson{inner: "{}"}
}

func (s *SJson) Set(path string, value interface{}) {
	s.inner, _ = sjson.Set(s.inner, path, value)
}

func (s *SJson) String() string {
	return s.inner
}
