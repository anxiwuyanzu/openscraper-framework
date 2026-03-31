package dot

import (
	"context"
	"github.com/anxiwuyanzu/openscraper-framework/v4/reqwest/dnscache"
	"github.com/anxiwuyanzu/openscraper-framework/v4/util"
	"github.com/go-redis/redis/v8"
	"github.com/sirupsen/logrus"
	"net"
	"time"
)

func NewRedisClient(addr string) *redis.Client {
	return NewRedisClientWithConfig(addr, nil)
}

// NewRedisClient create redid client; addr as:
// redis://<user>:<password>@<host>:<port>/<db_number>
// example: redis://:password@10.64.108.1:6379/0
func NewRedisClientWithConfig(addr string, cfg *RedisConfig) *redis.Client {
	if cfg == nil {
		cfg = &Conf().Redis
	}
	if u, ok := cfg.Others[addr]; ok {
		addr = u
	}

	if len(addr) == 0 {
		return nil
	}

	option, err := redis.ParseURL(addr)
	if err != nil {
		logrus.WithField("url", addr).WithError(err).Warn("failed to parse redis addr")
	}

	option.PoolTimeout = cfg.PoolTimeout
	option.PoolSize = cfg.PoolSize

	if cfg.DnsCache {
		netDialer := &net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 5 * time.Minute,
		}
		option.Dialer = dnscache.DialFunc(netDialer.DialContext)
	}

	rdb := redis.NewClient(option)

	_, err = rdb.Ping(context.Background()).Result()
	if err != nil {
		logrus.WithField("addr", addr).WithError(err).Info("failed to ping redis")
		panic(err)
	}
	return rdb
}

// RedisPipeLine redis 批量写入功能; pipeline 和redis集群兼容性不好，要修改成集群版本的谨慎！！！
type RedisPipeLine struct {
	*redis.Pipeline

	logger    *logrus.Entry
	stopCh    chan bool
	quitCh    chan bool
	batchSize int
	retry     bool
}

// NewRedisPipeLine create redis pipeline worker
func NewRedisPipeLine(redisClient *redis.Client) *RedisPipeLine {
	return NewRedisPipeLineWithConfig(redisClient, nil)
}

func NewRedisPipeLineWithConfig(redisClient *redis.Client, cfg *RedisConfig) *RedisPipeLine {
	if cfg == nil {
		cfg = &Conf().Redis
	}

	batchSize := cfg.BatchWriteSize

	pl := redisClient.Pipeline().(*redis.Pipeline)

	p := &RedisPipeLine{
		Pipeline:  pl,
		logger:    std.Logger(),
		stopCh:    make(chan bool, 1),
		quitCh:    make(chan bool, 1),
		batchSize: batchSize,
		retry:     cfg.RetryFlush,
	}

	go p.flushLoop(cfg)
	EnsureClose(p.Close)
	return p
}

func (r *RedisPipeLine) Close() {
	util.SafeClose(r.stopCh)
	select {
	case <-r.quitCh:
		return
	}
}

func (r *RedisPipeLine) flushLoop(cfg *RedisConfig) {
	defer util.SafeClose(r.quitCh)
	defer r.Pipeline.Close()

	ticker := time.NewTicker(time.Duration(cfg.FlushIntervalSec) * time.Second)

	for {
		select {
		case <-ticker.C:
			r.exec()
		case <-std.Context().Done():
			r.logger.Info("redis stop")
			r.exec()
			return
		case <-r.stopCh:
			r.logger.Info("redis stop")
			r.exec()
			return
		default:
			if r.Len() > r.batchSize {
				r.exec()
			} else {
				time.Sleep(500 * time.Millisecond)
			}
		}
	}
}

func (r *RedisPipeLine) exec() {
	l := r.Len()
	cmds, err := r.Exec(context.Background())
	if err != nil {
		r.logger.WithError(err).Errorf("redis pipeline exec error; count: %d", l)

		if r.retry && len(cmds) > 0 {
			for _, cmd := range cmds {
				r.Process(context.Background(), cmd)
			}
		}
	}
}
