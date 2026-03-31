package redis_ext

import (
	"context"
	"fmt"
	"github.com/anxiwuyanzu/openscraper-framework/spider-common-go/v4/util"
	"github.com/go-redis/redis/v8"
	"sync"
)

const (
	ScanDefault  int = 0 // 默认 不包含 cursor 和 value
	ScanIncluded int = 1 // scan 的时候 key 包含 cursor
	ScanValue    int = 2 // scan 的时候返回 value
)

type KvScanCmd struct {
	baseCmd *redis.Cmd

	page    []string
	cursor  string
	flag    int
	process cmdable
	args    []interface{}
}

func NewKvScanCmd(ctx context.Context, process cmdable, cursor, match string, count int64, flag int) *KvScanCmd {
	args := []interface{}{"scan", cursor, "match", match, "count", count, "flag", flag}
	cmd := redis.NewCmd(ctx, args...)
	return &KvScanCmd{
		baseCmd: cmd,
		flag:    flag,
		process: process,
		args:    args,
	}
}

type KeyValue struct {
	Key   string
	Value []byte
}

func (cmd *KvScanCmd) KeyVal() (kvs []*KeyValue, cursor string, err error) {
	if cmd.flag&ScanValue == 0 {
		panic("don't support KeyVal as not scan value")
	}
	err = cmd.readReply()
	if err != nil {
		return nil, cmd.cursor, err
	}
	for i := 0; i < len(cmd.page); i += 2 {
		kvs = append(kvs, &KeyValue{Key: cmd.page[i], Value: util.StringToBytes(cmd.page[i+1])})
	}
	return kvs, cmd.cursor, cmd.baseCmd.Err()
}

func (cmd *KvScanCmd) readReply() (err error) {
	cmd.page, cmd.cursor, err = cmd.Val()
	return err
}

func (cmd *KvScanCmd) Val() (keys []string, cursor string, err error) {
	slice, err := cmd.baseCmd.Slice()
	if err != nil {
		return nil, "", err
	}

	if len(slice) != 2 {
		return nil, "", fmt.Errorf("redis: got %d elements in scan reply, expected 2", len(slice))
	}

	cursor, err = toString(slice[0])
	if err != nil {
		return nil, "", err
	}

	data, err := toSlice(slice[1])
	if err != nil {
		return nil, "", err
	}
	n := len(data)

	keys = make([]string, n)

	for i := 0; i < n; i++ {
		key, err := toString(data[i])
		if err != nil && err != redis.Nil {
			return nil, "", err
		}
		keys[i] = key
	}

	return keys, cursor, err
}

// Iterator creates a new ScanIterator.
func (cmd *KvScanCmd) Iterator() *KvScanIterator {
	_ = cmd.readReply()
	return &KvScanIterator{
		cmd: cmd,
	}
}

// KvScanIterator is used to incrementally iterate over a collection of elements.
// It's safe for concurrent use by multiple goroutines.
type KvScanIterator struct {
	mu  sync.Mutex // protects Scanner and pos
	cmd *KvScanCmd
	pos int
}

// Err returns the last iterator error, if any.
func (it *KvScanIterator) Err() error {
	it.mu.Lock()
	err := it.cmd.baseCmd.Err()
	it.mu.Unlock()
	return err
}

// Next advances the cursor and returns true if more values can be read.
func (it *KvScanIterator) Next(ctx context.Context) bool {
	it.mu.Lock()
	defer it.mu.Unlock()

	// Instantly return on errors.
	if it.cmd.baseCmd.Err() != nil {
		return false
	}

	// Advance cursor, check if we are still within range.
	if it.pos < len(it.cmd.page) {
		it.pos++
		return true
	}

	for {
		// Return if there is no more data to fetch.
		if it.cmd.cursor == "0" {
			return false
		}

		// Fetch next page.
		it.cmd.args[1] = it.cmd.cursor
		it.cmd.baseCmd = redis.NewCmd(ctx, it.cmd.args...)
		err := it.cmd.process(ctx, it.cmd.baseCmd)
		if err != nil {
			return false
		}
		_ = it.cmd.readReply()

		it.pos = 1

		// Redis can occasionally return empty page.
		if len(it.cmd.page) > 0 {
			return true
		}
	}
}

// Val returns the key/field at the current cursor position.
func (it *KvScanIterator) Val() string {
	var v string
	it.mu.Lock()
	if it.cmd.baseCmd.Err() == nil && it.pos > 0 && it.pos <= len(it.cmd.page) {
		v = it.cmd.page[it.pos-1]
	}
	it.mu.Unlock()
	return v
}

func (it *KvScanIterator) KeyVal() (k string, v string) {
	if it.cmd.flag&ScanValue == 0 {
		panic("don't support KeyVal as not scan value")
	}
	it.mu.Lock()
	if it.cmd.baseCmd.Err() == nil && it.pos > 0 && it.pos <= len(it.cmd.page) {
		k = it.cmd.page[it.pos-1]
		it.pos++
		if it.pos <= len(it.cmd.page) {
			v = it.cmd.page[it.pos-1]
		}
	}
	it.mu.Unlock()
	return k, v
}

func (it *KvScanIterator) KeyValBytes() (k string, v []byte) {
	if it.cmd.flag&ScanValue == 0 {
		panic("don't support KeyVal as not scan value")
	}
	it.mu.Lock()
	if it.cmd.baseCmd.Err() == nil && it.pos > 0 && it.pos <= len(it.cmd.page) {
		k = it.cmd.page[it.pos-1]
		it.pos++
		if it.pos <= len(it.cmd.page) {
			v = util.StringToBytes(it.cmd.page[it.pos-1])
		}
	}
	it.mu.Unlock()
	return k, v
}
