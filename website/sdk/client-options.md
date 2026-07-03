# ClientOptions

<p align="center">⚙️ `pkg/sdk/options.go` — Client 与截图选项。</p>

> 📁 源码：[`pkg/sdk/options.go`](https://github.com/cyberspacesec/snir-skills/blob/main/pkg/sdk/options.go)

## 类型

| 符号 | 源码 | 说明 |
|------|------|------|
| `ClientOptions` | [L11](https://github.com/cyberspacesec/snir-skills/blob/main/pkg/sdk/options.go#L11) | Client 级配置 |
| `DefaultClientOptions()` | [L97](https://github.com/cyberspacesec/snir-skills/blob/main/pkg/sdk/options.go#L97) | 安全默认 |
| `ScreenshotOptions` | [L118](https://github.com/cyberspacesec/snir-skills/blob/main/pkg/sdk/options.go#L118) | 单次截图配置 |
| `toRunnerOptions(co)` | [L201](https://github.com/cyberspacesec/snir-skills/blob/main/pkg/sdk/options.go#L201) | 转 runner.Options |
| `mergeWithScreenshotOptions(base, so)` | [L326](https://github.com/cyberspacesec/snir-skills/blob/main/pkg/sdk/options.go#L326) | 合并截图选项 |
| `formHasConfig(form)` | [L557](https://github.com/cyberspacesec/snir-skills/blob/main/pkg/sdk/options.go#L557) | 表单是否有配置 |
| `applyDevicePreset(device, opts)` | [L564](https://github.com/cyberspacesec/snir-skills/blob/main/pkg/sdk/options.go#L564) | 应用设备预设 |

## ClientOptions 字段

| 字段 | 说明 |
|------|------|
| `MaxConcurrent` | 池大小/并发上限 |
| `ChromePath` | 本地 Chrome 路径 |
| `WSSURL` | 远程调试 URL |
| `IdleTimeout` | 空闲超时 |
| `BlacklistEnabled` | SSRF 防护开关 |
| `OutputPath` | 默认输出目录 |

## 两层配置

```mermaid
flowchart TD
  CO[ClientOptions 全局] --> TO[toRunnerOptions]
  TO --> BASE[runner.Options 基线]
  SO[ScreenshotOptions 单次] --> MW[mergeWithScreenshotOptions]
  BASE --> MW
  MW --> FIN[最终 runner.Options]
```

- `ClientOptions`：跨多次截图的稳定配置（Chrome、池大小）
- `ScreenshotOptions`：单次覆盖（视口、证据、代理）

## DefaultClientOptions

[`DefaultClientOptions`](https://github.com/cyberspacesec/snir-skills/blob/main/pkg/sdk/options.go#L97) 提供开箱即用的安全默认：启用黑名单、合理并发与超时。

## 设备预设

[`applyDevicePreset`](https://github.com/cyberspacesec/snir-skills/blob/main/pkg/sdk/options.go#L564)：当指定 `device` 时，从 [`device_presets`](../internals/runner-device) 取视口/UA/DPR 应用到 Options。

## 下一步

- [Client](./client)
- [构建器](./builders)
- [Options（内部）](../internals/runner-options)
- [设备模拟](../advanced/device)
