package spider

import (
	"context"
	"time"
)

// Anchor 用来标记爬虫名 和 ItemCh topic; 利用编译期检查确保对象存在
// 祥见 ExampleAnchor_Register
type Anchor string

func (a Anchor) String() string {
	return string(a)
}

// Register 注册 Builder
func (a Anchor) Register(creator Builder) {
	register(a, creator)
}

// CreateItemCh 创建 ItemCh
func (a Anchor) CreateItemCh(maxSize int) ItemCh {
	return createItemCh(a, maxSize)
}

// AddItem 在爬虫运行时往 ItemCh 添加 item; 成功返回 1,
func (a Anchor) AddItem(item Item) int {
	return addItem(a, item)
}

// AddTimeoutItem 在爬虫运行时往 ItemCh 添加 ContextItem; 成功返回 1,
func (a Anchor) AddTimeoutItem(item Item, timeout time.Duration) (context.Context, int) {
	return addTimeoutItem(a, item, timeout)
}

// Depth 返回 ItemCh 长度
func (a Anchor) Depth() int {
	return depth(a)
}

func (a Anchor) GetSpiderState() (*Factory, bool) {
	return getSpiderState(a)
}
