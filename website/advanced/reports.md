# 报告生成

<p align="center">📊 富 HTML 报告、转换、合并。</p>

`pkg/report` 处理采集产出，生成可视化报告。

## report html

从 JSONL 生成自包含富 HTML 报告：

```bash
snir scan file -f urls.txt --write-jsonl \
  --save-html --save-headers
snir report html -i results.jsonl -o report.html
snir webserve --dir .
```

报告包含：结果总览、截图缩略图网格、每结果元信息（URL/标题/状态码/技术栈/哈希）、证据展开。

::: tip 自包含 HTML
生成的 `report.html` 是**单文件自包含**，CSS/截图全内嵌，可直接发邮件、存档、离线打开，无需额外资源。
:::

## report convert

格式转换（如 JSONL → CSV）：

```bash
snir report convert -i results.jsonl -o results.csv
```

## report merge

合并多次扫描：

```bash
snir report merge -i batch1.jsonl -i batch2.jsonl -o merged.jsonl
snir report html -i merged.jsonl -o report.html
```

## webserve

本地 Web 服务托管产物：

```bash
snir webserve --host 0.0.0.0 --port 8080
```

浏览器访问查看报告与截图。

## 流程

```mermaid
flowchart LR
  SCAN[scan --write-jsonl] --> J[results.jsonl]
  J --> H[report html]
  J --> M[report merge]
  J --> C[report convert]
  H --> WS[webserve 查看]
```

## 内部

- `RichHTMLTemplate`：内置富 HTML 模板
- `ReadJSONLResults`：读 JSONL 反序列化
- `ReportData`：模板数据

报告从 JSONL 到自包含 HTML 的渲染时序：

```mermaid
sequenceDiagram
  participant U as 用户
  participant CMD as snir report html
  participant READ as ReadJSONLResults
  participant TPL as 模板渲染
  participant FS as 文件系统
  U->>CMD: -i results.jsonl -o report.html
  CMD->>READ: 逐行读 JSONL
  READ-->>CMD: []Result
  CMD->>CMD: 组装 ReportData（总览/缩略图/元信息）
  CMD->>TPL: 渲染 RichHTMLTemplate
  TPL->>TPL: 内联 CSS + base64 截图
  TPL-->>CMD: 自包含 HTML 字符串
  CMD->>FS: 写 report.html
  CMD-->>U: 完成（可邮件/离线打开）
```

见 [pkg/report](../internals/report)。

## 下一步

- [report 命令族](../cli/report)
- [pkg/report](../internals/report)
- [输出格式](./output-formats)
