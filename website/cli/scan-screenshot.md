# 截图选项

<p align="center">🖼️ 控制截图的格式、范围与分辨率。</p>

## 标志

| 标志 | 默认 | 说明 |
|------|------|------|
| `--screenshot-path` | `screenshots` | 截图保存目录 |
| `--screenshot-format` | `png` | 格式（`png`/`jpeg`） |
| `--screenshot-quality` | `90` | JPEG 质量（1-100，仅 jpeg） |
| `--skip-screenshot` | `false` | 跳过保存截图（仅采集证据） |
| `--full-page` | `false` | 截完整页面（含滚动区域） |
| `--selector` | — | CSS 选择器，仅截匹配元素 |
| `--xpath` | — | XPath，仅截匹配元素 |
| `--resolution-x` | `1280` | 窗口宽度 |
| `--resolution-y` | `800` | 窗口高度 |

## 截图模式

```mermaid
flowchart TD
  M{截图模式}
  M -- 默认 --> V[视口截图]
  M -- --full-page --> F[完整页面]
  M -- --selector --> S[CSS 元素]
  M -- --xpath --> X[XPath 元素]
  M -- --skip-screenshot --> N[不截图,仅证据]
```

## 示例

```bash
# 视口截图（默认）
snir scan example.com

# 完整长页面
snir scan example.com --full-page

# 截某元素
snir scan example.com --selector "#main-content"
snir scan example.com --xpath "//div[@class='hero']"

# JPEG + 质量
snir scan example.com --screenshot-format jpeg --screenshot-quality 80

# 高分辨率
snir scan example.com --resolution-x 1920 --resolution-y 1080

# 仅证据不截图
snir scan example.com --skip-screenshot --save-html --save-headers
```

## 格式选择

- **PNG**：无损，适合需要清晰文字/UI 细节
- **JPEG**：有损体积小，适合批量存档

## 文件命名

截图文件名经 `SanitizeFilename` 清理非法字符（`\ / : * ? " < > | %`），保证跨平台安全。

## 下一步

- [证据选项](./scan-evidence)
- [Chrome 选项](./scan-chrome)
- [输出选项](./scan-output)
