package redis_ext

import (
	"context"
	"fmt"
	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestKvScanVal(t *testing.T) {
	assert := require.New(t)
	option, err := redis.ParseURL("redis://10.65.22.148:6666")
	assert.Nil(err)

	ctx := context.Background()
	rdb := NewRedisExt(redis.NewClient(option))
	val, c, err := rdb.KvScan(ctx, "S:0000N", "S:*", 10, 0).Val()
	fmt.Println(val, c, err)
}

func TestKvScanIter(t *testing.T) {
	assert := require.New(t)
	option, err := redis.ParseURL("redis://10.65.22.148:6666")
	assert.Nil(err)

	ctx := context.Background()
	rdb := NewRedisExt(redis.NewClient(option))
	iter := rdb.KvScan(ctx, "S:0000N", "S:*", 10, 2).Iterator()

	n := 0
	for iter.Next(ctx) {
		n++
		fmt.Println(iter.KeyVal())
		if n > 32 {
			break
		}
	}
}
