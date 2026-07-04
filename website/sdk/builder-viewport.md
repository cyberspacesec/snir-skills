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

::: tip 三条路径，从简到繁按需选
| 路径 | 选项 | 设了什么 | 适合 |
|------|------|---------|------|
| 仅视口 | `WithViewport(w,h)` | 只改尺寸 | 桌面端不同分辨率 |
| 内置预设 | `WithDevice("iphone-15")` | UA+视口+像素比+移动+触摸 | 模拟某真机 |
| 全手动 | `WithDeviceEmulation(...)` | 逐项指定 | 自定义设备/反检测 |

预设清单见 [设备模拟 CLI](../cli/scan-device) 与 `pkg/runner/device_presets.go`。
:::

## 视口与设备应用到 CDP 时序

三条路径最终都汇入 CDP 的 `Emulation` 域，时序如下：

```mermaid
sequenceDiagram
    participant U as 用户
    participant O as ScreenshotOptions
    participant P as 设备预设表
    participant Dr as Driver
    participant CDP as CDP Emulation 域

    alt WithViewport(w,h)
        U->>O: 仅视口尺寸
        O->>Dr: 传入宽高
        Dr->>CDP: Emulation.setDeviceMetricsOverride(宽高)
    else WithDevice("iphone-15")
        U->>O: 设备名
        O->>P: 查 applyDevicePreset
        P-->>O: UA+视口+DPR+mobile+touch
        O->>Dr: 应用全套设备参数
        Dr->>CDP: Emulation.setDeviceMetricsOverride + setUserAgentOverride
    else WithDeviceEmulation(w,h,scale,...)
        U->>O: 全手动参数
        O->>Dr: 逐项参数
        Dr->>CDP: Emulation.setDeviceMetricsOverride(全参数)
    else WithMobileEmulation + WithTouchEmulation
        U->>O: 移动视口 + 触摸
        O->>Dr: DPR + 触摸开关
        Dr->>CDP: Emulation.setTouchEmulationEnabled
    end
    CDP-->>Dr: 模拟已生效
    Dr->>CDP: 导航 + 截图
```

四条路径差异在"设了什么"，但最终都通过 CDP `Emulation` 域落地。

## 下一步

- [构建器总览](./builders)
- [截图构建器](./builder-screenshot)
- [设备模拟（进阶）](../advanced/device)
