# 选项构建器

<p align="center">🏗️ `pkg/sdk/builders.go` — 链式 With* 函数。</p>

`ScreenshotOption` 是函数式选项，`With*` 函数返回它，组合传入 `Capture`。

> 📁 源码：[`pkg/sdk/builders.go`](https://github.com/cyberspacesec/snir-skills/blob/main/pkg/sdk/builders.go)

## 常用构建器

| 函数 | 说明 |
|------|------|
| `WithFullPage()` | 整页截图 |
| `WithViewport(w, h)` | 视口 |
| `WithHTML()` | 采 HTML |
| `WithConsoleLogs()` | 采 Console |
| `WithHAR()` | 采网络 HAR |
| `WithCookies()` | 采 Cookies |
| `WithTimeout(d)` | 超时 |
| `WithDelay(d)` | 额外等待 |
| `WithProxy(p)` | 代理 |
| `WithUserAgent(ua)` | UA |
| `WithDevice(name)` | 设备预设 |
| `WithFormat(fmt, q)` | 输出格式 |
| `WithOutputPath(p)` | 输出目录 |
| `WithBlacklist(enabled)` | 黑名单开关 |

> 完整列表见源码 [`builders.go`](https://github.com/cyberspacesec/snir-skills/blob/main/pkg/sdk/builders.go)。

## 链式组合

```mermaid
flowchart LR
  W1[WithFullPage] --> OPT
  W2[WithHTML] --> OPT
  W3[WithConsoleLogs] --> OPT
  OPT[[]ScreenshotOption] --> CAP[Capture]
  CAP --> R[*Result]
```

## 工作原理

::: details 函数式选项，类型安全可组合
每个 `With*` 修改 `*ScreenshotOptions`：

```go
type ScreenshotOption func(*ScreenshotOptions)

func WithFullPage() ScreenshotOption {
    return func(o *ScreenshotOptions) { o.FullPage = true }
}
```

`Capture` 先建默认 `ScreenshotOptions`，依次应用各 option，再 [`mergeWithScreenshotOptions`](https://github.com/cyberspacesec/snir-skills/blob/main/pkg/sdk/options.go#L326) 合并到基线。

这种"函数式选项"模式的好处：零值即默认、可任意组合、新增选项不破坏旧调用方。
:::

## 示例

```go
res, _ := client.Capture("https://example.com",
    sdk.WithFullPage(),
    sdk.WithViewport(1920, 1080),
    sdk.WithHTML(),
    sdk.WithConsoleLogs(),
    sdk.WithDelay(2*time.Second),
    sdk.WithProxy("http://proxy:8080"),
)
```

## 子专题

各专项构建器有单独详解：[截图](./builder-screenshot)、[视口](./builder-viewport)、[代理](./builder-proxy)、[Cookie](./builder-cookie)、[指纹](./builder-fingerprint)、[JS](./builder-js)、[表单](./builder-form)、[黑名单](./builder-blacklist)、[端口](./builder-ports)。

## 下一步

- [Client](./client)
- [ClientOptions](./client-options)
- [builders.go 源码](https://github.com/cyberspacesec/snir-skills/blob/main/pkg/sdk/builders.go)
- [五分钟上手](../guide/five-minutes)
