# pkg/ascii

<p align="center">🎨 `pkg/ascii/ascii.go` — 终端艺术与渲染。</p>

提供 ASCII logo、版本信息横幅，并把 Markdown 渲染为终端彩色文本（基于 glamour）。

> 📁 源码：[`pkg/ascii/ascii.go`](https://github.com/cyberspacesec/snir-skills/blob/main/pkg/ascii/ascii.go)

## 函数

| 符号 | 源码 | 说明 |
|------|------|------|
| `Logo()` | [L26](https://github.com/cyberspacesec/snir-skills/blob/main/pkg/ascii/ascii.go#L26) | 返回 ASCII logo |
| `VersionInfo()` | [L59](https://github.com/cyberspacesec/snir-skills/blob/main/pkg/ascii/ascii.go#L59) | 版本横幅 |
| `Markdown(markdown)` | [L73](https://github.com/cyberspacesec/snir-skills/blob/main/pkg/ascii/ascii.go#L73) | Markdown→终端渲染 |

## Logo

[`Logo`](https://github.com/cyberspacesec/snir-skills/blob/main/pkg/ascii/ascii.go#L26) 返回带颜色的 snir ASCII 艺术，CLI 启动与 `version` 命令展示。

```
   _____            _    _   _____
  / ___/____  _____| |  | | / ___/  ____
  \__ \/ __ \/ ___/ |  | | \__ \  / __ \
 ___/ / /_/ / /  | |  | |___/ /_/ /_/ /
/____/\____/_/   |_|  |_/____/\____/
```

## VersionInfo

[`VersionInfo`](https://github.com/cyberspacesec/snir-skills/blob/main/pkg/ascii/ascii.go#L59)：组合 Logo + 版本号 + 构建信息，`snir version` 输出。

## Markdown 渲染

[`Markdown`](https://github.com/cyberspacesec/snir-skills/blob/main/pkg/ascii/ascii.go#L73) 用 glamour 把 Markdown 字符串渲染为带样式的终端输出，长帮助、README 摘要可美观展示。

```mermaid
flowchart LR
  MD[Markdown 字符串] --> GL[glamour.Render]
  GL --> T[彩色终端文本]
```

## 下一步

- [pkg/log](./log)
- [CLI version](../cli/version)
- [pkg/islazy](./islazy)
