# 黑名单构建器

<p align="center">🚫 SDK 配置目标黑名单。</p>

## 选项

| 选项 | 说明 |
|------|------|
| `WithBlacklist(patterns...)` | 自定义规则 |
| `WithDefaultBlacklist()` | 启用内置默认规则 |
| `WithBlacklistFile(path)` | 规则文件 |
| `WithNoBlacklist()` | 禁用黑名单 |

## 示例

```go
// 默认 + 自定义
opts := sdk.NewScreenshotOptions(
    sdk.WithDefaultBlacklist(),
    sdk.WithBlacklist("internal.local", "10.0.0.0/8"),
)

// 规则文件
opts := sdk.NewScreenshotOptions(
    sdk.WithDefaultBlacklist(),
    sdk.WithBlacklistFile("blocklist.txt"),
)

// 禁用（仅受控内网）
opts := sdk.NewScreenshotOptions(
    sdk.WithNoBlacklist(),
)
```

## 规则语法

支持 CIDR（`10.0.0.0/8`）、正则（`.*:6379`）、字面量（`localhost`）。默认规则屏蔽内网与云元数据，见 [黑名单 CLI](../cli/scan-blacklist)。

URL 在进入浏览器前需通过黑名单关卡：

```mermaid
flowchart TD
    U[目标 URL] --> Chk{黑名单匹配}
    subgraph 规则来源
      D[WithDefaultBlacklist<br/>内网+元数据]
      C[WithBlacklist<br/>自定义]
      F[WithBlacklistFile<br/>文件]
    end
    D & C & F --> R[合并规则集]
    R --> Chk
    Chk -- 命中 --> Block[拒绝, 记录]
    Chk -- 通过 --> Nav[导航截图]
    No[WithNoBlacklist<br/>禁用] -. 关闭关卡 .-> Chk

    style Chk fill:#fff4e6,stroke:#e8a317,color:#1a1a1a
    style Block fill:#fde8e8,stroke:#d23a3a
    style Nav fill:#e6f4ea,stroke:#3aa676
```

## 安全

生产环境保留默认黑名单防 SSRF。仅在授权内网扫描时才 `WithNoBlacklist()`。见 [安全注意](../advanced/security)。

## 下一步

- [构建器总览](./builders)
- [黑名单（进阶）](../advanced/blacklist)
- [内部 pkg/runner/blacklist](../internals/runner-blacklist)
