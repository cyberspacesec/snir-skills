# report merge

<p align="center">🔗 `snir report merge` — 合并多个结果文件。</p>

把多次扫描的 JSONL 结果合并为一个文件，便于统一查看或报告。

## 用法

```bash
snir report merge [flags]
```

## 示例

```bash
# 合并两个批次
snir report merge -i batch1.jsonl -i batch2.jsonl -o merged.jsonl

# 合并多个并生成报告
snir report merge -i *.jsonl -o all.jsonl
snir report html -i all.jsonl -o report.html
```

## 合并选项

`MergeOptions` 控制多个输入与输出路径。`Merge` 读取所有输入 JSONL，去重/合并后写出。

::: info 多 worker 分布式采集的标准收尾
多机/多 worker 各扫一批 → 各产 `batchN.jsonl` → `report merge -i *.jsonl -o all.jsonl` 汇总 → `report html` 出一份总报告。这是分布式采集的典型收尾流程。
:::

```mermaid
flowchart LR
    I1["batch1.jsonl"] --> M[Merge]
    I2["batch2.jsonl"] --> M
    In["...*.jsonl"] --> M
    M --> Dedup[去重/合并]
    Dedup --> O["all.jsonl"]
    O --> RH["→ report html 生成报告"]

    style M fill:#3aa676,stroke:#2a7a56,color:#fff
    style O fill:#e6f4ea,stroke:#3aa676
```

多批次 JSONL 合并的去重/汇总时序：

```mermaid
sequenceDiagram
  participant W1 as worker 1
  participant W2 as worker 2
  participant Wn as worker N
  participant M as Merge
  participant D as 去重/合并
  participant O as all.jsonl
  W1->>M: batch1.jsonl
  W2->>M: batch2.jsonl
  Wn->>M: batchN.jsonl
  M->>M: 逐行读取所有输入
  M->>D: 按 URL/哈希判重
  D->>D: 合并同目标多版本
  D->>O: 写出合并结果
  O-->>M: 完成
  Note over M,O: 后续 report html 生成总报告
```

## 适用场景

- 多次分批扫描后统一汇总
- 多 worker 各自产出后合并
- 跨时间段结果归并

## 下一步

- [report 总览](./report)
- [report html](./report-html)
- [输出格式](../advanced/output-formats)
