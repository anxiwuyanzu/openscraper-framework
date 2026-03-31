package dot

import (
	"fmt"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
)

func initViper() {
	viper.SetConfigName("config.example")
	viper.SetConfigType("yaml")
	viper.AutomaticEnv()
	//viper.SetEnvPrefix("C")
	viper.AllowEmptyEnv(true)

	viper.AddConfigPath(".")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	err := viper.ReadInConfig()
	if err != nil {
		panic(err)
	}
	fmt.Println("Using config file:", viper.ConfigFileUsed())
}

func TestConfig(t *testing.T) {
	assert := require.New(t)
	initViper()

	os.Setenv("DOT_SPIDER_WORKER_NUM", "3")
	os.Setenv("DOT_SPIDER_MAX_RETRY_TIMES", "3")
	os.Setenv("DOT_SPIDER_RANDOM_DELAY", "3ms")
	os.Setenv("DOT_PROXY_GET_LOCAL_IP_URIS", "u1 u2")

	start := time.Now()
	std.InitFromViper()
	fmt.Println(time.Since(start))

	cfg := std.config
	assert.Equal(cfg.Debug, false)
	assert.Equal(cfg.Sinker, "kafka")
	assert.Equal(cfg.Sql.MaxOpenConns, 60)

	//assert.Equal(cfg.Proxy.GetLocalIpUris, "http://ms.cds8.cn/ip")
	fmt.Printf("%#v\n", cfg.Proxy.GetLocalIpUris)

	assert.Equal(cfg.Kafka.BootstrapServers, "127.0.0.1:9092")
	assert.Equal(cfg.Kafka.QueueBufferingMaxKbytes, 1000000)
	assert.Equal(cfg.Reqwest.Timeout, 30*time.Second)

	assert.Equal(ViperInt("dot.spider.worker_num", 1), 3)
	assert.Equal(cfg.Spider.WorkerNum, 3)

	assert.Equal(ViperInt("dot.spider.max_retry_times", 1), 3)
	assert.Equal(cfg.Spider.MaxRetryTimes, 3)

	assert.Equal(len(cfg.Redis.Others), 2)
	assert.Equal(cfg.Redis.Others["dy"], "redis://127.0.0.1:3690")
	assert.Equal(cfg.Spider.Delay, time.Duration(0))
	assert.Equal(cfg.Spider.RandomDelay, 3*time.Millisecond)
	assert.Equal(cfg.Reqwest.ProxyDropIn, 45*time.Second)
}

func TestDefault(t *testing.T) {
	cfg := &Config{}
	valueObject := reflect.ValueOf(cfg).Elem()
	viperSetDefault(valueObject, "dot")
}

func TestGetDriverName(t *testing.T) {
	addr := "mysql://user:pwd@tcp(rm-cmm01.mysql.rds.aliyuncs.com:3306)/dbname?charset=utf8mb4&parseTime=true&loc=PRC"
	require.Equal(t, "mysql", getDriverName(addr))
	addr = "postgres://postgres:password@localhost/DB_1?sslmode=disable"
	require.Equal(t, "postgres", getDriverName(addr))
	addr = "user:pwd@tcp(rm-cmm01.mysql.rds.aliyuncs.com:3306)/dbname?charset=utf8mb4&parseTime=true&loc=PRC"
	require.Equal(t, "mysql", getDriverName(addr))
}
