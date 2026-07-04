# 代理与轮换

<p align="center">🔀 用代理采集，支持单代理与多策略轮换池。</p>

## 代理来源

snir 支持四种代理来源，由 `pkg/runner/proxy.go` 的 `ProxyProvider` 抽象：

| 来源 | CLI | SDK | 特点 |
|------|-----|-----|------|
| 单代理 | `--proxy` | `WithProxy` | 单一出口 |
| 列表轮换 | `--proxy-list` | `WithProxyList` | 少量固定代理 |
| 文件热加载 | `--proxy-file` | `WithProxyFile` | 大量、可更新 |
| 动态 API | `--proxy-url` | `WithProxyURL` | 商业动态池 |

## 轮换策略

`ProxyStrategy`：

| 策略 | 行为 |
|------|------|
| `round-robin` | 顺序循环 |
| `random` | 随机选取 |
| `sequential` | 顺序，失败切换 |

## 选择决策

```mermaid
flowchart TD
  Q{代理规模?}
  Q -- 1 个 --> S[单代理]
  Q -- 少量固定 --> L[列表]
  Q -- 大量/需更新 --> F[文件]
  Q -- 商业动态池 --> A[动态 API]
```

## CLI 示例

```bash
# 单代理
snir scan example.com --proxy http://127.0.0.1:8080

# SOCKS5
snir scan example.com --proxy socks5://127.0.0.1:1080

# 列表轮换
snir scan file -f urls.txt \
  --proxy-list http://p1:8080 --proxy-list http://p2:8080 \
  --proxy-strategy round-robin

# 文件热加载
snir scan file -f urls.txt --proxy-file proxies.txt --proxy-strategy random

# 动态 API
snir scan file -f urls.txt --proxy-url http://proxy-service/api
```

## SDK 示例

```go
opts := sdk.NewScreenshotOptions(
    sdk.WithProxyList(runner.ProxyStrategyRoundRobin,
        "http://p1:8080", "http://p2:8080",
    ),
)
```

## 代理文件格式

每行一个代理，支持 `http://`、`https://`、`socks5://`，可带认证：

```
http://127.0.0.1:8080
socks5://user:pass@host:1080
http://proxy:3128
```

`--proxy-file` 支持热加载，文件变更自动生效。

## 动态代理 API

`--proxy-url` 指向返回代理地址的 HTTP 服务，每次请求获取新代理，适合按需取号的商业池。

## 失败与切换

`sequential` 策略在代理失败时自动切换下一个，适合不稳定代理。配合 `--max-retries` 提高成功率。

`sequential` 策略的代理切换状态机：

```mermaid
stateDiagram-v2
  [*] --> 取下一个代理
  取下一个代理 --> 通过代理采集
  通过代理采集 --> 成功: 200/可接受
  通过代理采集 --> 失败: 超时/连接错误
  失败 --> 重试计数+1
  重试计数+1 --> 通过代理采集: 未达 max-retries
  重试计数+1 --> 取下一个代理: 达 max-retries
  成功 --> [*]
```

## 安全注意

::: warning 代理 ≠ 匿名
代理只改 HTTP/HTTPS 出口，**不等于匿名**。以下渠道仍可能泄露真实 IP：

- 🌐 **WebRTC**：浏览器 WebRTC 会绕过代理直连 STUN 服务器暴露内网 IP
- 🐘 **DNS 泄漏**：DNS 查询可能不走代理解析
- 🧩 **浏览器插件/扩展**：可能直连回源

务必配合 `WithDisableWebRTC()`，并在采集前用 `https://browserleaks.com/webrtc` 类站点验证。
:::

::: danger 合规底线
- ✅ 遵守代理服务商条款与目标站点的访问策略
- ✅ 遵守 `robots.txt` 与目标站点速率限制
- ❌ 勿用代理规避法律或服务条款约束
:::

## 下一步

- [代理选项 CLI](../cli/scan-proxy)
- [代理构建器 SDK](../sdk/builder-proxy)
- [内部 pkg/runner/proxy](../internals/runner-proxy)
- [浏览器指纹](./fingerprint)
