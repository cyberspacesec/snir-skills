# 网络空间测绘底层库支撑性评估

## 结论

当前库可以作为网络空间测绘系统中的 **Web 资产采集、浏览器渲染截图、Web 指纹与页面证据采集子系统**。

不建议把它单独定义为完整的网络空间测绘底层库。它目前缺少主机发现、端口发现、非 HTTP 协议探测、资产生命周期模型、标准化资产库存储、调度与增量扫描等测绘底座能力。

更准确的定位是：

```text
上游发现层
  域名 / IP / CIDR / 端口 / 证书 / DNS / 队列任务
      |
      v
snir
  HTTP/HTTPS URL 访问
  Chrome 渲染
  截图
  HTML / Header / Cookie / Console / Network / TLS 采集
  Web 技术栈识别
  页面感知哈希
      |
      v
下游资产层
  资产库 / 搜索索引 / 去重聚类 / 风险引擎 / 报告 / 画像
```

## 当前已具备的支撑能力

| 能力域 | 当前状态 | 说明 |
|---|---:|---|
| 单 URL 采集 | 支持 | CLI、SDK、HTTP API 均可触发截图与采集 |
| 批量 URL 采集 | 支持 | 文件、API batch、SDK batch、流式/回调结果 |
| CIDR 输入 | 部分支持 | `scan cidr` 会展开 IP，并可通过 `--ports` 组合常见 Web 端口；仍不做 TCP/UDP 端口发现 |
| HTTP/HTTPS 浏览器渲染 | 支持 | 基于 Chrome DevTools Protocol |
| 截图证据 | 支持 | PNG/JPEG、质量控制、全页/元素截图、内存字节返回 |
| 页面交互 | 支持 | JS 注入、点击、滚动、输入、表单 |
| Web 元数据 | 支持 | HTML、响应头、Cookie、控制台、网络请求、TLS、最终 URL、状态码 |
| Web 技术识别 | 支持 | 内置常见 CMS、框架、服务器、CDN 等指纹规则 |
| 页面去重/聚类 | 支持 | dHash/aHash/pHash 与相似度分组 |
| 并发执行 | 支持 | CLI 线程数、API 并发限制、Chrome 连接池 |
| 代理能力 | 支持 | 代理列表、文件热加载、代理 API、轮换策略 |
| 输出 | 部分支持 | JSONL 保留完整 `models.Result`；SQLite 已保存 endpoint 与主要 Web 证据 JSON；CSV 仍偏基础字段 |
| 安全边界 | 部分支持 | URL 黑名单、默认黑名单、自定义规则；缺少测绘任务级 allowlist/rate budget |

## 关键证据

- `pkg/scan/scan.go` 的 `ScanSingle` 和 `ScanMulti` 以 URL 为核心输入；批量目标可通过 `--ports` 把无协议 host/IP 展开为 scheme + host + port URL。
- `cmd/scan_cidr.go` 会把 CIDR 展开成 IP 列表，然后交给 `ScanMulti`；它不是 TCP/UDP 端口扫描器，也不会识别开放服务。
- `pkg/runner/options.go` 的 `Scan.Ports []int` 已接入 CLI 批量扫描路径，用于 Web endpoint URL 组合。
- `pkg/models/models.go` 的 `Result` 已能承载 `schema_version`、`scheme`、`host`、`port`、`endpoint`、URL、FinalURL、状态码、HTML、Title、pHash、TLS、Technologies、Headers、Network、Console、Cookies 等 Web 采集证据。
- `pkg/runner/writer.go` 的 JSONL Writer 会序列化完整 `models.Result`；CSV Writer 只输出 URL、标题、响应码、截图路径、扫描时间、最终 URL、状态等基础字段。
- `pkg/database/models.go` 和 `pkg/database/database.go` 的 SQLite 持久化会保存 endpoint 字段，并以 JSON 字段保存 TLS、Headers、Network、Console、Cookies、Technologies。

## 与完整测绘底座的差距

| 底座能力 | 当前状态 | 风险 |
|---|---:|---|
| 主机发现 | 缺失 | 不能从网段主动发现存活主机 |
| TCP/UDP 端口扫描 | 缺失 | 不能发现非默认 Web 端口和开放服务面 |
| 非 HTTP 协议探测 | 缺失 | 不能识别 SSH、RDP、Redis、MySQL、MQTT、SMTP 等服务 |
| Banner 抓取 | 缺失 | 非 Web 服务没有证据链 |
| 服务指纹 | 部分缺失 | 目前主要是 Web 技术栈，不是全协议服务指纹 |
| 统一资产模型 | 缺失 | 没有 Host、Endpoint、Service、WebApp、Certificate、Domain 等一等模型 |
| 资产生命周期 | 缺失 | 没有 first_seen、last_seen、scan_id、source、change tracking 等字段约束 |
| 标准化持久化 | 部分缺失 | JSONL 完整，SQLite 已覆盖主要 Web 证据但仍不是资产主库；CSV 不完整 |
| 调度与增量扫描 | 缺失 | 缺少周期任务、断点续扫、差异对比 |
| 分布式任务队列 | 缺失 | 不能直接承担大规模测绘调度 |
| DNS/证书/ASN/地理位置富化 | 缺失 | 资产画像需要外部模块补充 |
| 任务级范围控制 | 部分缺失 | 有黑名单，但缺少 allowlist、速率预算、作用域策略、审计策略 |

## 推荐集成方式

如果上层系统已经有目标发现、端口扫描、任务调度和资产库，当前库可以直接承担 Web 采集执行器角色：

1. 上游把开放 Web 端口转换成 URL，例如 `http://1.2.3.4:8080`、`https://example.com:8443`；也可以传 host/IP + `--ports` 让 snir 组合 Web URL。
2. snir 负责浏览器访问、截图、HTML/Header/Cookie/Console/Network/TLS 采集、技术栈识别和 pHash。
3. 下游优先消费 JSONL 或 SDK/API 返回的完整 `models.Result`。
4. 下游资产库继续归一化 domain、service、first_seen/last_seen、source、scan_id、risk、cert 关联、tech、screenshot 等字段。
5. SQLite 可作为轻量证据库或离线导出，不建议作为测绘主库存储；CSV 只适合基础清单。

## 补齐优先级

### P0：支撑测绘系统底层库的最低补齐项

- Web Endpoint 展开能力：已支持批量 host/IP + ports + schemes 生成 URL；仍需接入 API/SDK 批量辅助能力和更完整的 scope 策略。
- 资产结果模型：已在 `Result` 拆出 `schema_version`、`scheme`、`host`、`port`、`endpoint`；仍缺少一等 `WebAsset`、`ProbeResult`、`source`、`scan_id`。
- Web 证据持久化：SQLite 已以 JSON 字段保存 TLS、Headers、Technologies、Network、Console、Cookies；后续可演进为关系表或资产库适配。
- 结果 schema version：已加入 `schema_version`。
- 增加任务级 scope policy：allowlist、denylist、最大目标数、最大并发、每目标超时、每网段速率。

### P1：成为更完整测绘底座需要补的能力

- TCP connect 或 SYN 扫描模块，先覆盖常见端口发现。
- 非 HTTP 协议 banner/probe 框架。
- DNS 解析、反查、CNAME 链、证书 SAN 提取与域名关联。
- ASN、地理位置、云厂商、CDN/WAF 富化。
- 增量扫描：first_seen、last_seen、last_changed、fingerprint diff。
- 资产去重和归并规则：IP:port、域名、证书、标题、页面 hash、技术栈共同归并。

### P2：大规模平台化能力

- 分布式任务队列和 worker 心跳。
- 分片、断点续扫、失败重试队列。
- 存储适配：PostgreSQL、ClickHouse、Elasticsearch/OpenSearch、对象存储。
- 指标与审计：任务耗时、错误分布、代理质量、截图成功率、采集覆盖率。

## 阶段性判断

当前支撑等级：**适合作为 Web 测绘采集子模块，不足以单独作为完整网络空间测绘底座**。

如果产品目标是“给已有资产发现系统补 Web 截图、页面证据、Web 指纹和聚类”，当前库已经具备较好的基础。

如果产品目标是“从 IP 段/域名种子出发，独立完成全网资产发现、服务识别、画像入库、周期更新和检索”，当前库还需要补齐 P0 和 P1 中的大部分能力。
