# 远程 Chrome

<p align="center">🌐 用远程 CDP 端点，免本地浏览器。</p>

无本地 Chrome 或需多进程复用时，连接远程 Chrome DevTools Protocol 端点。

## 连接

```bash
snir scan example.com --wss ws://host:9222/devtools/browser/<id>
```

`--wss` 指向远程 Chrome 的 WebSocket 调试 URL。

## 部署远程 Chrome

在服务器上启动 headless Chrome：

```bash
chromium --headless --remote-debugging-port=9222 --remote-debugging-address=0.0.0.0 \
  --no-sandbox --disable-gpu
```

获取 `webSocketDebuggerUrl`：

```bash
curl http://host:9222/json/version
```

## 共享 Provider

`snir provider` 启动常驻 Chrome 供多进程复用：

```bash
# 服务端
snir provider --port 9223 --max-concurrent 20

# 各 worker
snir scan example.com --wss ws://provider-host:9222/devtools/browser/<id>
```

见 [provider 命令](../cli/provider)。

## SDK
```go
client, _ := sdk.NewRemoteClient("ws://host:9222/devtools/browser/xxx", 10)
defer client.Close()
```

或自动连接：

```go
client, mode, _ := sdk.AutoConnectClient(opts)  // opts.WSSURL 指定远程
```

见 [自动连接](../sdk/autoconnect)。

## 决策

```mermaid
flowchart TD
  Q{有本地 Chrome?}
  Q -- 有 --> L[本地]
  Q -- 无 --> R{需多进程共享?}
  R -- 是 --> P[snir provider]
  R -- 否 --> W[直接 --wss 远程]
```

客户端经 `--wss` 连接远程 Chrome 完成一次截图的时序：

```mermaid
sequenceDiagram
    autonumber
    participant Cli as snir 客户端
    participant RCDP as 远程 Chrome CDP
    participant Page as 目标页面
    Cli->>RCDP: WebSocket 连接 ws://host:9222/devtools/browser/<id>
    RCDP-->>Cli: 连接建立
    Cli->>RCDP: Target.createTarget 开新标签页
    RCDP-->>Cli: 返回 targetId
    Cli->>Page: Page.navigate 目标 URL
    Page-->>RCDP: 触发加载事件
    RCDP-->>Cli: Page.loadEventFired
    Cli->>RCDP: Page.captureScreenshot
    RCDP-->>Cli: 返回截图 base64
    Cli->>RCDP: Target.closeTarget 关闭标签页
    Cli->>Cli: 写入产物并断开会话
```

## 安全

::: danger 远程调试端口暴露=完全控制
Chrome 远程调试端口等价于"能在该 Chrome 里执行任意操作"。**切勿暴露公网**：
- 限内网访问，或加网络层鉴权
- `--remote-debugging-address` 默认 `127.0.0.1`，开放 `0.0.0.0` 需极度谨慎
- Provider/远程 Chrome 应部署在受控网络
:::

## 下一步

- [provider 命令](../cli/provider)
- [Chrome 选项](../cli/scan-chrome)
- [自动连接](../sdk/autoconnect)
- [并发与池](./concurrency)
