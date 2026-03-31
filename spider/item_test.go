package spider

import (
	"context"
	"fmt"
	"github.com/anxiwuyanzu/openscraper-framework/v4/dot"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
	"time"
)

func TestAnchor(t *testing.T) {
	assert := require.New(t)
	var T1 Anchor = "test1"
	var T2 Anchor = "test1"
	T1.Register(nil)
	T2.Register(nil)

	assert.Equal(1, len(factories))
	assert.Equal(T1, T2)
}

const (
	TestAddItemSpider       Anchor = "test/add-item"
	TestAddPackedItemSpider Anchor = "test/add-packed-item"
	ChAddPackedItem         Anchor = "ch/test/add-packed-item"
)

// 测试 addItem 用爬虫。
type testAddItemSpider struct {
	Application
}

func (s *testAddItemSpider) Start(ctx Context) {
	id := ctx.Params().Id()
	ctx.WithCtxValue("test-id", id)
	fmt.Println(ctx.CtxValue("test-id"))
}

func TestAddItem(t *testing.T) {
	// 注册爬虫
	TestAddItemSpider.Register(func() *Factory {
		return &Factory{
			SourceFactory: func(itemCh ItemCh, workerNum, mode int) {},
			SpiderFactory: func(logger *log.Entry) Spider {
				s := &testAddItemSpider{}
				return s
			},
		}
	})
	engine := NewEngine()
	go engine.StartForever("test/add-item")

	time.Sleep(time.Second) // wait for engine init.

	assert := require.New(t)
	ctx, added := TestAddItemSpider.AddTimeoutItem(dot.Item{"id": "000"}, 10*time.Second)
	assert.Equal(1, added)

	<-ctx.Done()
	assert.Equal(ctx.Value("test-id").(string), "000")
}

// 测试 AddPackedItem 用爬虫。
type testAddPackedItemSpider struct {
	Application
}

func (s *testAddPackedItemSpider) Start(ctx Context) {
	id := ctx.Params().Id()
	spl := strings.Split(id, ",")

	for _, itemId := range spl {
		ctx.WithChildCtxValue("test-id", itemId, itemId)
		ctx.WithChildCtxValue("test-id2", itemId, itemId+"2")
	}
	if sleep := ctx.Params().(dot.Item).GetInt("sleep"); sleep > 0 {
		time.Sleep(time.Duration(sleep) * time.Second)
	}
}

func TestAddPacked(t *testing.T) {
	// 注册爬虫
	TestAddPackedItemSpider.Register(func() *Factory {
		return &Factory{
			SourceFactory: func(itemCh ItemCh, workerNum, mode int) {
				// 通过 ChAddPackedItem 向本爬虫添加item, 并且将多个item组合为一个item; 模拟批量请求
				ch := ChAddPackedItem.CreateItemCh(100)
				ch.Pack(itemCh, 3, 50, func(items []Item) Item {
					idJoin := make([]string, len(items))
					sleep := 0
					for i, item := range items {
						idJoin[i] = item.Id()
						if item.(dot.Item).GetInt("sleep") > sleep {
							sleep = item.(dot.Item).GetInt("sleep")
						}
					}
					return dot.Item{"id": strings.Join(idJoin, ","), "sleep": sleep}
				})
			},
			SpiderFactory: func(logger *log.Entry) Spider {
				s := &testAddPackedItemSpider{}
				return s
			},
		}
	})
	engine := NewEngine()
	go engine.StartForever("test/add-packed-item")

	time.Sleep(time.Second) // wait for engine init.

	assert := require.New(t)
	ctx1, added := ChAddPackedItem.AddTimeoutItem(dot.Item{"id": "001"}, 10*time.Second)
	assert.Equal(1, added)

	ctx2, added := ChAddPackedItem.AddTimeoutItem(dot.Item{"id": "002"}, 10*time.Second)
	assert.Equal(1, added)

	ctx3, added := ChAddPackedItem.AddTimeoutItem(dot.Item{"id": "003"}, 10*time.Second)
	assert.Equal(1, added)

	<-ctx1.Done()
	<-ctx2.Done()
	<-ctx3.Done()

	fmt.Println("test multi")
	assert.Equal(ctx1.Value("test-id").(string), "001")
	assert.Equal(ctx2.Value("test-id").(string), "002")
	assert.Equal(ctx3.Value("test-id").(string), "003")
	assert.Equal(ctx3.Value("test-id2").(string), "0032")
	assert.Equal(ctx3.Err(), context.Canceled)

	fmt.Println("test single")
	ctx4, _ := ChAddPackedItem.AddTimeoutItem(dot.Item{"id": "004"}, 10*time.Second)
	<-ctx4.Done()
	assert.Equal(ctx4.Value("test-id").(string), "004")
	assert.Equal(ctx4.Value("test-id2").(string), "0042")

	// test timeout
	fmt.Println("test timeout")
	ctx5, _ := ChAddPackedItem.AddTimeoutItem(dot.Item{"id": "005", "sleep": 6}, 5*time.Second)
	<-ctx5.Done()
	assert.Equal(ctx5.Err(), context.DeadlineExceeded)
	assert.Equal(ctx5.Value("test-id"), nil)
}
