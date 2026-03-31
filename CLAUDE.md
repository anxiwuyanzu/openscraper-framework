# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**Module:** `github.com/anxiwuyanzu/openscraper-framework/spider-common-go/v4`

A Go shared library for building web crawling/data collection services (数据采集公共库). Provides a middleware-based spider engine with support for multiple HTTP protocols, proxy management, message queues, and data sinks.

**Go version:** 1.22.0 | **Branch:** `v4` | **Releases:** `./tag.sh` (auto-tags `v4.0.{N}` based on commit count since `v4.0.0`)

## Build & Test Commands

```bash
go test ./...                          # Run all tests
go test ./spider/...                   # Run tests in a specific package
go test -v -run TestName ./package/... # Run a single test
```

No Makefile — use standard Go tooling. Many tests are integration tests requiring Redis, Mongo, or Kafka.

CI is GitLab CI (`.gitlab-ci.yml`) — pushes to `v4` branch trigger `tag.sh` to auto-release.

## Architecture

### Core Packages

- **`spider/`** — The spider engine. Key types:
  - `Spider` interface (`app.go`): User implements `Start(ctx)`, `Parse(ctx)`, `OnRetry(ctx)`, `OnFailed(ctx)`, `OnFinished(ctx)`. Embed `Application` for default no-ops.
  - `Engine` (`engine.go`): Top-level entrypoint. `engine.Start("name")`, `engine.StartForever("name")`, `engine.StartSpiderGroup(configs)`.
  - `Factory` (`factory.go`): Configures workers. Holds `SourceFactory` (task producer), `SpiderFactory` (constructs Spider per worker), `WorkerNum`, `MaxRetryTimes`, `Delay`, `BackLog`, `SubSpiders`.
  - `Context` (`context.go`): Per-task context with `Params()`, `Values()`, `Request()`, `Logger()`, status control (`Ok()`, `Fail()`, `Skip()`).
  - `Item` (`item.go`): A crawl task unit. Must implement `Id()`, `Marshal()`, `Unmarshal()`, `Elapsed()`.
  - `Anchor` (`anchor.go`): Typed `string` for spider name/topic. `Anchor.Register(builder)` to register, `Anchor.AddItem(item)` to enqueue.
  - `Handler` = `func(ctx Context)` — middleware chain, call `ctx.Next()` to proceed.

- **`dot/`** — Global singleton service container (`GContext`). All framework services initialized here.
  - Setup: `dot.ConfigViper("DOT", ".")` then `dot.WithRedis()`, `dot.WithMongo()`, `dot.WithSql()`, `dot.WithSinker()`.
  - Accessors: `dot.Conf()`, `dot.Logger()`, `dot.Redis()`, `dot.Mongo()`, `dot.Sql()`, `dot.Sinker()`, `dot.Amqp()`, `dot.Context()`, `dot.Cancel()`.
  - **Never use `viper.Get` directly** — always go through `dot.Conf()`.

- **`reqwest/`** — HTTP client abstraction over multiple backends.
  - Backends: `fasthttp` (default), `net/http`, `tls-client` (browser fingerprint mimicry via `bogdanfinn/tls-client`).
  - HTTP versions: 1 (HTTP/1.1), 2 (HTTP/2), 3 (QUIC/HTTP3 via `quic-go`).
  - `reqwest/proxz/` — Proxy management with providers: `zhima`, `zhima-relay`, `xigua`, `relay`.
  - `reqwest/dnscache/` — DNS caching.
  - Concurrency limiting via `semaphore.Weighted`.

- **`util/`** — Utilities: `atof/` (float parsing), `collection/`, `compress/`, `crypto_utils/`, `log/`, `redis_ext/` (Redis helpers), `serde/` (JSON via `json-iterator`).

- **`exp/watchman/`** — Experimental OpenTelemetry integration (metrics + tracing via OTLP).

- **`internal/tests/`** — Shared test infrastructure (TLS certs, test HTTP server).

### Data Flow

```
proc/timer → MQ → Spider → Kafka/DB
```

Tasks are produced by `SourceFactory`, flow through `ItemCh`, processed by workers running the middleware chain (`PreCheck → PreStart → Start → [HTTP request] → Parse`), and results are sent to sinks (Kafka, DB).

### Configuration

Config loaded from `config.yaml` via Viper. Environment overrides use `DOT_` prefix (e.g., `DOT_PROXY_PROXY`). Key sections: `spider`, `reqwest`, `proxy`, `kafka`, `redis`, `mongo`, `sql`, `amqp`.

### ISinker Interface

Abstracts data output: `Sink()`, `SinkString()`, `SinkBytes()`. Backed by Kafka in production, `DebugSinker` (stdout) in debug mode.

## Key Dependencies

| Package | Purpose |
|---|---|
| `valyala/fasthttp` | Default HTTP client |
| `bogdanfinn/tls-client` | Browser-fingerprint TLS |
| `quic-go/quic-go` | HTTP3/QUIC |
| `go-redis/redis/v8` | Redis |
| `mongo-driver` | MongoDB |
| `jmoiron/sqlx` | SQL (MySQL) |
| `confluent-kafka-go/v2` | Kafka producer |
| `streadway/amqp` | RabbitMQ |
| `sirupsen/logrus` | Logging |
| `spf13/viper` | Configuration |
| `json-iterator/go` | Fast JSON (`util/serde`) |
| `tidwall/gjson` + `sjson` | JSON path query/mutation |

## Coding Style Guide (编码规范)

以下规范从项目现有代码中提炼而来，所有新增/修改代码应遵循。

### 命名规范

- **包名**: 短小、全小写、单词，允许缩写（`proxz`, `reqwest`, `serde`, `atof`, `dnscache`）
- **接口**: 核心服务接口用 `I` 前缀（`IClient`, `ISinker`, `IRequest`）；小型行为接口不加前缀（`ProxyFetcher`, `ProxyPoolManager`）
- **配置结构体**: 以 `Config` 结尾（`SpiderConfig`, `RedisConfig`, `KafkaConfig`）
- **未导出实现**: camelCase，如 `contextImpl`
- **错误哨兵**: `Err` 前缀 + 小写描述，放在 `var` 块中（`ErrNoProxy`, `ErrEmptyBody`）
- **常量**: 导出用 PascalCase，未导出用 camelCase，枚举用 `iota`
- **类型别名**: 用于增加编译期安全性（`type Anchor string`, `type StatusCode int`, `type Handler func(ctx Context)`）
- **构造函数**: `New*` 命名（`NewEngine()`, `NewClient()`）；池化资源用 `Acquire*/Release*`（`AcquireContext`, `ReleaseProxy`）

### Import 组织

标准库在前，第三方在后，空行分隔。内部包与第三方归为一组：

```go
import (
    "context"
    "sync"
    "time"

    "github.com/anxiwuyanzu/openscraper-framework/spider-common-go/v4/dot"
    "github.com/sirupsen/logrus"
)
```

副作用导入用 `_`：`_ "github.com/anxiwuyanzu/openscraper-framework/.../providers"`。包名冲突时用别名：`log "github.com/sirupsen/logrus"`。

### 注释规范

- **中英文混用**: 业务领域和框架 API 的 godoc 用**中文**，实现细节的行内注释可用中文或英文
- 所有导出标识符必须有 godoc 注释
- 行内注释解释"为什么"，不解释"是什么"

```go
// Engine 负责启动爬虫
type Engine struct { ... }

// 提前为子爬虫创建 itemCh; 避免 itemCh 不存在
createItemCh(spiderName, subFactory.BackLog)
```

### 错误处理

- 错误立即检查并处理，不忽略
- 框架初始化阶段（配置缺失、连接失败）允许 `panic`，业务代码禁止
- Spider 上下文中通过 `ctx.Ok()` / `ctx.Fail(err)` / `ctx.Skip(err)` 传递状态，不用返回值
- 可选错误参数用变参：`func Fail(errs ...error)`

### 并发模式

- goroutine 用闭包启动，**捕获变量通过参数传递**避免竞态：
  ```go
  go func(f *Factory, name Anchor) {
      defer wg.Done()
      f.Start(ctx, name, logName, false)
  }(subFactory, spiderName)
  ```
- `sync.WaitGroup` 用于 fan-out worker 池，`wg.Add(1)` 在 `go` 之前，`defer wg.Done()` 在 goroutine 内部
- `sync.RWMutex` 直接嵌入结构体（不命名），`defer Unlock()` / `defer RUnlock()`
- 简单计数器用 `sync/atomic`
- 取消信号通过 `select` + `context.Done()` / `dot.Context().Done()` 传递

### 配置规范

- **永远不要直接调用 `viper.Get*`**，通过 `dot.Conf()` 获取类型安全的配置
- 配置结构体使用 `yaml` + `default` 标签：
  ```go
  type SpiderConfig struct {
      WorkerNum int `yaml:"worker_num" default:"1"`
  }
  ```
- 环境变量覆盖使用 `DOT_` 前缀（`DOT_PROXY_PROXY`）

### 日志规范

- 统一使用 `logrus`，通过 `dot.Logger()` 获取全局 logger
- 结构化日志用 `WithField` / `WithFields`，错误用 `WithError`
- 不可恢复的初始化错误用 `Panic`，运行时用 `Error` / `Warn`
- 每个 worker 创建子 logger 携带 spider 名称字段

### 接口设计

- 接口小而聚焦，**在消费方定义**而非实现方
- 函数类型当接口用（`Handler`, `Builder`, `PoolBuilder`），允许直接传递函数
- `Context` 接口例外——作为框架核心允许方法数较多

### 文件组织

- 每个文件单一职责：`engine.go`（启动停止）、`factory.go`（worker 循环）、`context.go`（上下文）、`types.go`（常量和类型定义）
- 全局注册表独立成文件（`register.go`）
- 平台相关代码用构建约束：`sink_unix.go` / `sink_windows.go`
- 包级 godoc 放在 `doc.go`

### 测试规范

- 标准 `_test.go` 命名，函数用 `Test` 前缀 PascalCase
- 以集成测试为主（需要真实 Redis/Mongo/Kafka），不使用 mock 框架
- 共享测试设施放在 `internal/tests/`（TLS 证书、测试 HTTP 服务器）
- 断言使用 `stretchr/testify`

### JSON 使用

- 序列化/反序列化统一用 `util/serde`（底层 `json-iterator`），不直接用 `encoding/json`
- JSON 路径查询用 `tidwall/gjson`，路径修改用 `tidwall/sjson`
- 高性能场景用 `valyala/fastjson`（通过 `serde.AcquireFastjsonParser()` 池化）

### 泛型使用

- 谨慎使用，仅在有明确收益时（如 `util.Itoa[T]`, `util.If[T]`）
- 核心引擎层（spider、context、request）使用接口和 `any` 类型断言，不用泛型
