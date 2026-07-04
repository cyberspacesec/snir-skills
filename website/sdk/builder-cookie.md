# Cookie 构建

<p align="center">🍪 SDK 注入与管理 Cookie。</p>

## 选项

| 选项 | 说明 |
|------|------|
| `WithInjectedCookies(cookies...)` | 注入 `CustomCookie` |
| `WithCookieHeader(header)` | `name=value; name2=value2` 格式 |
| `WithCookieStrings(headers...)` | 多个 `name=value` |
| `WithCookieImport(path)` | 导入 Netscape 文件 |
| `WithCookieExport(path)` | 导出 Netscape |
| `WithCookieFile(path)` | Cookie 持久化文件（JSON） |
| `WithCookieWriteBack()` | 截图后写回 |

## CustomCookie

`runner.CustomCookie` 是注入用的 Cookie 结构（含 name/value/domain/path 等）。

## 示例

```go
// 注入
opts := sdk.NewScreenshotOptions(
    sdk.WithInjectedCookies(
        runner.CustomCookie{Name: "session", Value: "abc123", Domain: "example.com"},
    ),
)

// Header 格式
opts := sdk.NewScreenshotOptions(
    sdk.WithCookieHeader("session=abc123; token=xyz"),
)

// 持久化 + 写回
opts := sdk.NewScreenshotOptions(
    sdk.WithCookieFile("cookies.json"),
    sdk.WithCookieWriteBack(),
)

// 导入 Netscape（curl 登录态）
opts := sdk.NewScreenshotOptions(
    sdk.WithCookieImport("login.txt"),
)
```

## 工作流

```mermaid
flowchart LR
  I[注入/导入] --> B[浏览器]
  B --> S[截图]
  S --> W[写回 cookie-file]
  S --> E[导出 Netscape]
```

## 与证据采集区别

::: warning 注入 ≠ 采集，方向相反
| 类别 | 选项 | 方向 | 作用 |
|------|------|------|------|
| **注入** | `WithCookie*` / `WithCookieImport` | 外部 → 浏览器 | 设会话 Cookie 维持登录态 |
| **采集** | `WithCookies()` | 浏览器 → Result | 把浏览器 Cookie 存为证据 |

可同用：`WithCookieImport` 带登录态截图，再 `WithCookies()` 把实际 Cookie 采下来。
:::

详见 [Cookie 管理（进阶）](../advanced/cookie)。

## Cookie 生命周期状态

一张 Cookie 从外部来源进入浏览器、被使用、再到写回/导出，经历以下状态：

```mermaid
stateDiagram-v2
    [*] --> 导入: WithCookieImport / WithCookieHeader
    [*] --> 注入: WithInjectedCookies
    [*] --> 持久: WithCookieFile
    导入 --> 解析: Netscape/Header 解析
    注入 --> 解析: CustomCookie 结构
    持久 --> 加载: 读取 JSON 文件
    解析 --> 浏览器: 注入到 CookieJar
    加载 --> 浏览器: 注入到 CookieJar
    浏览器 --> 使用: 导航携带 Cookie
    使用 --> 截图: 维持登录态访问
    截图 --> 写回: WithCookieWriteBack
    截图 --> 导出: WithCookieExport
    写回 --> 持久: 更新 cookie-file
    导出 --> [*]: 生成 Netscape 文件
    使用 --> 浏览器: 后续请求继续携带
```

注：**注入**（外部→浏览器）与**采集**（浏览器→Result，`WithCookies()`）方向相反，可同用。

## 下一步

- [构建器总览](./builders)
- [Cookie 管理（进阶）](../advanced/cookie)
- [CookieJar 内部](../internals/runner-cookie-jar)
