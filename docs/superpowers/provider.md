# snir provider — CDP Provider（跨进程共享 Chrome）

> **渐进式披露**：[快速上手](#快速上手) → [常用选项](#常用选项) → [高级选项](#高级选项) → [完整参数参考](#完整参数参考)

---

## 快速上手

```bash
# 启动 Provider（默认端口 9223）
./snir provider

# 指定端口和 Chrome 调试端口
./snir provider --port 9223 --chrome-port 9222
```

---

## 工作原理

Provider 是一个 CDP 代理服务，它管理 Chrome 浏览器进程，并将 WebSocket 连接暴露给其他工具：

```
其他工具 (snir api / Nuclei / 自定义脚本)
    ↓ HTTP: 查询 WebSocket URL
    ↓ WS:   直接连接 Chrome 实例
Provider (端口 9223)
    ↓ 管理
Chrome 浏览器进程 (端口 9222)
```

**核心价值：**
- 多个独立进程/工具共享同一个 Chrome 实例，节省内存
- 自动管理 Chrome 生命周期（启动/空闲关闭/崩溃重启）
- 提供 HTTP API 供其他工具发现和连接

---

## 常用选项

### 端口配置

```bash
# Provider 监听端口（其他工具连接此端口查询信息）
./snir provider --port 9223

# Chrome 远程调试端口（Provider 管理的 Chrome 实例）
./snir provider --chrome-port 9222
```

| 标志 | 默认值 | 说明 |
|------|--------|------|
| `--port` | `9223` | Provider 服务监听端口 |
| `--chrome-port` | `9222` | Chrome 远程调试端口 |

### 并发控制

```bash
# 最大并发截图数（超出则排队等待）
./snir provider --max-concurrent 20
```

| 标志 | 默认值 | 说明 |
|------|--------|------|
| `--max-concurrent` | `10` | 最大并发截图数 |

---

## 高级选项

### 空闲超时

```bash
# 5 分钟不活动自动关闭浏览器（下次截图自动重启）
./snir provider --idle-timeout 5m

# 30 秒超时
./snir provider --idle-timeout 30s

# 不自动关闭（默认行为，浏览器一直运行）
./snir provider --idle-timeout 0
```

| 标志 | 默认值 | 说明 |
|------|--------|------|
| `--idle-timeout` | `0` | 浏览器空闲超时（如 `5m`、`30s`，`0` = 不自动关闭） |

### Chrome 配置

```bash
# 自定义 Chrome 路径
./snir provider --chrome-path /opt/google/chrome/chrome

# 自定义 User-Agent
./snir provider --user-agent "Mozilla/5.0 Custom Agent"

# 使用代理
./snir provider --proxy http://127.0.0.1:8080
```

| 标志 | 默认值 | 说明 |
|------|--------|------|
| `--chrome-path` | `""` | Chrome 可执行文件路径 |
| `--user-agent` | `""` | 自定义 User-Agent |
| `--proxy` | `""` | 代理服务器地址 |

### 浏览器模式

```bash
# 非无头模式（显示浏览器界面，调试用）
./snir provider --headless=false

# 忽略证书错误
./snir provider --ignore-cert-errors
```

| 标志 | 默认值 | 说明 |
|------|--------|------|
| `--headless` | `true` | 使用无头模式（`--headless=false` 显示界面） |
| `--ignore-cert-errors` | `false` | 忽略证书错误 |

---

## 完整参数参考

| 标志 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `--port` | int | `9223` | Provider 服务监听端口 |
| `--chrome-port` | int | `9222` | Chrome 远程调试端口 |
| `--max-concurrent` | int | `10` | 最大并发截图数 |
| `--idle-timeout` | duration | `0` | 浏览器空闲超时（`5m`、`30s`，`0` = 不关闭） |
| `--chrome-path` | string | `""` | Chrome 可执行文件路径 |
| `--user-agent` | string | `""` | 自定义 User-Agent |
| `--proxy` | string | `""` | 代理服务器地址 |
| `--headless` | bool | `true` | 使用无头模式 |
| `--ignore-cert-errors` | bool | `false` | 忽略证书错误 |

---

## Provider HTTP API

| 端点 | 方法 | 说明 |
|------|------|------|
| `/` | GET | Provider 信息 |
| `/ws` | GET | 获取 WebSocket URL |
| `/health` | GET | 健康检查 |
| `/stats` | GET | 连接池统计 |
| `/screenshot?url=` | POST | 直接截图 |

### 使用示例

```bash
# 查询 WebSocket URL
curl http://provider-host:9223/ws
# 返回: {"ws_url": "ws://provider-host:9222/devtools/browser/xxx", ...}

# 健康检查
curl http://provider-host:9223/health

# 统计信息
curl http://provider-host:9223/stats

# 直接截图
curl -X POST "http://provider-host:9223/screenshot?url=https://example.com"
```

### 其他工具连接 Provider

```bash
# CLI — 通过 --wss 连接
./snir scan example.com --wss ws://provider-host:9222/devtools/browser/xxx

# API 服务 — 连接 Provider 管理的 Chrome
./snir api --wss ws://provider-host:9222/devtools/browser/xxx
```

```go
// Go SDK — 连接远程 Chrome
client, _ := sdk.NewRemoteClient("ws://provider-host:9222/devtools/browser/xxx", 4)

// Go SDK — 自动发现本地 Provider
client, mode, _ := sdk.AutoConnectClient(sdk.DefaultClientOptions())
// mode: "discovered" = 找到了 Provider
```

---

## 实战组合示例

```bash
# 生产级部署：高并发 + 空闲超时 + 代理
./snir provider \
  --port 9223 \
  --chrome-port 9222 \
  --max-concurrent 20 \
  --idle-timeout 5m \
  --proxy http://proxy:8080

# 调试模式：显示浏览器 + 调试日志
./snir provider --headless=false -D

# 配合 API 服务使用
# 终端 1: 启动 Provider
./snir provider --port 9223

# 终端 2: API 连接 Provider 的 Chrome
./snir api --wss ws://localhost:9222/devtools/browser/xxx --port 8080
```