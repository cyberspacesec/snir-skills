# 目标展开

<p align="center">🎯 `pkg/sdk/targets.go` — 把输入归一为目标列表。</p>

> 📁 源码：[`pkg/sdk/targets.go`](https://github.com/cyberspacesec/snir-skills/blob/main/pkg/sdk/targets.go)

## 函数

| 符号 | 源码 | 说明 |
|------|------|------|
| `ExpandTargets(targets, opts)` | [L6](https://github.com/cyberspacesec/snir-skills/blob/main/pkg/sdk/targets.go#L6) | 批量展开 |
| `ExpandTarget(target, opts)` | [L12](https://github.com/cyberspacesec/snir-skills/blob/main/pkg/sdk/targets.go#L12) | 单个展开 |

## 展开规则

```mermaid
flowchart TD
  T[输入字符串] --> Q{类型?}
  Q -- URL/域名 --> U[补协议后加入]
  Q -- IP --> I[加入]
  Q -- CIDR --> C[展开为 IP 列表]
  Q -- 文件路径 --> F[逐行读取]
  U --> OUT
  I --> OUT
  C --> OUT
  F --> OUT
  OUT[[]string]
```

::: info 四类输入自动识别，混合传入也无妨
`ExpandTargets` 对每个输入字符串按类型分流：URL/域名补 `http://` 协议后加入；IP 直接加入；CIDR（如 `192.168.1.0/30`）展开为 IP 列表；文件路径（存在且可读）逐行读取。

→ 一条命令里 URL、网段、文件混着传都行，统一成 `[]string` 喂给 `BatchCapture`。
:::

## 与批量采集配合

`ExpandTargets` 的输出喂给 `BatchCapture`：

```mermaid
flowchart LR
  IN[混合输入] --> ET[ExpandTargets]
  ET --> TL[[]string]
  TL --> BC[Client.BatchCapture]
  BC --> RES[[]BatchResult]
```

## 示例

```go
targets := sdk.ExpandTargets([]string{
    "example.com",
    "192.168.1.0/30",
    "urls.txt",
}, nil)
```

## 下一步

- [批量采集](./batch)
- [Scan（内部）](../internals/scan)
- [CLI scan file](../cli/scan-file)
- [CLI scan cidr](../cli/scan-cidr)
