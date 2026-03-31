package dot

import (
	"context"
	"database/sql"
	"github.com/go-redis/redis/v8"
	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/mongo"
)

var (
	std = NewContext(true, logrus.WithFields(nil))
)

func InitFromViper()                       { std.InitFromViper() }
func Cancel()                              { std.cancel() }
func Context() context.Context             { return std.ctx }
func Conf() *Config                        { return std.Conf() }
func Sinker() ISinker                      { return std.sinker }
func SinkerOthers(key string) ISinker      { return std.SinkerOthers(key) }
func Sql() *sql.DB                         { return std.sqlClient }
func Sqlx() *sqlx.DB                       { return std.sqlxClient }
func SqlOthers(key string) *sql.DB         { return std.SqlOthers(key) }
func SqlxOthers(key string) *sqlx.DB       { return std.SqlxOthers(key) }
func Debug() bool                          { return std.debug }
func Logger() *logrus.Entry                { return std.logger }
func Mongo() *mongo.Client                 { return std.mongoClient }
func Redis() *redis.Client                 { return std.redisClient }
func RedisOthers(key string) *redis.Client { return std.RedisOthers(key) }
func Amqp() *AmqpClient                    { return std.amqpClient }

func WithContext(ctx context.Context) { std.WithContext(ctx) }
func WithSql(addrs ...string)         { std.WithSql(addrs...) }
func WithSqlOthers(key string)        { std.WithSqlOthers(key) }
func WithMongo(addrs ...string)       { std.WithMongo(addrs...) }
func WithRedis(addrs ...string)       { std.WithRedis(addrs...) }
func WithRedisOthers(key string)      { std.WithRedisOthers(key) }
func WithAmqp(urls ...string)         { std.WithAmqp(urls...) }
func WithSinker(sinker ...ISinker)    { std.WithSinker(sinker...) }

func WithSinkerOthers(key string, sinker ...ISinker) {
	std.WithSinkerOthers(key, sinker...)
}

func WithLogger(logger *logrus.Entry)              { std.WithLogger(logger) }
func WithValue(key string, value any)              { std.WithValue(key, value) }
func WithImmutableValue(key string, value any) any { return std.WithImmutableValue(key, value) }
func Value(key string) any                         { return std.Value(key) }
func Has(key string) bool                          { return std.Has(key) }

func EnsureClose(cb Handler)         { std.EnsureClose(cb) }
func EnsureCloseHandlers() []Handler { return std.EnsureCloseHandlers() }
