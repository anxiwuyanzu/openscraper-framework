package dot

import (
	"encoding/json"
	"fmt"
	"github.com/anxiwuyanzu/openscraper-framework/v4/util/serde"
	jsoniter "github.com/json-iterator/go"
	log "github.com/sirupsen/logrus"
	"strconv"
)

type Item map[string]any

func (item Item) Id() string {
	return item.GetString("id")
}

func (item Item) IdAny() any {
	return item.GetString("id")
}

func (item Item) Get(key string) (v any, ok bool) {
	v, ok = item[key]
	return
}

func (item Item) GetString(key string) string {
	v, ok := item[key]
	if !ok {
		return ""
	}

	switch v.(type) {
	case string:
		return v.(string)
	default:
		return fmt.Sprintf("%v", v)
	}
}

func (item Item) GetStringOr(key string, orFunc func() string) string {
	v, ok := item[key]
	if !ok {
		def := orFunc()
		item[key] = def
		return def
	}

	switch v.(type) {
	case string:
		return v.(string)
	default:
		return fmt.Sprintf("%v", v)
	}
}

func (item Item) GetIntOr(key string, orFunc func() int) int {
	_, ok := item[key]
	if !ok {
		def := orFunc()
		item[key] = def
		return def
	}
	return item.GetInt(key)
}

func (item Item) GetInt(key string) int {
	v, ok := item[key]
	if !ok {
		return 0
	}

	switch v.(type) {
	case string:
		vi, err := strconv.ParseInt(v.(string), 10, 64)
		if err != nil {
			log.WithField("v", v.(string)).WithError(err).Error("failed to parse int")
			return 0
		}
		return int(vi)
	case float64:
		return int(v.(float64))
	case int:
		return v.(int)
	case int64:
		return int(v.(int64))
	case json.Number:
		vi, err := v.(json.Number).Int64()
		if err != nil {
			log.WithField("v", v).WithError(err).Error("failed to parse int")
			return 0
		}
		return int(vi)
	case jsoniter.Number:
		vi, err := v.(jsoniter.Number).Int64()
		if err != nil {
			log.WithField("v", v).WithError(err).Error("failed to parse int")
			return 0
		}
		return int(vi)
	default:
		return 0
	}
}

func (item Item) GetInt64Or(key string, orFunc func() int64) int64 {
	_, ok := item[key]
	if !ok {
		def := orFunc()
		item[key] = def
		return def
	}
	return item.GetInt64(key)
}

func (item Item) GetInt64(key string) int64 {
	v, ok := item[key]
	if !ok {
		return 0
	}

	switch v.(type) {
	case string:
		vi, err := strconv.ParseInt(v.(string), 10, 64)
		if err != nil {
			log.WithField("v", v.(string)).WithError(err).Error("failed to parse int")
			return 0
		}
		return vi
	case float64:
		return int64(v.(float64))
	case int:
		return int64(v.(int))
	case int64:
		return v.(int64)
	case json.Number:
		vi, err := v.(json.Number).Int64()
		if err != nil {
			log.WithField("v", v).WithError(err).Error("failed to parse int")
			return 0
		}
		return vi
	case jsoniter.Number:
		vi, err := v.(jsoniter.Number).Int64()
		if err != nil {
			log.WithField("v", v).WithError(err).Error("failed to parse int")
			return 0
		}
		return vi
	default:
		return 0
	}
}

func (item Item) GetFloat64(key string) float64 {
	v, ok := item[key]
	if !ok {
		return 0
	}

	switch v.(type) {
	case string:
		vi, err := strconv.ParseFloat(v.(string), 64)
		if err != nil {
			log.WithField("v", v.(string)).WithError(err).Error("failed to parse float64")
			return 0
		}
		return vi
	case float64:
		return v.(float64)
	case int:
		return float64(v.(int))
	case int64:
		return float64(v.(int64))
	case json.Number:
		vi, err := v.(json.Number).Float64()
		if err != nil {
			log.WithField("v", v).WithError(err).Error("failed to parse float64")
			return 0
		}
		return vi
	case jsoniter.Number:
		vi, err := v.(jsoniter.Number).Float64()
		if err != nil {
			log.WithField("v", v).WithError(err).Error("failed to parse float64")
			return 0
		}
		return vi
	default:
		return 0
	}
}

func (item Item) GetBool(key string) bool {
	v, ok := item[key]
	if !ok {
		return false
	}

	switch v.(type) {
	case string:
		return len(v.(string)) > 0
	case float64:
		return v.(float64) > 0
	case int:
		return v.(int) > 0
	case int64:
		return v.(int64) > 0
	case bool:
		return v.(bool)
	default:
		return false
	}
}

func (item Item) Set(key string, val any) {
	item[key] = val
}

func (item Item) Clone() Item {
	newItem := Item{}
	for k, v := range item {
		newItem[k] = v
	}
	return newItem
}

func (item Item) Marshal() ([]byte, error) {
	buf := serde.Marshal(item)
	return buf, nil
}

func (item Item) Unmarshal(data []byte) error {
	return serde.Unmarshal(data, &item)
}

func (item Item) Elapsed() int64 {
	return 0
}
