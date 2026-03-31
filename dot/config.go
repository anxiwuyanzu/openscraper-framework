package dot

import (
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func ConfigViper(envPrefix string, path ...string) {
	viper.AutomaticEnv()
	if len(envPrefix) > 0 {
		viper.SetEnvPrefix(envPrefix)
	}

	viper.AllowEmptyEnv(true)
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")

	for _, p := range path {
		viper.AddConfigPath(p)
	}

	err := viper.ReadInConfig()
	if err != nil {
		logrus.WithError(err).Warn("failed to read config")
		return
	}
	logrus.Infof("Using config file: %s", viper.ConfigFileUsed())
	//Conf()
}

// ProjectRoot 返回工程目录, 简单使用 viper config 的目录
func ProjectRoot() string {
	if viper.ConfigFileUsed() != "." {
		return filepath.Dir(viper.ConfigFileUsed())
	}
	return "."
}

// Config 用作本库用到的所有配置, 避免 viper 满天飞
// 支持 bool, string, (u)int[8,32,64], map[string]string, []string, []int, time.Duration
// 如果是在环境变量里设置slice, 用空格隔开多个选项
type Config struct {
	Debug   bool         `yaml:"debug"`
	Sinker  string       `yaml:"sinker" default:"debug"` // kafka, debug, or dummy
	Spider  SpiderConfig `yaml:"spider"`
	Kafka   KafkaConfig  `yaml:"kafka"`
	Sql     SqlConfig    `yaml:"sql"`
	Redis   RedisConfig  `yaml:"redis"`
	Mongo   MongoConfig  `yaml:"mongo"`
	Amqp    AmqpConfig   `yaml:"amqp"`
	Proxy   ProxyConfig  `yaml:"proxy"`
	Mmq     MmqConfig    `yaml:"mmq"`
	Reqwest Reqwest      `yaml:"reqwest"`
}

type SpiderConfig struct {
	MaxRetryTimes int           `yaml:"max_retry_times" default:"0"`
	WorkerNum     int           `yaml:"worker_num" default:"1"`
	Delay         time.Duration `yaml:"delay" default:"0"`
	RandomDelay   time.Duration `yaml:"random_delay" default:"0"`
	Backlog       int           `yaml:"backlog" default:"3000"`
	// DOT_SPIDER_LOG_TAIL
	LogTail string `yaml:"log_tail"`
}

type KafkaConfig struct {
	BootstrapServers          string            `yaml:"bootstrap_servers"`
	QueueBufferingMaxKbytes   int               `yaml:"queue_buffering_max_kbytes" default:"2000000"`
	QueueBufferingMaxMessages int               `yaml:"queue_buffering_max_messages" default:"1000000"`
	LingerMs                  int               `yaml:"linger_ms" default:"100"`
	FlushTimeoutSec           int               `yaml:"flush_timeout_sec" default:"60"`
	Others                    map[string]string `yaml:"others"`
}

type SqlConfig struct {
	Uri string `yaml:"uri"`
	//Driver       string            `yaml:"driver" default:"mysql"`
	MaxOpenConns int               `yaml:"max_open_conns" default:"60"`
	Others       map[string]string `yaml:"others"`
}

type RedisConfig struct {
	Uri              string            `yaml:"uri"`
	PoolTimeout      time.Duration     `yaml:"pool_timeout" default:"20s"`
	PoolSize         int               `yaml:"pool_size" default:"10"`
	BatchWriteSize   int               `yaml:"batch_write_size" default:"1000"`
	FlushIntervalSec int               `yaml:"flush_interval_sec" default:"5"`
	RetryFlush       bool              `yaml:"retry_flush" default:"false"` // Flush 失败时重试
	DnsCache         bool              `yaml:"dns_cache" default:"false"`
	Others           map[string]string `yaml:"others"`
}

type MongoConfig struct {
	Uri              string            `yaml:"uri"`
	BatchWriteSize   int               `yaml:"batch_write_size" default:"1000"`
	FlushIntervalSec int               `yaml:"flush_interval_sec" default:"5"`
	FlushMin         int               `yaml:"flush_min" default:"0"`
	FlushWorker      int               `yaml:"flush_worker" default:"1"`
	LogVerbose       bool              `yaml:"log_verbose" default:"false"`
	Others           map[string]string `yaml:"others"`
}

type AmqpConfig struct {
	Uri    string            `yaml:"uri"`
	Others map[string]string `yaml:"others"`
}

type MmqConfig struct {
	Uri             string            `yaml:"uri"`
	Interval        time.Duration     `yaml:"interval" default:"2s"`
	IntervalOnEmpty time.Duration     `yaml:"intervalOnEmpty" default:"10s"`
	Verbose         bool              `yaml:"verbose" default:"false"`
	Others          map[string]string `yaml:"others"`
}

type Reqwest struct {
	Idle            time.Duration `yaml:"idle" default:"5s"`
	Timeout         time.Duration `yaml:"timeout" default:"30s"`
	DialTimeout     time.Duration `yaml:"dial_timeout" default:"15s"`
	ReadBufferSize  int           `yaml:"read_buffer_size" default:"4096"`
	MaxConnsPerHost int           `yaml:"max_conns_per_host" default:"1024"`
	// HttpVersion 包括 `1`, `2`, `3`
	HttpVersion int `yaml:"http_version" default:"1"`
	// Client 包括 `net/http` 和 `fasthttp`
	Client          string        `yaml:"client" default:"fasthttp"`
	ProxyDropIn     time.Duration `yaml:"proxy_drop_in" default:"5s"`
	MaxCallAttempts int           `yaml:"max_call_attempts" default:"1"`
	Concurrence     int64         `yaml:"concurrence"`
	ChromeVersion   string        `yaml:"chrome_version"`
}

type ProxyConfig struct {
	// DOT_PROXY_PROXY
	Proxy                  string   `yaml:"proxy"`
	KuaiApi                string   `yaml:"kuai_api" default:""`
	ZhiMaAcquireIpServer   string   `yaml:"zhima_acquire_ip_server" default:"http://zhima-acquire-ip.example.com"`
	ZhiMaSetWhitelistUri   string   `yaml:"zhima_set_whitelist_uri" default:"http://zhima-acquire-ip.example.com/zhima_white_ip?ip=%s"`
	QingGuoAcquireIpServer string   `yaml:"qingguo_acquire_ip_server" default:"http://qingguo-acquire-ip.example.com"`
	QingGuoSetWhitelistUri string   `yaml:"qingguo_set_whitelist_uri" default:"http://qingguo-acquire-ip.example.com/white?ip=%s"`
	KuaiAcquireIpServer    string   `yaml:"kuai_acquire_ip_server" default:"http://kuai-acquire-ip.example.com"`
	KuaiSetWhitelistUri    string   `yaml:"kuai_set_whitelist_uri" default:"http://kuai-acquire-ip.example.com/add_white?ip=%s"`
	RelayApi               string   `yaml:"relay_api" default:"http://proxy-relay.example.com"`
	GetLocalIpUris         []string `yaml:"get_local_ip_uris" default:"http://ms.cds8.cn/ip"`
}

func (c *GContext) InitFromViper() {
	cfg := &Config{}

	// 用 viper.UnmarshalKey 无法兼容环境变量
	viperSetDefault(reflect.ValueOf(cfg).Elem(), "dot")

	c.debug = cfg.Debug
	c.config = cfg
}

func viperSetDefault(valueObject reflect.Value, path string) {
	typeObject := valueObject.Type()
	count := valueObject.NumField()

	for i := 0; i < count; i++ {
		value := valueObject.Field(i)
		field := typeObject.Field(i)
		var key string
		if t := field.Tag.Get("yaml"); len(t) > 0 {
			key = path + "." + t
		} else {
			continue
		}

		if field.Type.Kind() == reflect.Struct {
			viperSetDefault(value, key)
		} else {
			if def := field.Tag.Get("default"); len(def) > 0 {
				viper.SetDefault(key, field.Tag.Get("default"))
			}

			switch field.Type.Kind() {
			case reflect.Bool:
				value.SetBool(viper.GetBool(key))
			case reflect.Int, reflect.Int8, reflect.Int64, reflect.Uint8, reflect.Uint64, reflect.Int32, reflect.Uint32:
				pkg := fmt.Sprintf("%s.%s", field.Type.PkgPath(), field.Type.Name())
				if pkg == "time.Duration" {
					value.SetInt(int64(viper.GetDuration(key)))
				} else {
					value.SetInt(viper.GetInt64(key))
				}
			case reflect.Float32, reflect.Float64:
				value.SetFloat(viper.GetFloat64(key))
			case reflect.String:
				value.SetString(viper.GetString(key))
			case reflect.Slice:
				elemKind := field.Type.Elem().Kind()
				if elemKind == reflect.String {
					setSlice(value, viper.GetStringSlice(key))
				} else if elemKind == reflect.Int {
					setSlice(value, viper.GetIntSlice(key))
				}
			case reflect.Map:
				if field.Type.Key().Kind() == reflect.String && field.Type.Elem().Kind() == reflect.String {
					m := viper.GetStringMapString(key)
					value.Set(reflect.MakeMap(value.Type()))
					for k, v := range m {
						value.SetMapIndex(reflect.ValueOf(k), reflect.ValueOf(v))
					}
				}
			}
		}
	}
}

func setSlice[T any](value reflect.Value, elements []T) {
	result := reflect.MakeSlice(value.Type(), len(elements), len(elements))
	for j := 0; j < len(elements); j++ {
		itemValue := result.Index(j)
		itemValue.Set(reflect.ValueOf(elements[j]))
	}
	value.Set(result)
}

func ViperString(key, def string) string {
	if v := viper.GetString(key); len(v) > 0 {
		return v
	}
	return def
}

func ViperInt(key string, def int) int {
	if v := viper.GetInt(key); v > 0 {
		return v
	}
	return def
}

func ViperInt64(key string, def int64) int64 {
	if v := viper.GetInt64(key); v > 0 {
		return v
	}
	return def
}

func ViperUint64(key string, def uint64) uint64 {
	if v := viper.GetUint64(key); v > 0 {
		return v
	}
	return def
}

func ViperBool(key string) bool {
	return viper.GetBool(key)
}
