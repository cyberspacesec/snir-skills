# SDK 总览

<p align="center">🧩 `pkg/sdk` — 把 snir 能力暴露为 Go API。</p>

Go SDK 让你把网页截图/情报采集嵌入自己的 Go 程序，链式配置、类型安全、自带连接池与黑名单。

> 📁 源码目录：[`pkg/sdk/`](https://github.com/cyberspacesec/snir-skills/blob/main/pkg/sdk)

## 两种风格

::: tip 选哪个？看场景
| 风格 | 入口 | 适合 |
|------|------|------|
| **链式 `Capture`** | 可变参数 `With*` | 灵活组合、可选参数多、写起来省 |
| **结构体 `Screenshot`** | 传 `*ScreenshotOptions` | 显式完整、配置来自外部反序列化/配置文件 |

两者底层走同一套流水线，结果一致。不确定就先用 `Capture`。
:::

```mermaid
flowchart TD
  U[你的代码] --> CH{风格?}
  CH -- 链式 Option --> CA[Capture(url, With*...)]
  CH -- 结构体 --> SS[Screenshot(url, &ScreenshotOptions{})]
  CA --> RES[*models.Result]
  SS --> RES
```

## 核心入口

| 入口 | 源码 | 说明 |
|------|------|------|
| `Client` | [client.go#L54](https://github.com/cyberspacesec/snir-skills/blob/main/pkg/sdk/client.go#L54) | 长生命周期客户端 |
| `NewClient(opts)` | [client.go#L78](https://github.com/cyberspacesec/snir-skills/blob/main/pkg/sdk/client.go#L78) | 构造 |
| `NewRemoteClient(wsURL, max)` | [client.go#L113](https://github.com/cyberspacesec/snir-skills/blob/main/pkg/sdk/client.go#L113) | 远程 Chrome |
| `Shared*` 函数 | [shared.go](https://github.com/cyberspacesec/snir-skills/blob/main/pkg/sdk/shared.go) | 无需管理 Client |

## 文件结构

| 文件 | 源码 | 内容 |
|------|------|------|
| `client.go` | [→](https://github.com/cyberspacesec/snir-skills/blob/main/pkg/sdk/client.go) | Client 与所有方法 |
| `options.go` | [→](https://github.com/cyberspacesec/snir-skills/blob/main/pkg/sdk/options.go) | ClientOptions/ScreenshotOptions |
| `builders.go` | [→](https://github.com/cyberspacesec/snir-skills/blob/main/pkg/sdk/builders.go) | With* 链式函数 |
| `result.go` | [→](https://github.com/cyberspacesec/snir-skills/blob/main/pkg/sdk/result.go) | 结果包装 |
| `shared.go` | [→](https://github.com/cyberspacesec/snir-skills/blob/main/pkg/sdk/shared.go) | 共享池便捷函数 |
| `targets.go` | [→](https://github.com/cyberspacesec/snir-skills/blob/main/pkg/sdk/targets.go) | 目标展开 |
| `autoconnect.go` | [→](https://github.com/cyberspacesec/snir-skills/blob/main/pkg/sdk/autoconnect.go) | 自动连接 |

## 最小示例

```go
client, _ := sdk.NewClient(sdk.DefaultClientOptions())
defer client.Close()

result, _ := client.Capture("https://example.com",
    sdk.WithFullPage(),
    sdk.WithHTML(),
    sdk.WithConsoleLogs(),
)
```

更多见 [五分钟上手](../guide/five-minutes) 与 [SDK 安装](./installation)。

## 内部视角

实现细节见 [pkg/sdk（内部）](../internals/sdk)。

## 下一步

- [Client](./client)
- [ClientOptions](./client-options)
- [构建器](./builders)
- [结果与证据](./result)
- [共享池](./shared)
