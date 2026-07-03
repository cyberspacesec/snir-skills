# scan file

<p align="center">📄 `snir scan file` — 从文件批量扫描 URL。</p>

从文本文件读取 URL 列表，并发批量扫描。

## 用法

```bash
snir scan file -f <文件>
```

## 标志

| 标志 | 简写 | 默认 | 说明 |
|------|------|------|------|
| `--file` | `-f` | — | 包含 URL 列表的文件路径 |

文件每行一个 URL/host/IP。继承所有 scan 公共标志。

## 示例

```bash
# 基本批量
snir scan file -f urls.txt --threads 10

# 完整证据 + 持久化
snir scan file -f urls.txt --threads 10 \
  --full-page --save-html --save-headers \
  --write-jsonl --db

# 端口展开（裸 host/IP）
snir scan file -f hosts.txt --ports 80,443,8080,8443

# 代理轮换
snir scan file -f urls.txt \
  --proxy-list http://p1:8080 --proxy-list http://p2:8080 \
  --proxy-strategy round-robin
```

## 文件格式

每行一个目标，支持：

```
example.com
https://example.com/path
192.168.1.10
10.0.0.0/24
```

- 裸 host/IP：配合 `--ports`/`--http`/`--https` 展开为候选 URL
- 完整 URL：直接使用
- CIDR 行：由 `ExpandTargets` 展开为 IP 列表

文件各行按类型分流处理：

```mermaid
flowchart TD
    F[urls.txt 每行] --> Cl{类型判断}
    Cl -- 完整 URL --> U1[直接使用]
    Cl -- 裸 host/IP --> U2[按 --ports/--http/--https 展开]
    Cl -- CIDR --> U3[展开为 IP 列表]
    U1 & U2 & U3 --> M[合并候选 URL]
    M --> Pool[--threads 并发池]
    Pool --> Isolate[单目标失败不中断<br/>记 Result.Failed]

    style M fill:#3aa676,stroke:#2a7a56,color:#fff
    style Isolate fill:#e6f4ea,stroke:#3aa676
```

## 并发

`--threads`（默认 2）控制并发数。批量建议 5-20，视机器与目标限流而定。见 [并发与池](../advanced/concurrency)。

## 失败隔离

单个目标失败不中断整体，记录在 `Result.Failed`/`FailedReason`。`--max-retries` 控制单目标重试。

## 下一步

- [scan 总览](./scan)
- [端口展开](./scan-ports)
- [输出选项](./scan-output)
- [并发与池](../advanced/concurrency)
