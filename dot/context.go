package dot

import (
	"context"
	"database/sql"
	"github.com/jmoiron/sqlx"
	"sync"

	"github.com/go-redis/redis/v8"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/mongo"
)

type Handler func()

type GContext struct {
	sync.RWMutex
	debug  bool
	ctx    context.Context
	cancel context.CancelFunc
	config *Config

	sinker ISinker
	// sinkerOthers 方便配置多个sinker, 同时配合others配置
	sinkerOthers map[string]ISinker

	sqlClient        *sql.DB
	sqlxClient       *sqlx.DB
	sqlClientOthers  map[string]*sql.DB
	sqlxClientOthers map[string]*sqlx.DB

	mongoClient         *mongo.Client
	redisClient         *redis.Client
	redisClientOthers   map[string]*redis.Client
	amqpClient          *AmqpClient
	logger              *logrus.Entry
	values              map[string]any
	ensureCloseHandlers []Handler
}

func NewContext(debug bool, logger *logrus.Entry) *GContext {
	innerCtx, cancel := context.WithCancel(context.Background())
	ctx := &GContext{
		ctx:               innerCtx,
		cancel:            cancel,
		debug:             debug,
		logger:            logger,
		values:            make(map[string]any, 10),
		sinkerOthers:      make(map[string]ISinker),
		sqlClientOthers:   make(map[string]*sql.DB),
		sqlxClientOthers:  make(map[string]*sqlx.DB),
		redisClientOthers: make(map[string]*redis.Client),
	}
	return ctx
}

func (c *GContext) Cancel()                  { c.cancel() }
func (c *GContext) Context() context.Context { return c.ctx }
func (c *GContext) Sinker() ISinker          { return c.sinker }
func (c *GContext) SinkerOthers(key string) ISinker {
	c.RLock()
	defer c.RUnlock()
	return c.sinkerOthers[key]
}
func (c *GContext) Sql() *sql.DB { return c.sqlClient }
func (c *GContext) SqlOthers(key string) *sql.DB {
	c.RLock()
	defer c.RUnlock()
	return c.sqlClientOthers[key]
}
func (c *GContext) SqlxOthers(key string) *sqlx.DB {
	c.RLock()
	defer c.RUnlock()
	return c.sqlxClientOthers[key]
}
func (c *GContext) Debug() bool           { return c.debug }
func (c *GContext) Logger() *logrus.Entry { return c.logger }
func (c *GContext) Mongo() *mongo.Client  { return c.mongoClient }
func (c *GContext) Redis() *redis.Client  { return c.redisClient }
func (c *GContext) RedisOthers(key string) *redis.Client {
	c.RLock()
	defer c.RUnlock()
	return c.redisClientOthers[key]
}
func (c *GContext) Amqp() *AmqpClient { return c.amqpClient }

func (c *GContext) Conf() *Config {
	if c.config == nil {
		c.InitFromViper()
	}
	return c.config
}

func (c *GContext) WithContext(ctx context.Context) {
	c.ctx = ctx
}

func (c *GContext) WithSql(addrs ...string) {
	if c.sqlClient == nil {
		var addr string
		if len(addrs) == 0 {
			addr = c.config.Sql.Uri
		} else {
			addr = addrs[0]
		}

		c.sqlxClient = NewSqlxClientWithConfig(addr, &c.config.Sql)
		c.sqlClient = c.sqlxClient.DB
	}
}

func (c *GContext) WithSqlOthers(key string) {
	if len(key) == 0 {
		return
	}
	c.Lock()
	defer c.Unlock()

	if _, ok := c.sqlClientOthers[key]; ok {
		return
	}

	db := NewSqlxClientWithConfig(key, &c.config.Sql)
	c.sqlClientOthers[key] = db.DB
	c.sqlxClientOthers[key] = db
}

func (c *GContext) WithMongo(addrs ...string) {
	if c.mongoClient == nil {
		var addr string
		if len(addrs) == 0 {
			addr = c.config.Mongo.Uri
		} else {
			addr = addrs[0]
		}
		c.mongoClient = NewMongoClient(addr)
	}
}

func (c *GContext) WithRedis(addrs ...string) {
	if c.redisClient == nil {
		var addr string
		if len(addrs) == 0 {
			addr = c.config.Redis.Uri
		} else {
			addr = addrs[0]
		}

		c.redisClient = NewRedisClient(addr)
	}
}

func (c *GContext) WithRedisOthers(key string) {
	if len(key) == 0 {
		return
	}
	c.Lock()
	defer c.Unlock()

	if _, ok := c.redisClientOthers[key]; ok {
		return
	}

	c.redisClientOthers[key] = NewRedisClient(key)
}

func (c *GContext) WithAmqp(urls ...string) {
	if c.amqpClient == nil {
		var url string
		if len(urls) == 0 {
			url = c.config.Amqp.Uri
		} else {
			url = urls[0]
		}
		c.amqpClient = NewAmqpClient(url)
	}
}

func (c *GContext) WithSinker(sinker ...ISinker) {
	if len(sinker) > 0 {
		c.sinker = sinker[0]
		return
	}
	if c.sinker == nil {
		c.sinker = NewSinker(c.config.Kafka.BootstrapServers)
	}
}

func (c *GContext) WithSinkerOthers(key string, sinker ...ISinker) {
	if len(key) == 0 {
		return
	}
	c.Lock()
	defer c.Unlock()
	if len(sinker) > 0 {
		c.sinkerOthers[key] = sinker[0]
		return
	}

	if _, ok := c.sinkerOthers[key]; !ok {
		c.sinkerOthers[key] = NewSinker(key)
	}
}

func (c *GContext) WithLogger(logger *logrus.Entry) {
	c.logger = logger
}

// WithValue 用 WithValue 存储数据
func (c *GContext) WithValue(key string, value any) {
	c.Lock()
	defer c.Unlock()
	c.values[key] = value
}

// WithImmutableValue 存储数据, 如果存在, 返回旧数据
func (c *GContext) WithImmutableValue(key string, value any) any {
	c.Lock()
	defer c.Unlock()
	if v, ok := c.values[key]; ok {
		return v
	}
	c.values[key] = value
	return value
}

func (c *GContext) Value(key string) any {
	c.RLock()
	defer c.RUnlock()
	return c.values[key]
}

func (c *GContext) Has(key string) bool {
	c.RLock()
	defer c.RUnlock()
	_, ok := c.values[key]
	return ok
}

// EnsureClose 在程序结束是调用callback
func (c *GContext) EnsureClose(cb Handler) {
	c.ensureCloseHandlers = append(c.ensureCloseHandlers, cb)
}

// EnsureCloseHandlers 返回 CloseHandlers
func (c *GContext) EnsureCloseHandlers() []Handler {
	return c.ensureCloseHandlers
}
