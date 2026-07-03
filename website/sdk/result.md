# 结果与证据

<p align="center">📦 `pkg/sdk/result.go` — 结果包装与便捷提取。</p>

> 📁 源码：[`pkg/sdk/result.go`](https://github.com/cyberspacesec/snir-skills/blob/main/pkg/sdk/result.go)

## 类型

| 符号 | 源码 | 说明 |
|------|------|------|
| `EvidenceSummary` | [L15](https://github.com/cyberspacesec/snir-skills/blob/main/pkg/sdk/result.go#L15) | 证据摘要 |
| `EvidenceBundle` | [L30](https://github.com/cyberspacesec/snir-skills/blob/main/pkg/sdk/result.go#L30) | 完整证据包 |
| `ResultWrapper` | [L41](https://github.com/cyberspacesec/snir-skills/blob/main/pkg/sdk/result.go#L41) | Result 包装器 |
| `WrapResult(r)` | [L46](https://github.com/cyberspacesec/snir-skills/blob/main/pkg/sdk/result.go#L46) | 包装 |
| `hasTLSInfo(tls)` | [L352](https://github.com/cyberspacesec/snir-skills/blob/main/pkg/sdk/result.go#L352) | 是否有 TLS 信息 |
| `writePrettyJSON(path, value)` | [L378](https://github.com/cyberspacesec/snir-skills/blob/main/pkg/sdk/result.go#L378) | 美化写 JSON |

## Result 字段（核心）

`*models.Result` 是一切采集的顶层结构，字段见 [Result Schema](../reference/result-schema) 与 [pkg/models](../internals/models)。

```
Result
├── URL / FinalURL / Status / Title
├── Screenshot []byte (PNG)
├── HTML / Headers / Cookies / ConsoleLogs
├── NetworkLogs (HAR)
├── Technologies []Technology
├── PerceptionHash (pHash)
├── TLS 信息
├── Timestamp / Duration
└── Error (若有)
```

## 字节提取

`screenshotBytesFromResult`（[client.go#L231](https://github.com/cyberspacesec/snir-skills/blob/main/pkg/sdk/client.go#L231)）从 Result 安全取出截图字节，供 `*Bytes` 系列方法使用。

## EvidenceBundle

`EvidenceBundle` 把截图+HTML+HAR+Console+Cookies 打包，便于一次导出全部证据，批量场景常用。

## 包装流程

```mermaid
flowchart LR
  R[*models.Result] --> W[WrapResult]
  W --> RW[ResultWrapper]
  RW --> SUM[EvidenceSummary 摘要]
  RW --> BUN[EvidenceBundle 全量]
  RW --> PP[writePrettyJSON 落盘]
```

## 下一步

- [Result Schema](../reference/result-schema)
- [字段说明](../reference/fields)
- [pkg/models](../internals/models)
- [Client](./client)
