# GET /stats

<p align="center">📊 运行统计端点。</p>

> 📁 源码：[`pkg/api/server.go`](https://github.com/cyberspacesec/snir-skills/blob/main/pkg/api/server.go)

## Handler

| 符号 | 源码 | 说明 |
|------|------|------|
| `HandleStats` | [L156](https://github.com/cyberspacesec/snir-skills/blob/main/pkg/api/server.go#L156) | `GET /stats` |
| `GetConcurrencyStats()` | [L98](https://github.com/cyberspacesec/snir-skills/blob/main/pkg/api/server.go#L98) | 取统计 |

## 响应

返回并发与池负载：

```json
{
  "success": true,
  "data": {
    "active": 3,
    "waiting": 0,
    "max": 10,
    "queue": 100,
    "uptime": 3600
  }
}
```

| 字段 | 说明 |
|------|------|
| `active` | 执行中请求数 |
| `waiting` | 排队数 |
| `max` | 并发上限 |
| `queue` | 队列容量 |
| `uptime` | 运行时长（秒） |

## 监控

```mermaid
flowchart LR
  PR[Prometheus] --> S[GET /stats]
  S --> M[指标]
  M --> GR[Grafana 面板]
```

::: tip active 持续接近 max = 该扩容了
定期抓 `/stats`，若 `active` 长期贴近 `max`、`waiting` 常年 > 0，说明 Chrome 池是瓶颈——要么调大 `--max-concurrent`（同时扩 Chrome 池），要么上 `provider` 多机分担。可接 Prometheus 抓取做容量规划与告警。
:::

## 下一步

- [并发限流](./concurrency)
- [GET /health](./endpoint-health)
- [监控](../guide/monitoring)
- [性能调优](../advanced/performance)
