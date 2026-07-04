# report convert

<p align="center">🔁 `snir report convert` — 转换结果格式。</p>

在不同结果格式间转换，例如 JSONL → CSV 等。

## 用法

```bash
snir report convert [flags]
```

## 示例

```bash
# JSONL 转 CSV
snir report convert -i results.jsonl -o results.csv

# 转换并指定格式
snir report convert -i results.jsonl --format csv -o out.csv
```

## 转换选项

`ConvertOptions` 控制输入/输出路径与目标格式。具体支持的格式组合见 `pkg/report/convert.go`。

```mermaid
flowchart LR
    In["results.jsonl"] --> Conv[report convert]
    Conv -->|JSONL→CSV| O1["results.csv"]
    Conv -->|提取扁平字段| O2["BI 友好表格"]
    O1 --> Excel[(Excel / BI 工具)]

    style Conv fill:#3aa676,stroke:#2a7a56,color:#fff
    style Excel fill:#e6f4ea,stroke:#3aa676
```

JSONL → CSV 转换的逐行处理时序：

```mermaid
sequenceDiagram
  participant CLI as report convert
  participant R as JSONL Reader
  participant F as 字段扁平化
  participant W as CSV Writer
  participant FS as out.csv
  CLI->>R: 打开 results.jsonl
  loop 每行
  R->>F: 一条 Result JSON
  F->>F: 提取 url/title/code/hash 等标量
  F->>W: 一行 CSV 记录
  W->>FS: 追加写入
  end
  R-->>CLI: EOF
  CLI-->>CLI: 关闭文件
  Note over F,W: 嵌套证据 headers/network 会被省略或序列化为字符串
```

## 适用场景

::: tip JSONL → CSV 是最常用转换
JSONL 适合管线与 jq，但同事/老板要看 Excel。`report convert -i x.jsonl -o x.csv` 一步转成扁平表格，直接双击打开。

注意：CSV 会丢嵌套证据（headers/network 等），要全量证据请保留 JSONL。
:::

- 把 JSONL 转为 CSV 供 Excel 分析
- 把采集结果转为下游工具所需格式
- 提取扁平字段用于 BI

## 下一步

- [report 总览](./report)
- [report merge](./report-merge)
- [输出格式](../advanced/output-formats)
