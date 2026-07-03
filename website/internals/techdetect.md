# 技术检测（TechDetect）

<p align="center">🔍 `pkg/techdetect/techdetect.go` — 识别目标网站技术栈。</p>

`pkg/techdetect` 通过分析 HTTP 头、HTML、Cookie、JS 全局变量等指纹，推断目标使用的前端框架、CMS、服务器、分析工具等。

> 📁 源码：[`pkg/techdetect/techdetect.go`](https://github.com/cyberspacesec/snir-skills/blob/main/pkg/techdetect/techdetect.go)

## 核心类型

| 符号 | 源码 | 说明 |
|------|------|------|
| `Technology` | [L11](https://github.com/cyberspacesec/snir-skills/blob/main/pkg/techdetect/techdetect.go#L11) | 一项技术定义 |
| `Fingerprint` | [L18](https://github.com/cyberspacesec/snir-skills/blob/main/pkg/techdetect/techdetect.go#L18) | 单条指纹规则 |
| `Detector` | [L31](https://github.com/cyberspacesec/snir-skills/blob/main/pkg/techdetect/techdetect.go#L31) | 检测器 |
| `NewDetector()` | [L42](https://github.com/cyberspacesec/snir-skills/blob/main/pkg/techdetect/techdetect.go#L42) | 构造（含内置规则） |
| `(*Detector) Detect(evidence)` | [L67](https://github.com/cyberspacesec/snir-skills/blob/main/pkg/techdetect/techdetect.go#L67) | 执行检测 |
| `DefaultTechnologies` | [L120](https://github.com/cyberspacesec/snir-skills/blob/main/pkg/techdetect/techdetect.go#L120) | 内置规则集 |

## 指纹维度

`Fingerprint` 可在多个维度匹配：

| 维度 | 来源 | 示例 |
|------|------|------|
| `Headers` | HTTP 响应头 | `Server: nginx`、`X-Powered-By: Express` |
| `HTML` | 页面 HTML | `<meta name="generator" content="WordPress">` |
| `Cookies` | Set-Cookie 名 | `session_id`、`csrftoken` |
| `ScriptSrc` | `<script src>` | `jquery.js`、`react.production` |
| `Meta` | meta 标签 | generator、viewport |
| `JS` | JS 全局变量 | `window.React`、`window.Vue` |

## Detect 流程

```mermaid
flowchart TD
  E[Result 证据] --> D[Detect]
  D --> LP[遍历 DefaultTechnologies]
  LP --> MT{匹配任一指纹?}
  MT -- 是 --> HIT[记录 Technology]
  MT -- 否 --> SKIP[跳过]
  HIT --> NX[下一项技术]
  SKIP --> NX
  NX --> ALL{全部遍历完?}
  ALL -- 否 --> LP
  ALL -- 是 --> OUT[返回 []Technology]
```

## Technology 字段

| 字段 | 说明 |
|------|------|
| `Name` | 技术名（如 `nginx`、`React`） |
| `Category` | 分类（`Web servers`、`JS frameworks`） |
| `Version` | 推断版本（可能为空） |
| `Confidence` | 置信度 0~1 |
| `Matches` | 命中的指纹详情 |

## 内置规则

[`DefaultTechnologies`](https://github.com/cyberspacesec/snir-skills/blob/main/pkg/techdetect/techdetect.go#L120) 内置常见技术规则，覆盖主流 CMS、框架、服务器、CDN、分析工具。可自行扩展 `Detector.Technologies` 加自定义规则。

## 与采集的衔接

`Detect` 输入是 `Result` 中已采集的证据（Headers/HTML/Cookies/JS），无需额外请求，零成本附加在每次截图后。

```mermaid
flowchart LR
  R[Result 含证据] --> TD[techdetect.Detect]
  TD --> TS[[]Technology]
  TS --> M[写入 Result.Technologies]
  M --> RPT[报告展示]
```

见 [技术检测（进阶）](../advanced/tech-detection)。

## 下一步

- [技术检测（进阶）](../advanced/tech-detection)
- [pkg/models](./models)
- [证据（进阶）](../advanced/evidence)
- [感知哈希](./phash)
