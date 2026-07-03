# pkg/provider

<p align="center">🔌 跨进程共享 Chrome/CDP Provider。</p>

`pkg/provider` 实现常驻 Chrome 端点，供多进程 worker 复用。

## 核心类型

### ProviderOptions

| 字段 | 说明 |
|------|------|
| `ChromePath` | Chrome 路径 |
| `Headless` | 无头模式 |
| `WindowWidth` / `WindowHeight` | 窗口尺寸 |
| `UserAgent` | UA |
| `Proxy` | 代理 |
| `IgnoreCertErrors` | 忽略证书 |
| Provider 服务配置 | 端口/并发/空闲超时等 |

`DefaultProviderOptions()` 给默认值。

### Provider

```go
func NewProvider(opts ProviderOptions) *Provider
func WaitForSignal(p *Provider)
func DiscoverChrome(host string, ports []int) (string, error)
```

## 启动

```bash
snir provider --port 9223 --max-concurrent 20
```

各 worker 用 `--wss` 连接其暴露的 Chrome。见 [provider 命令](../cli/provider)。

## 与 DriverPool 的区别

- `DriverPool`：进程内多任务复用
- `Provider`：跨进程共享同一 Chrome 端点

两者的作用域对比：

```mermaid
flowchart TB
    subgraph 进程A [进程 A]
        P1[DriverPool<br/>进程内复用]
        T1[任务1] --> P1
        T2[任务2] --> P1
    end
    subgraph 进程B [进程 B]
        P2[DriverPool<br/>进程内复用]
        T3[任务3] --> P2
    end
    Prov[Provider<br/>常驻 Chrome 端点 :9223]
    P1 -- --wss --> Prov
    P2 -- --wss --> Prov
    Prov --> Chrome[(共享 Chrome/CDP)]

    style Prov fill:#3aa676,stroke:#2a7a56,color:#fff
    style Chrome fill:#e6f4ea,stroke:#3aa676
```

::: tip 💡 何时用 Provider
单进程批量用 `DriverPool`（SDK `Shared*` 自动管理）；多进程/多机 worker 共用一个 Chrome 实例时启动 `snir provider`，各 worker 以 `--wss` 接入，省去每进程各起一份 Chrome 的开销。
:::

## 下一步

- [provider 命令](../cli/provider)
- [远程 Chrome](../advanced/remote-chrome)
- [并发与池](../advanced/concurrency)
