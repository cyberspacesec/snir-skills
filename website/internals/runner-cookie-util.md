# Cookie 工具

<p align="center">🔧 `pkg/runner/cookie_util.go` — Cookie 字符串解析。</p>

在 HTTP 头字符串、`CustomCookie` 结构、注入用 header 之间转换。

> 📁 源码：[`pkg/runner/cookie_util.go`](https://github.com/cyberspacesec/snir-skills/blob/main/pkg/runner/cookie_util.go)

## 函数

| 符号 | 源码 | 说明 |
|------|------|------|
| `ParseCookieHeader(headerStr, defaultDomain)` | [L17](https://github.com/cyberspacesec/snir-skills/blob/main/pkg/runner/cookie_util.go#L17) | 解析 `Cookie:` 请求头 |
| `ParseSetCookieHeaders(headers, defaultDomain)` | [L64](https://github.com/cyberspacesec/snir-skills/blob/main/pkg/runner/cookie_util.go#L64) | 解析 `Set-Cookie` 响应头 |
| `CustomCookiesToHeaderString(cookies)` | [L133](https://github.com/cyberspacesec/snir-skills/blob/main/pkg/runner/cookie_util.go#L133) | 拼回 `name=val; ...` |

## 转换关系

```mermaid
flowchart LR
  H["Cookie: a=1; b=2"] --> PCH[ParseCookieHeader]
  PCH --> CC[[]CustomCookie]
  SC["Set-Cookie: c=3; Path=/"] --> PSC[ParseSetCookieHeaders]
  PSC --> CC
  CC --> CTH[CustomCookiesToHeaderString]
  CTH --> H2["a=1; b=2; c=3"]
  H2 --> INJ[注入请求]
```

从 Cookie 字符串解析到注入浏览器请求的时序：

```mermaid
sequenceDiagram
  participant U as 调用方
  participant PC as ParseCookieHeader
  participant CC as []CustomCookie
  participant CTH as CustomCookiesToHeaderString
  participant CH as Chrome
  U->>PC: "Cookie: a=1; b=2"
  PC-->>CC: []CustomCookie
  U->>CTH: cookies[]
  CTH-->>U: "a=1; b=2"
  U->>CH: 导航时注入 Cookie 头
  CH-->>U: 带会话的页面
```

## 区别

| 函数 | 输入 | 特点 |
|------|------|------|
| `ParseCookieHeader` | 单个 `Cookie:` 头 | 名值对，无属性 |
| `ParseSetCookieHeaders` | 多个 `Set-Cookie:` 头 | 含 Path/Domain/Secure 等属性 |

`defaultDomain` 用于无 Domain 属性时回填当前目标域名。

## 下一步

- [CookieJar](./runner-cookie-jar)
- [Netscape Cookie](./runner-cookie-netscape)
- [Cookie（进阶）](../advanced/cookie)
- [CLI scan cookie](../cli/scan-cookie)
