# 响应格式

<p align="center">📤 `pkg/api/types.go` + `helpers.go` — 统一响应。</p>

> 📁 源码：[`types.go`](https://github.com/cyberspacesec/snir-skills/blob/main/pkg/api/types.go) · [`helpers.go`](https://github.com/cyberspacesec/snir-skills/blob/main/pkg/api/helpers.go)

## APIResponse

[`APIResponse`](https://github.com/cyberspacesec/snir-skills/blob/main/pkg/api/types.go#L13) 是所有 JSON 响应的统一信封：

| 字段 | 说明 |
|------|------|
| `Success` | 是否成功 |
| `Data` | 数据载荷 |
| `Error` | 错误信息 |
| `Message` | 提示 |

## 示例

成功：

```json
{
  "success": true,
  "data": { "id": "abc123", "url": "https://example.com" },
  "message": "screenshot captured"
}
```

失败：

```json
{
  "success": false,
  "error": "scan timeout",
  "message": "无法在指定时间内完成页面加载"
}
```

## 输出

[`SendJSONResponse`](https://github.com/cyberspacesec/snir-skills/blob/main/pkg/api/helpers.go#L41) 设置状态码与 `Content-Type: application/json`，序列化 `APIResponse` 写回。

## 状态码

| 码 | 场景 |
|----|------|
| 200 | 成功 |
| 400 | 请求参数错误 |
| 401 | 鉴权失败 |
| 429/503 | 并发/队列满 |
| 500 | 内部错误 |

请求到响应的统一处理链路：所有出口都套同一 `APIResponse` 信封，由 `SendJSONResponse` 序列化：

```mermaid
flowchart LR
    Req[HTTP 请求] --> Auth{鉴权?}
    Auth -- 失败 --> E1["401 {success:false,error}"]
    Auth -- 通过 --> Valid{参数校验}
    Valid -- 失败 --> E2["400 {success:false,error}"]
    Valid -- 通过 --> Exec[执行截图]
    Exec -- 并发/队列满 --> E3["429/503 {success:false}"]
    Exec -- 成功 --> Ok["200 {success:true,data}"]
    Exec -- 异常 --> E4["500 {success:false,error}"]
    E1 & E2 & E3 & Ok & E4 --> Wrap[SendJSONResponse<br/>统一信封]
    Wrap --> Resp[JSON 响应]

    style Ok fill:#e6f4ea,stroke:#3aa676
    style E1 fill:#fde8e8,stroke:#d23a3a
    style E2 fill:#fde8e8,stroke:#d23a3a
    style E3 fill:#fde8e8,stroke:#d23a3a
    style E4 fill:#fde8e8,stroke:#d23a3a
    style Wrap fill:#3aa676,stroke:#2a7a56,color:#fff
```

## 下一步

- [请求类型](./request-types)
- [辅助函数](./helpers)
- [POST /screenshot](./endpoint-screenshot)
