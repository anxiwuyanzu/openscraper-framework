
## 代理配置
协议头包括 http 和 socks

### 基本类型
协议头支持 `http` 和 `socks`

- http://127.0.0.1:8889
- socks://127.0.0.1:8889
- http://username:pwd@127.0.0.1:8889

### 提供商类型
这类代理 host 代表提供商, 提供商支持:
```
  ProviderZhiMa        = "zhima"
  ProviderZhiMaRelay   = "zhima-relay"
  ProviderXiGua        = "xigua"
  ProviderRelay        = "relay"
```

不同的代理商有不同的配置参数: 如
- http://zhima?chan=zl&topic=default&size=100
- socks://zhima?chan=3m&topic=default&size=100&provider=fly,qingguo
- http://relay?platform=mix
- http://xigua?ipv6=1

公用配置:
- `size` 一次获取多少ip
- `drop_in` 在过期前 {drop_in} 秒释放代理, 默认 60.
- `filter_in` 如果上次使用这个代理小于 filter_in 秒，会过滤这个代理， 默认0，即不启用
- `timeout` 设置获取ip超时(秒), 默认 15

#### zhima & zhima-relay 参数
芝麻协议头支持 `http` 和 `socks`

服务端代码: https://github.com/anxiwuyanzu/openscraper-framework/proxy-relay/-/tree/v2

- `chan` 芝麻通道, 支持 3m, 10m
- `provider` 目前支持 `qingguo`, `fly`, 默认为 `qingguo,fly`, fly为必选, 可以用`fly,qingguo`, 让 `fly` 成为优先选择
- `topic` zhima-relay 通过 topic 和其他爬虫公用代理 [详见](https://github.com/anxiwuyanzu/openscraper-framework/proxy-relay/-/tree/v2)
- `city` 设置城市, 可以是省, 也可以是市
  - 默认留空, 表示没有city限制; 
  - 可以设置为固定的 cityCode, 详见 zhiMaCities; 比如 `350000` (福建), `340000` (安徽), cityCode 由芝麻提供
  - 也可以设置为 `random`, 表示为Client随机一个固定的city, 以后都使用这个city

