# 视口与设备构建器

<p align="center">📱 控制视口尺寸与设备模拟。</p>

## 选项

| 选项 | 说明 |
|------|------|
| `WithViewport(width, height)` | 视口尺寸 |
| `WithDevice(name)` | 设备预设（如 `iphone-15`） |
| `WithDeviceEmulation(w, h, scale, isMobile, hasTouch)` | 精细设备模拟 |
| `WithMobileEmulation(scaleFactor)` | 移动端视口 |
| `WithTouchEmulation(enabled)` | 触摸仿真 |

## 示例

```go
// 简单视口
opts := sdk.NewScreenshotOptions(
    sdk.WithViewport(1920, 1080),
)

// 设备预设
opts := sdk.NewScreenshotOptions(
    sdk.WithDevice("iphone-15"),
)

// 精细控制
opts := sdk.NewScreenshotOptions(
    sdk.WithDeviceEmulation(390, 844, 3.0, true, true),
)

// 移动端 + 触摸
opts := sdk.NewScreenshotOptions(
    sdk.WithMobileEmulation(3.0),
    sdk.WithTouchEmulation(true),
)
```

## WithDevice vs WithDeviceEmulation

设备模拟有三条路径，从简到繁：

```mermaid
flowchart TD
    Need{需要设备模拟?}
    Need -- 否 --> V[WithViewport<br/>仅设视口尺寸]
    Need -- 是 --> Preset{用内置预设?}
    Preset -- 是 --> D[WithDevice<br/>iphone-15 等]
    Preset -- 否 --> Fine{手动指定参数?}
    Fine -- 仅移动视口 --> M[WithMobileEmulation<br/>+ WithTouchEmulation]
    Fine -- 全参数 --> E[WithDeviceEmulation<br/>w/h/scale/mobile/touch]
    D & E & M --> Apply[应用到 CDP Emulation 域]

    style D fill:#e6f4ea,stroke:#3aa676
    style E fill:#e6f4ea,stroke:#3aa676
    style Apply fill:#3aa676,stroke:#2a7a56,color:#fff
```

- `WithDevice(name)`：用预设（含 UA/视口/像素比/移动/触摸）
- `WithDeviceEmulation(...)`：手动指定各参数

预设清单见 [设备模拟 CLI](../cli/scan-device) 与 `pkg/runner/device_presets.go`。

## 下一步

- [构建器总览](./builders)
- [截图构建器](./builder-screenshot)
- [设备模拟（进阶）](../advanced/device)
