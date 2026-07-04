# scan single

<p align="center">📷 `snir scan single [url]` — 扫描单个 URL。</p>

对单个 URL 执行截图与信息收集。也可直接 `snir scan <url>`。

## 用法

```bash
snir scan single <url>
# 或
snir scan <url>
```

## 示例

```bash
# 基本截图
snir scan example.com

# 带协议
snir scan https://example.com/path

# 完整证据
snir scan single example.com \
  --full-page --save-html --save-headers \
  --save-cookies --save-console --save-network \
  --write-jsonl --db

# 设备模拟 + 代理
snir scan single example.com --device iphone-15 --proxy http://127.0.0.1:8080

# 元素截图
snir scan single example.com --selector "#main"
```

## URL 归一化

::: info 无需手动补协议
直接传 `example.com` 即可——snir 的 `EnrichEndpoint` 会自动补全 `http://`/`https://`（依 `--http`/`--https` 默认两者都试），并解析出 host/port/scheme。
:::

```mermaid
flowchart LR
    In["example.com / https://x.com/p"] --> Enr[EnrichEndpoint]
    Enr --> Parse[解析 host/port/scheme]
    Parse --> Nav[导航截图]
    Nav --> Out[截图 + Result]

    style Enr fill:#3aa676,stroke:#2a7a56,color:#fff
    style Out fill:#e6f4ea,stroke:#3aa676
```

## 输出

::: details 输出位置一览
| 产物 | 默认位置 | 调整标志 |
|------|---------|---------|
| 截图 | `./screenshots/` | `--screenshot-path` |
| JSONL | `./results.jsonl` | `--write-jsonl` 启用 |
| CSV | `./results.csv` | `--write-csv` 启用 |
| SQLite | `./snir.db` | `--db` 启用 |
| 控制台 | 总是输出 | `--write-stdout=false` 关闭 |
:::

## 错误处理

::: tip 错误信息已美化
常见错误（超时、`net::ERR_*`、`element not found`）会被翻译成可读的中文建议而非裸堆栈。见 [错误码](../reference/error-codes)。
:::

## 下一步

- [scan 总览](./scan)
- [截图选项](./scan-screenshot)
- [证据选项](./scan-evidence)
