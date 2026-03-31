package dot

import (
	"context"
	"github.com/sirupsen/logrus"
	"testing"
	"time"
)

func TestRedisClient(t *testing.T) {
	WithRedis("redis://:123456@127.0.0.1:6379/0")
	start := time.Now()
	for i := 0; i < 1000000; i++ {
		Redis().LPush(context.Background(), "TestRedisClient", i)
	}
	logrus.WithField("cost", time.Now().Sub(start).Milliseconds()).Info("finish")
}

func TestRedisPipeline(t *testing.T) {
	WithRedis("redis://@127.0.0.1:6379/0")
	start := time.Now()
	pipeLiner := NewRedisPipeLine(Redis())
	for i := 0; i < 100; i++ {
		pipeLiner.LPush(context.Background(), "TestRedisPipeline", i)
		//pipeLiner.BatchOperation(pipeLiner.PipeLiner.LPush(context.Background(), "TestRedisPipeline", i))
	}
	logrus.WithField("cost", time.Now().Sub(start).Milliseconds()).Info("finish")
	time.Sleep(100 * time.Second)
}
