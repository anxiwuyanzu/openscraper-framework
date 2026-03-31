# OTel Go SDK Wrapper - Metrics

### `import "github.com/anxiwuyanzu/openscraper-framework/spider-common-go/v4/exp/watchman/metrics"`

## 介绍
Metrics主要用于记录系统中一段时间内某个指标的数值(比如成功率), 只支持`Gauge`且为`Int64`类型的数据(Prometheus的限制).
```
注意: 由于OTel协议规范, Gauge为异步的计数器, 这就意味着必须要传入一个callback function. OTel每次Export的时候会调用这个callback.
     所以, 在callback中记录(更新)指标的值是必须的. 而为了开发者更方便, Wrapper通过传入一个int64的指针来实现引用, 从而解耦SDK和业务代码 
```

## WebUI
[Grafana(使用admin:admin默认账号登录. 可创建自己的账号和组织)](http://10.128.1.187:3000)

## 使用说明([样例文件和详细备注说明请点我](../../../examples/watchman/metrics_test.go))

1. 在爬虫启动前(如main包里的init函数),调用`metrics.Setup()`函数 <u>**启用**</u> 指标记录功能
2. 在需要注册计数器的地方(如engine根据os.Args启动某个爬虫时, 就可以根据爬虫的 Anchor 注册一个计数器), 调用`metrics.RegisterGauge`, 其中:
   - `*ptr`: 用于记录指标数值的变量的指针
   - `unit`: 指标的单位,可选,必须为英文
   - `name`: 指标的名称, 如"success_rate"
   - `attrs`: 指标的属性, 用于识别该指标, 如设置该指标对应的爬虫名称`attribute.String("spider_name", Anchor.String())`
3. 在需要的地方,对上述`*ptr`所指的值进行修改(这样就能同步到Metrics里)
4. 程序结束前调用`metrics.Shutdown()`停止Metrics功能

## OTel Resource 说明[resource.go](../common/resource.go)
**Resource是OTel中对数据的概述的变量, 类似metadata的概念**
- `trace`包只提供了基础的 name/namespace设置, 如需要补充, 请通过`OTEL_RESOURCE_ATTRIBUTES`环境变量设置, 详情见`resource.go`