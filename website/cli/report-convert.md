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

## 适用场景

- 把 JSONL 转为 CSV 供 Excel 分析
- 把采集结果转为下游工具所需格式
- 提取扁平字段用于 BI

## 下一步

- [report 总览](./report)
- [report merge](./report-merge)
- [输出格式](../advanced/output-formats)
