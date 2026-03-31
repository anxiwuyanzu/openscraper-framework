package redis_ext

import (
	"context"
	"fmt"
	"github.com/go-redis/redis/v8"
)

type cmdable func(ctx context.Context, cmd redis.Cmder) error

type RedisExt struct {
	cmdable
	*redis.Client
}

func NewRedisExt(rdb *redis.Client) *RedisExt {
	return &RedisExt{
		cmdable: rdb.Process,
		Client:  rdb,
	}
}

func (c cmdable) KvScan(ctx context.Context, cursor string, match string, count int64, flag int) *KvScanCmd {
	cmd := NewKvScanCmd(ctx, c, cursor, match, count, flag)
	_ = c(ctx, cmd.baseCmd)
	return cmd
}

func toString(val interface{}) (string, error) {
	switch val := val.(type) {
	case string:
		return val, nil
	default:
		err := fmt.Errorf("redis: unexpected type=%T for String", val)
		return "", err
	}
}

func toSlice(val interface{}) ([]interface{}, error) {
	switch val := val.(type) {
	case []interface{}:
		return val, nil
	default:
		return nil, fmt.Errorf("redis: unexpected type=%T for Slice", val)
	}
}
