# 辅助函数

<p align="center">🧰 `pkg/api/helpers.go` — 小工具。</p>

> 📁 源码：[`pkg/api/helpers.go`](https://github.com/cyberspacesec/snir-skills/blob/main/pkg/api/helpers.go)

## 函数

| 符号 | 源码 | 说明 |
|------|------|------|
| `CreateDir(path)` | [L15](https://github.com/cyberspacesec/snir-skills/blob/main/pkg/api/helpers.go#L15) | 建截图目录 |
| `GetImageContentType(filename)` | [L20](https://github.com/cyberspacesec/snir-skills/blob/main/pkg/api/helpers.go#L20) | 推断图片 MIME |
| `IsImageFile(filename)` | [L35](https://github.com/cyberspacesec/snir-skills/blob/main/pkg/api/helpers.go#L35) | 是否图片 |
| `SendJSONResponse(w, code, resp)` | [L41](https://github.com/cyberspacesec/snir-skills/blob/main/pkg/api/helpers.go#L41) | 统一 JSON 响应 |
| `UrlWithProtocol(url, https, http)` | [L60](https://github.com/cyberspacesec/snir-skills/blob/main/pkg/api/helpers.go#L60) | 补协议 |
| `UrlHasProtocol(url)` | [L82](https://github.com/cyberspacesec/snir-skills/blob/main/pkg/api/helpers.go#L82) | 是否已有协议 |

## SendJSONResponse

[`SendJSONResponse`](https://github.com/cyberspacesec/snir-skills/blob/main/pkg/api/helpers.go#L41) 统一封装 `APIResponse`（成功/失败、数据、消息），所有 handler 都用它输出，保证响应格式一致。

```mermaid
flowchart LR
  H[Handler] --> S[SendJSONResponse]
  S --> AR[APIResponse]
  AR --> W[http.ResponseWriter]
```

## URL 协议处理

`UrlWithProtocol`/`UrlHasProtocol`：用户传 `example.com` 时补 `http://` 或 `https://`，决定探测策略。截图端点 [`ensureProtocol`](https://github.com/cyberspacesec/snir-skills/blob/main/pkg/api/screenshot.go#L296) 用它。

## 下一步

- [响应格式](./response)
- [Server](./server)
- [pkg/islazy](../internals/islazy)
