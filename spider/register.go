package spider

import (
	"context"
	"sync"
	"time"

	"github.com/anxiwuyanzu/openscraper-framework/v4/dot"
)

// Builder 构造 Factory 的方法
type Builder func() *Factory

var (
	lock = sync.RWMutex{}
	// factories 维护全局 Builder;
	factories = make(map[Anchor]Builder, 10)
	// spiderItemCh 维护全局 ItemCh
	spiderItemCh = make(map[Anchor]ItemCh, 5)
	// runningSpiders 维护所有启动的spider, 避免重复启动(多个爬虫的sub spider可能存在重复)
	runningSpiders = make(map[Anchor]*Factory, 10)
)

// GetAllSpiders 获取爬虫列表
func GetAllSpiders() []Anchor {
	spiders := make([]Anchor, 0, len(factories))
	for a := range factories {
		spiders = append(spiders, a)
	}
	return spiders
}

func Register(name string, creator Builder) {
	factories[Anchor(name)] = creator
}

// register 注册 Builder (内部方法)
func register(name Anchor, creator Builder) {
	factories[name] = creator
}

// createItemCh 创建 ItemCh
func createItemCh(topic Anchor, maxSize int) ItemCh {
	lock.Lock()
	defer lock.Unlock()
	ch, ok := spiderItemCh[topic]
	if !ok {
		spiderItemCh[topic] = make(ItemCh, maxSize)
		return spiderItemCh[topic]
	}
	return ch
}

func AddItem(itemChanName string, item Item) int {
	return addItem(Anchor(itemChanName), item)
}

// addItem 在爬虫运行时往 ItemCh 添加 item; 成功返回 1,
func addItem(itemChanName Anchor, item Item) int {
	if itemCh, ok := spiderItemCh[itemChanName]; ok {
		select {
		case itemCh <- item:
			return 1
		default:
			return 0
		}
	} else {
		// 使用 Panic 过度代码兼容
		dot.Logger().WithField("ch", itemChanName).Panic("topic not found")
	}
	return 0
}

// addContextItem 在爬虫运行时往 ItemCh 添加 ContextItem; 成功返回 1
// ContextItem 主要为了可以 Cancel
//
//	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
//	ctx, _ = addContextItem(ctx, cancel, "dy-author", dot.Item{"id": "111"})
//	<- ctx.Done()
//	fmt.Println(ctx.Value("result"))
func addContextItem(ctx context.Context, cancel context.CancelFunc, itemChanName Anchor, item Item) (context.Context, int) {
	if itemCh, ok := spiderItemCh[itemChanName]; ok {
		ctxItem := &ContextItem{Item: item, Context: ctx, cancelFunc: cancel}
		select {
		case itemCh <- ctxItem:
			return ctxItem, 1
		default:
			ctxItem.cancel()
			return ctxItem, 0
		}
	} else {
		dot.Logger().WithField("ch", itemChanName).Panic("topic not found")
	}
	return ctx, 0
}

func AddTimeoutItem(itemChanName string, item Item, timeout time.Duration) (context.Context, int) {
	return addTimeoutItem(Anchor(itemChanName), item, timeout)
}

// addTimeoutItem 在爬虫运行时往 ItemCh 添加 ContextItem; 任务超时会自动取消;
func addTimeoutItem(itemChanName Anchor, item Item, timeout time.Duration) (context.Context, int) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	return addContextItem(ctx, cancel, itemChanName, item)
}

// depth 返回 ItemCh 长度
func depth(itemChanName Anchor) int {
	if itemCh, ok := spiderItemCh[itemChanName]; ok {
		return len(itemCh)
	} else {
		dot.Logger().WithField("ch", itemChanName).Error("topic not found")
	}
	return -1
}

func markRunning(name Anchor, f *Factory) bool {
	lock.Lock()
	defer lock.Unlock()
	if _, ok := runningSpiders[name]; ok {
		return false
	}
	runningSpiders[name] = f
	return true
}

func getSpiderState(name Anchor) (*Factory, bool) {
	lock.RLock()
	defer lock.RUnlock()
	f, ok := runningSpiders[name]
	return f, ok
}
