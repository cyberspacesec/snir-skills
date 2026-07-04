# 代理选项

<p align="center">🔀 通过代理采集，支持单代理与轮换池。</p>

## 标志

| 标志 | 说明 |
|------|------|
| `--proxy` | 单代理地址（如 `http://127.0.0.1:8080`） |
| `--proxy-list` | 代理列表（可多次使用，轮换） |
| `--proxy-file` | 代理文件路径（每行一个，支持热加载） |
| `--proxy-url` | 动态代理 API URL（每次获取新代理） |
| `--proxy-strategy` | 轮换策略：`round-robin`/`random`/`sequential` |

## 策略

| 策略 | 行为 |
|------|------|
| `round-robin` | 按顺序循环使用 |
| `random` | 随机选取 |
| `sequential` | 顺序使用，失败切换 |

## 示例

```bash
# 单代理
snir scan example.com --proxy http://127.0.0.1:8080

# SOCKS5
snir scan example.com --proxy socks5://127.0.0.1:1080

# 列表轮换
snir scan file -f urls.txt \
  --proxy-list http://p1:8080 \
  --proxy-list http://p2:8080 \
  --proxy-list http://p3:8080 \
  --proxy-strategy round-robin

# 文件（热加载）
snir scan file -f urls.txt --proxy-file proxies.txt --proxy-strategy random

# 动态代理 API
snir scan file -f urls.txt --proxy-url http://proxy-service/api --proxy-strategy random
```

## 代理文件格式

每行一个代理，支持注释：

```
http://127.0.0.1:8080
socks5://127.0.0.1:1080
http://user:pass@proxy:3128
```

`--proxy-file` 支持热加载，文件变更自动生效。

::: info sequential 策略 = 自动故障转移
代理不稳定时选 `--proxy-strategy sequential`——当前代理失败会**自动切到下一个**，配合 `--max-retries` 重试最大化成功率。`round-robin`/`random` 则不论成败轮换。
:::

## 动态代理 API

`--proxy-url` 指向一个返回代理地址的 HTTP 服务，每次请求获取新代理，适合商业动态代理池。

## 选择建议

按代理规模与变更频率选择来源，再配策略：

```mermaid
flowchart TD
    Q{代理规模?}
    Q -- 单一 --> P1[--proxy]
    Q -- 少量固定 --> P2[--proxy-list]
    Q -- 大量、需更新 --> P3[--proxy-file 热加载]
    Q -- 商业动态池 --> P4[--proxy-url API]
    P1 & P2 & P3 & P4 --> S{轮换策略}
    S --> |round-robin| S1[顺序循环]
    S --> |random| S2[随机]
    S --> |sequential| S3[顺序失败切换]

    style S fill:#3aa676,stroke:#2a7a56,color:#fff
```

`--proxy-url` 动态代理池的取号时序：

```mermaid
sequenceDiagram
  participant SCAN as scan
  participant API as proxy-url 服务
  participant CH as Chrome
  participant POOL as 商业代理池
  SCAN->>API: GET（每次取新代理）
  API->>POOL: 申请一个出口
  POOL-->>API: 1.2.3.4:8888
  API-->>SCAN: 返回代理地址
  SCAN->>CH: 用该代理采集
  CH-->>SCAN: Result（出口 IP 随每次变化）
```

| 场景 | 选 |
|------|-----|
| 单一出口 | `--proxy` |
| 少量固定代理 | `--proxy-list` |
| 大量代理、需更新 | `--proxy-file` |
| 商业动态代理 | `--proxy-url` |

## 下一步

- [Chrome 选项](./scan-chrome)
- [代理与轮换（进阶）](../advanced/proxy)
- [内部 pkg/runner/proxy](../internals/runner-proxy)
