# 数据流

<p align="center">🌊 一次截图从输入到产出的完整数据流。</p>

## 总览

```mermaid
flowchart LR
  T[目标 Target] --> S[scan.Scanner]
  S --> E[ExpandTargets 展开]
  E --> R[runner.Runner]
  R --> D[Driver: ChromeDP]
  D --> C[Chrome / CDP]
  C --> RES[Result]
  RES --> W1[JSONL]
  RES --> W2[CSV]
  RES --> W3[SQLite]
  RES --> W4[Stdout]
  RES --> RP[Report]
```

## 时序

```mermaid
sequenceDiagram
  participant U as 调用方
  participant S as Scanner
  participant R as Runner
  participant D as Driver(CDP)
  participant C as Chrome
  participant W as Writers

  U->>S: 目标 + Options
  S->>S: 归一化/端口展开(ExpandTargets)
  S->>R: 执行单次截图
  R->>D: 配置(视口/代理/Cookie/JS/设备)
  D->>C: 启动/复用会话
  D->>C: 导航到 URL
  C-->>D: 加载完成
  R->>R: 执行交互动作(可选)
  D->>C: 截图 + 采集证据(HTML/头/Cookie/控制台/网络/TLS)
  C-->>D: 原始数据
  D-->>R: 组装 Result
  R->>R: 感知哈希 / 技术检测
  R->>W: 分发 Result(JSONL/CSV/SQLite/Stdout)
  R-->>U: Result
```

## 各阶段详解

### 1. 目标输入

单 URL、文件列表、CIDR、裸 host+端口。见 [核心概念](./core-concepts)。

### 2. 归一化与展开

`scan.ExpandTargets` 把裸 host/IP 按协议与端口展开为候选 URL 列表。`models.EnrichEndpoint` 补全 host/port/scheme/endpoint。

### 3. Runner 执行

`Runner` 持 `Driver` + `[]Writer`，按 `Options` 配置浏览器，导航、执行交互、采集。

### 4. Driver 与 Chrome

`ChromeDP` 经 CDP 与 Chrome 通信：设置视口、注入 Cookie、应用指纹、走代理、注入 JS、触发动作、截图、抓证据。

### 5. Result 组装

浏览器原始数据组装为 `models.Result`，补 `schema_version`、`probed_at`、感知哈希、技术栈。

### 6. 持久化分发

`Result` 同时分发给所有启用的 `Writer`：JSONL（追加）、CSV（表格）、SQLite（结构化）、Stdout（控制台）。可再生成报告。

## 并发与池

批量时，多个目标并发从 `DriverPool` 借 Driver，复用浏览器实例。见 [并发与池](../advanced/concurrency)。

## 下一步

- [架构](./architecture)
- [Result Schema](../reference/result-schema)
- [输出格式](../advanced/output-formats)
