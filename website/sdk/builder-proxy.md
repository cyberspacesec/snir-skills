# 代理构建器

<p align="center">🔀 SDK 配置代理与轮换。</p>

## 选项

| 选项 | 说明 |
|------|------|
| `WithProxy(proxy)` | 单代理 |
| `WithProxyList(strategy, proxies...)` | 代理列表 |
| `WithProxyFile(path, strategy)` | 代理文件（热加载） |
| `WithProxyURL(url, strategy)` | 动态代理 API |
| `WithProxyStrategy(strategy)` | 轮换策略 |

策略类型 `runner.ProxyStrategy`：`round-robin`/`random`/`sequential`。

::: info sequential = 自动故障转移
代理不稳定时选 `ProxyStrategySequential`——当前代理失败**自动切下一个**，配合 `WithMaxRetries` 重试最大化成功率。`round-robin`/`random` 则不论成败轮换。
:::

## 示例

```go
// 单代理
opts := sdk.NewScreenshotOptions(
    sdk.WithProxy("http://127.0.0.1:8080"),
)

// 列表轮换
opts := sdk.NewScreenshotOptions(
    sdk.WithProxyList(runner.ProxyStrategyRoundRobin,
        "http://p1:8080", "http://p2:8080", "http://p3:8080",
    ),
)

// 文件
opts := sdk.NewScreenshotOptions(
    sdk.WithProxyFile("proxies.txt", runner.ProxyStrategyRandom),
)

// 动态 API
opts := sdk.NewScreenshotOptions(
    sdk.WithProxyURL("http://proxy-service/api", runner.ProxyStrategyRandom),
)
```

## 选择建议

| 场景 | 选 |
|------|-----|
| 单出口 | `WithProxy` |
| 少量固定 | `WithProxyList` |
| 大量、需更新 | `WithProxyFile` |
| 商业动态池 | `WithProxyURL` |

四种代理来源与轮换策略如何配合：

```mermaid
flowchart LR
    subgraph 来源
        S1[WithProxy<br/>单代理]
        S2[WithProxyList<br/>列表]
        S3[WithProxyFile<br/>文件热加载]
        S4[WithProxyURL<br/>动态 API]
    end
    S1 --> Pool[代理池]
    S2 --> Pool
    S3 --> Pool
    S4 --> Pool
    Pool --> Strat{轮换策略}
    Strat --> |round-robin| R1[轮询]
    Strat --> |random| R2[随机]
    Strat --> |sequential| R3[顺序]
    R1 & R2 & R3 --> Pick[为本次请求选代理]
    Pick --> Req[Chrome 经代理访问目标]

    style Pool fill:#3aa676,stroke:#2a7a56,color:#fff
    style Req fill:#e6f4ea,stroke:#3aa676
```

## sequential 策略状态机

`sequential` 策略的核心是"当前代理失败自动切下一个"，状态流转如下：

```mermaid
stateDiagram-v2
    [*] --> 选中当前代理
    选中当前代理 --> 请求中: Chrome 经代理访问目标
    请求中 --> 成功: 200/截图完成
    请求中 --> 失败: 超时/连接错误/代理失效
    成功 --> [*]: 返回 Result
    失败 --> 重试判断: WithMaxRetries
    重试判断 --> 选中当前代理: 重试次数未用尽(同代理)
    重试判断 --> 切换下一代理: 重试用尽/代理失效
    切换下一代理 --> 选中当前代理: 新代理就位
    切换下一代理 --> [*]: 代理列表耗尽,返回错误
```

`round-robin`/`random` 不走"失败切换"分支——不论成败按策略轮换，每次请求都换代理。

## 下一步

- [构建器总览](./builders)
- [代理与轮换（进阶）](../advanced/proxy)
- [内部 pkg/runner/proxy](../internals/runner-proxy)
