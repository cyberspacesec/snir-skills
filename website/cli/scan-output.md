# 输出选项

<p align="center">📤 控制结果写往何处：JSONL/CSV/控制台。</p>

## 标志

| 标志 | 默认 | 说明 |
|------|------|------|
| `--write-jsonl` | `false` | 写 JSONL |
| `--jsonl-file` | `results.jsonl` | JSONL 文件路径 |
| `--write-csv` | `false` | 写 CSV |
| `--csv-file` | `results.csv` | CSV 文件路径 |
| `--write-stdout` | `true` | 输出到控制台 |

## 示例

```bash
# JSONL
snir scan example.com --write-jsonl

# CSV
snir scan example.com --write-csv

# 多个同时
snir scan file -f urls.txt \
  --write-jsonl --jsonl-file out.jsonl \
  --write-csv --csv-file out.csv \
  --write-stdout=false

# 自定义路径
snir scan example.com --write-jsonl --jsonl-file /data/scan.jsonl
```

## 各格式特点

```mermaid
flowchart LR
    R[Result] -->|write-jsonl| J[results.jsonl]
    R -->|write-csv| C[results.csv]
    R -->|write-stdout| T[终端实时]
    J --> jq[jq 管线]
    C --> ex[(Excel/BI)]
    T --> Script[脚本捕获]

    style R fill:#3aa676,stroke:#2a7a56,color:#fff
```

`Result` 产出后被分发给各 Writer、最终落盘为不同格式的状态流转：

```mermaid
stateDiagram-v2
  [*] --> 采集: 截图+证据完成
  采集 --> 分发: Result 就绪
  分发 --> 写JSONL: --write-jsonl
  分发 --> 写CSV: --write-csv
  分发 --> 写控制台: --write-stdout
  分发 --> 写SQLite: --db
  写JSONL --> 已落盘: results.jsonl
  写CSV --> 已落盘: results.csv
  写控制台 --> 已落盘: 终端实时
  写SQLite --> 已落盘: snir.db
  已落盘 --> [*]
```

| 格式 | 特点 | 适合 |
|------|------|------|
| JSONL | 流式、每行一条 JSON、追加友好 | 管线、jq 处理 |
| CSV | 表格、扁平 | Excel、BI |
| Stdout | 实时控制台 | 调试、脚本捕获 |

## JSONL 示例

每行一个完整 `Result` JSON：

```bash
snir scan file -f urls.txt --write-jsonl --write-stdout=false
jq -c 'select(.failed == true)' results.jsonl   # 看失败
jq -c '{url, title, code: .response_code}' results.jsonl
```

## CSV 注意

::: warning CSV 会丢嵌套结构
CSV 是扁平表格，`headers`/`network`/`cookies`/`console` 这类嵌套证据会被**序列化为字符串或省略**。

- ✅ 只要 URL/标题/状态码等标量字段 → 用 CSV，Excel 友好
- ❌ 需要完整证据 → 用 JSONL（流式）或 SQLite（结构化可查询）
:::

## 与数据库区别

::: info 文件 vs 数据库，可同时启用
- `--write-jsonl` / `--write-csv`：**文件**输出，适合一次性产物、管线处理
- `--db`：**SQLite 结构化**存储，适合跨次累积、SQL 查询、关联分析

两者不互斥，可同时启用——`Result` 会被分发给所有 Writer，写多份各取所需。
:::

## 下一步

- [数据库选项](./scan-db)
- [输出格式（进阶）](../advanced/output-formats)
- [Result Schema](../reference/result-schema)
