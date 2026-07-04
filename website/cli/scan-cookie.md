# Cookie 选项

<p align="center">🍪 管理 Cookie：内联、持久化、导入导出。</p>

## 标志

| 标志 | 说明 |
|------|------|
| `--cookie` | 内联 Cookie（`name=value`，可多次使用） |
| `--cookie-file` | Cookie 持久化文件（JSON，跨请求复用） |
| `--cookie-write-back` | 截图后把浏览器 Cookie 写回 `--cookie-file` |
| `--cookie-export` | 截图后导出 Cookie 到文件（Netscape 格式） |
| `--cookie-import` | 导入 Netscape 格式 Cookie 文件（curl/wget 格式） |

## 示例

```bash
# 内联 Cookie
snir scan example.com --cookie "session=abc123" --cookie "token=xyz"

# 持久化 Cookie 文件（JSON）
snir scan example.com --cookie-file cookies.json --cookie-write-back

# 导入 Netscape（如从 curl 导出）
snir scan example.com --cookie-import cookies.txt

# 导出为 Netscape（供 curl/wget 用）
snir scan example.com --cookie-export out.txt
```

## 工作流

```mermaid
flowchart LR
  I[导入] --> B[浏览器]
  B --> S[截图+采集]
  S --> W[写回 cookie-file]
  S --> E[导出 Netscape]
```

Cookie 从外部导入、注入浏览器、采集后再写回/导出的完整时序：

```mermaid
sequenceDiagram
  participant F as 外部 Cookie 源
  participant CLI as snir scan
  participant JAR as CookieJar
  participant CH as Chrome
  participant OUT as 输出文件
  F->>CLI: --cookie / --cookie-file / --cookie-import
  CLI->>JAR: 加载 Cookie
  JAR->>CH: 注入到浏览器上下文
  CH->>CH: 导航 + 截图 + 采集
  CH-->>JAR: 会话中新增/变更的 Cookie
  opt --cookie-write-back
  JAR->>OUT: 写回 cookie-file（JSON）
  end
  opt --cookie-export
  JAR->>OUT: 导出 Netscape cookies.txt
  end
  opt --save-cookies
  CH-->>CLI: Result.cookies 作为证据
  end
```

## Cookie 持久化文件（JSON）

`--cookie-file` 指定一个 JSON 文件，`CookieJar` 跨请求复用 Cookie。配合 `--cookie-write-back`，截图后把浏览器实际 Cookie（含登录态）写回，下次扫描自动带上。

## Netscape 格式

兼容 curl/wget 的 `cookies.txt` 格式：

- `--cookie-import`：从 curl/wget 导出登录态，让 snir 带登录截图
- `--cookie-export`：把 snir 采集到的 Cookie 导出，供 curl/wget 复用

## 与证据采集的区别

::: warning 别混淆：注入 Cookie ≠ 采集 Cookie
| 标志 | 作用 | 方向 |
|------|------|------|
| `--cookie` / `--cookie-file` / `--cookie-import` | **注入** Cookie 维持会话 | 外部 → 浏览器 |
| `--save-cookies` | 把浏览器 Cookie 作为**证据**存入 `Result.cookies` | 浏览器 → 结果 |

两者可同用：先 `--cookie-import` 带登录态，再 `--save-cookies` 把实际 Cookie 采下来。
:::

## 适用场景

::: tip 两个经典会话配方
- 🔐 **登录后截图**：先用 `curl` 登录导出 Netscape，再 `snir scan --cookie-import cookies.txt`
- 🔁 **跨任务复用会话**：`--cookie-file cookies.json --cookie-write-back`，每次截图后写回，下次自动带
:::

详见 [Cookie 管理（进阶）](../advanced/cookie)。

## 下一步

- [证据选项](./scan-evidence)
- [Cookie 管理（进阶）](../advanced/cookie)
- [内部 CookieJar](../internals/runner-cookie-jar)
