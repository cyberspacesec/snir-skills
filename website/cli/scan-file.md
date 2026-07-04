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

批量扫描中"并发下发 + 单点失败隔离"的时序：

```mermaid
sequenceDiagram
  participant F as urls.txt
  participant EX as 目标展开
  participant POOL as 并发池
  participant T1 as 任务1 Driver
  participant T2 as 任务2 Driver
  participant W as Writer
  F->>EX: 逐行解析
  EX-->>POOL: 候选 URL[]
  POOL->>T1: URL_A
  POOL->>T2: URL_B
  T1-->>POOL: Result_A（成功）
  T2-->>POOL: Result_B（失败）
  POOL->>W: 写 Result_A
  POOL->>W: 写 Result_B（Failed=true，不中断）
  Note over POOL: 继续后续 URL 直到列表耗尽
```

## 并发

`--threads`（默认 2）控制并发数。批量建议 5-20，视机器与目标限流而定。见 [并发与池](../advanced/concurrency)。

::: warning 并发不是越大越好
盲目调高 `--threads` 会导致：本地内存/CPU 打满、目标站点触发限流或封禁、Chrome 进程数爆炸（每并发一个 tab）。
建议从 `--threads 5` 起，观察资源占用与成功率再逐步上调。
:::

## 失败隔离

::: tip 单目标失败不连累全局
批量扫描中某个 URL 超时/报错**不会中断整个批次**，会被记录到 `Result.Failed` / `FailedReason` 继续下一个。配合 `--max-retries` 控制单目标重试次数。
:::

## 下一步

- [scan 总览](./scan)
- [端口展开](./scan-ports)
- [输出选项](./scan-output)
- [并发与池](../advanced/concurrency)
