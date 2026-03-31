package spider

import (
	"context"
	"github.com/anxiwuyanzu/openscraper-framework/v4/dot"
	"time"
)

// Item 代表一个爬虫任务
type Item interface {
	Id() string
	Marshal() ([]byte, error)
	Unmarshal(data []byte) error
	// Elapsed 记录放量到消费的时间, 单位秒
	Elapsed() int64
}

// ItemCh 代表爬虫任务队列
type ItemCh chan Item


// Pack 将多个item 组装打包到target
func (ch ItemCh) Pack(target ItemCh, interval, bulkSize int, fn func([]Item) Item) {
	d := time.Duration(interval) * time.Second
	if dot.Debug() {
		d = 300 * time.Millisecond
	}

	go func() {
		ticker := time.NewTicker(d)
		defer ticker.Stop()

		// 将多个item组合成一个
		var items []Item

		for {
			select {
			case item := <-ch:
				items = append(items, item)

				if len(items) >= bulkSize {
					if newItem := parseItems(items, fn); newItem != nil {
						target <- newItem
					}
					items = items[:0]
				}
			case <-ticker.C:
				if len(items) > 0 {
					if newItem := parseItems(items, fn); newItem != nil {
						target <- newItem
					}
					items = items[:0]
				}
			}
		}
	}()
}

// parseItems 返回新的item, 兼容 ContextItem 类型
func parseItems(items []Item, fn func([]Item) Item) Item {
	if ctxItem, ok := items[0].(*ContextItem); ok {
		// 去最早过期的时间
		if dl, ok := ctxItem.Deadline(); ok {
			ctx, cancel := context.WithDeadline(context.Background(), dl)
			packed := &ContextItem{Context: ctx, cancelFunc: cancel}

			rawItems := make([]Item, len(items))
			for i, item := range items {
				rawItems[i] = packed.addChild(item)
			}
			packed.Item = fn(rawItems)
			return packed
		}
	}

	return fn(items)
}

// ContextItem 包含 Context 的 Item; 可以同时利用 Context 行为;
type ContextItem struct {
	Item
	context.Context
	cancelFunc context.CancelFunc
	children   []*ContextItem // 没有使用map, 怕 withValue 结果会被覆盖
	values     map[string]map[string]any
}

func (item *ContextItem) addChild(child Item) Item {
	if subCtxItem, isCtxItem := child.(*ContextItem); isCtxItem {
		item.children = append(item.children, subCtxItem)
		return subCtxItem.Item
	} else {
		return item
	}
}

func (item *ContextItem) cancel() {
	if item.Err() != nil {
		return // already canceled
	}

	if item.cancelFunc != nil {
		item.cancelFunc()
	}

	for _, child := range item.children {
		// 为每个 child 附加 value
		for key, values := range item.values {
			child.Context = context.WithValue(child.Context, key, values[child.Id()])
		}
		if child.cancelFunc != nil {
			child.cancelFunc()
		}
	}
}

// withValue 设置key value; 可以为 child 设置
func (item *ContextItem) withValue(key string, value any) {
	item.Context = context.WithValue(item.Context, key, value)
}

// withChildCtxValue 为 child 设置 value; 等 cancel 时附加到 item.values
func (item *ContextItem) withChildCtxValue(key, id string, value any) {
	if item.values == nil {
		item.values = make(map[string]map[string]any)
	}
	if sub, ok := item.values[key]; ok {
		sub[id] = value
	} else {
		item.values[key] = make(map[string]any, len(item.children))
		item.values[key][id] = value
	}
}
