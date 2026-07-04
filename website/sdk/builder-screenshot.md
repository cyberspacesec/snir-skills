# 截图构建器

<p align="center">🖼️ 控制截图本身的 `With*` 选项。</p>

各选项作用于截图执行流程的不同阶段：

```mermaid
flowchart LR
    A[启动] --> B[导航加载<br/>WithTimeout]
    B --> C[加载后等待<br/>WithDelay]
    C --> D{截图范围}
    D -->|--full-page| E1[整页]
    D -->|--element/xpath| E2[元素区域]
    D -->|默认| E3[视口]
    E1 & E2 & E3 --> F[格式化<br/>WithFormat]
    F --> G{是否保存}
    G -->|--skip-save| H1[不落盘]
    G -->|默认| H2[写入<br/>WithScreenshotPath]
    H1 & H2 --> Z[结果]

    style B fill:#e6f4ea,stroke:#3aa676
    style C fill:#e6f4ea,stroke:#3aa676
    style F fill:#e6f4ea,stroke:#3aa676
    style H2 fill:#e6f4ea,stroke:#3aa676
```

失败时 `WithMaxRetries` 控制重试次数。

## 选项

| 选项 | 说明 |
|------|------|
| `WithTimeout(d)` | 页面加载超时 |
| `WithDelay(d)` | 截图前等待 |
| `WithFullPage()` | 完整页面截图 |
| `WithElement(selector)` | CSS 选择器截图 |
| `WithXPath(xpath)` | XPath 截图 |
| `WithFormat(format, quality)` | 格式（png/jpeg）与质量 |
| `WithScreenshotPath(path)` | 截图保存目录 |
| `WithSkipSave()` | 跳过保存截图 |
| `WithMaxRetries(n)` | 最大重试次数 |

## 示例

```go
opts := sdk.NewScreenshotOptions(
    sdk.WithTimeout(60 * time.Second),
    sdk.WithDelay(2 * time.Second),
    sdk.WithFullPage(),
    sdk.WithFormat("jpeg", 80),
    sdk.WithScreenshotPath("./out"),
    sdk.WithMaxRetries(3),
)

// 元素截图
opts2 := sdk.NewScreenshotOptions(
    sdk.WithElement("#main-content"),
    sdk.WithFormat("png", 0),
)
```

## 超时与延迟

::: tip timeout 管加载，delay 管渲染收尾
- `WithTimeout(d)`：**整体页面加载超时**，到点没 load 完直接失败
- `WithDelay(d)`：**加载完成后再等**，留给异步内容/动画/懒加载渲染

慢站点/SPA 常见组合：`WithTimeout(60*time.Second)` + `WithDelay(2*time.Second)`——给足加载，再等渲染收尾。
:::

## 格式

::: info PNG 无视 quality，JPEG 才看
`WithFormat("jpeg", 80)`：JPEG 质量 80（体积/质量甜点）。
`WithFormat("png", 0)`：PNG 无损，**quality 参数被忽略**传 0 即可。
:::

## 跳过保存

`WithSkipSave()` 不落盘截图，常与字节模式或纯证据采集搭配。

## 下一步

- [构建器总览](./builders)
- [视口与设备](./builder-viewport)
- [截图选项 CLI](../cli/scan-screenshot)
