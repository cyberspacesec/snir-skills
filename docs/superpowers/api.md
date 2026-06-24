# snir api — HTTP API 截图服务

> **渐进式披露**：[快速上手](#快速上手) → [常用选项](#常用选项) → [高级选项](#高级选项) → [完整参数参考](#完整参数参考)

---

## 快速上手

```bash
# 启动 API 服务（默认监听 0.0.0.0:8080）
./snir api

# 指定端口
./snir api --port 9090

# 截图请求
curl -X POST http://localhost:8080/screenshot \
  -H "Content-Type: application/json" \
  -d '{"url": "https://example.com"}'
```

---

## 常用选项

### 监听配置

```bash
# 指定监听地址和端口
./snir api --host 127.0.0.1 --port 9090

# 仅本地访问
./snir api --host 127.0.0.1 --port 8080
```

| 标志 | 默认值 | 说明 |
|------|--------|------|
| `--host` | `0.0.0.0` | API 服务监听地址 |
| `--port` | `8080` | API 服务监听端口 |

### API 鉴权

```bash
# 指定 API Key（鉴权模式）
./snir api --api-key my-secret-key

# 不指定则自动生成，启动时日志中会打印
./snir api
# 输出: API Key: auto-generated-xxxx
```

| 标志 | 默认值 | 说明 |
|------|--------|------|
| `--api-key` | `""` | API 密钥，用于 API 鉴权；不指定则自动生成 |

> **安全提示**：生产环境务必指定 `--api-key`，否则自动生成的 Key 可能暴露在日志中。

---

## 高级选项

### 并发与队列

```bash
# 调整最大并发请求数
./snir api --max-concurrent 20

# 调整请求队列大小（排队等待的截图任务数）
./snir api --queue-size 200
```

| 标志 | 默认值 | 说明 |
|------|--------|------|
| `--max-concurrent` | `10` | 最大并发请求数 |
| `--queue-size` | `100` | 请求队列大小 |

### 远程 Chrome

```bash
# 连接远程 Chrome 实例（避免本地启动浏览器进程）
./snir api --wss ws://chrome-server:9222/devtools/browser/xxx
```

| 标志 | 默认值 | 说明 |
|------|--------|------|
| `--wss` | `""` | 远程 Chrome WebSocket URL |

### 证书

```bash
# 忽略 SSL 证书错误（适用于自签证书环境）
./snir api --ignore-cert-errors
```

| 标志 | 默认值 | 说明 |
|------|--------|------|
| `--ignore-cert-errors` | `false` | 忽略证书错误 |

### 黑名单

```bash
# 禁用黑名单
./snir api --enable-blacklist=false

# 自定义黑名单规则
./snir api --blacklist-pattern ".*\.jpg$" --blacklist-pattern ".*\.png$"

# 使用黑名单文件
./snir api --blacklist-file blacklist.txt
```

| 标志 | 默认值 | 说明 |
|------|--------|------|
| `--enable-blacklist` | `true` | 启用 URL 黑名单检查 |
| `--default-blacklist` | `true` | 使用默认黑名单规则 |
| `--blacklist-pattern` | `[]` | 自定义黑名单规则（可多次使用） |
| `--blacklist-file` | `""` | 黑名单规则文件路径 |

---

## 完整参数参考

| 标志 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `--host` | string | `0.0.0.0` | API 服务监听地址 |
| `--port` | int | `8080` | API 服务监听端口 |
| `--api-key` | string | `""` | API 密钥（不指定则自动生成） |
| `--max-concurrent` | int | `10` | 最大并发请求数 |
| `--queue-size` | int | `100` | 请求队列大小 |
| `--wss` | string | `""` | 远程 Chrome WebSocket URL |
| `--ignore-cert-errors` | bool | `false` | 忽略证书错误 |
| `--enable-blacklist` | bool | `true` | 启用 URL 黑名单检查 |
| `--default-blacklist` | bool | `true` | 使用默认黑名单规则 |
| `--blacklist-pattern` | stringSlice | `[]` | 自定义黑名单规则 |
| `--blacklist-file` | string | `""` | 黑名单规则文件路径 |

---

## API 端点一览

| 端点 | 方法 | 说明 |
|------|------|------|
| `/` | GET | API 信息 |
| `/screenshot` | POST | 单张截图 |
| `/batch` | POST | 批量截图 |
| `/screenshots_list` | GET | 列出截图文件 |
| `/get_screenshot/{filename}` | GET | 获取截图文件 |
| `/screenshots/{path}` | GET | 静态文件服务 |
| `/stats` | GET | 统计信息（含连接池） |
| `/health` | GET | 健康检查 |

### 截图请求示例

```bash
# 单张截图
curl -X POST http://localhost:8080/screenshot \
  -H "X-API-Key: your-api-key" \
  -H "Content-Type: application/json" \
  -d '{
    "url": "https://example.com",
    "device": "iphone-15",
    "timeout": 30,
    "delay": 0,
    "user_agent": "",
    "proxy": "",
    "ignore_cert_errors": false,
    "screenshot_format": "jpeg",
    "screenshot_quality": 85,
    "skip_save": false,
    "save_html": true,
    "save_headers": true,
    "save_console": true,
    "save_cookies": true,
    "save_network": true,
    "javascript": "",
    "selector": "",
    "xpath": "",
    "capture_full_page": false,
    "https": true,
    "http": true
  }'

# 批量截图
curl -X POST http://localhost:8080/batch \
  -H "X-API-Key: your-api-key" \
  -H "Content-Type: application/json" \
  -d '{
    "urls": ["https://a.com", "https://b.com"],
    "device": "pixel-8-pro",
    "threads": 4,
    "timeout": 30,
    "screenshot_format": "png",
    "skip_save": true
  }'

# 查看统计
curl http://localhost:8080/stats -H "X-API-Key: your-api-key"

# 健康检查
curl http://localhost:8080/health
```

### 截图请求字段

| 字段 | 类型 | 说明 |
|------|------|------|
| `url` | string | 单张截图 URL |
| `urls` | []string | 批量截图 URL 列表 |
| `device` | string | 设备预设名称，例如 `iphone-15`、`pixel-8-pro`、`ipad-pro-12`、`desktop-1080p` |
| `timeout` | int | 页面加载超时时间（秒） |
| `delay` | int | 截图前等待时间（秒） |
| `user_agent` | string | 自定义 User-Agent |
| `proxy` | string | 代理服务器地址 |
| `ignore_cert_errors` | bool | 忽略证书错误 |
| `screenshot_format` | string | 截图格式，`png` 或 `jpeg` |
| `screenshot_quality` | int | JPEG 质量，1-100，仅对 `jpeg` 有效 |
| `skip_save` | bool | 不写截图文件，仅返回截图结果元数据 |
| `selector` | string | CSS 选择器元素截图 |
| `xpath` | string | XPath 元素截图 |
| `capture_full_page` | bool | 全页截图 |
| `javascript` / `javascript_file` | string | 截图前执行 JS 或加载 JS 文件 |
| `run_js_before` | bool | 页面加载前执行 JS |
| `save_html` / `save_headers` / `save_console` / `save_cookies` / `save_network` | bool | 保存对应采集数据 |
| `fingerprint` | object | 浏览器指纹覆盖，可设置 UA、语言、平台、WebGL、屏幕尺寸等 |
| `cookies` / `cookie_file` / `cookie_import` / `cookie_header` | mixed | Cookie 注入或导入 |
| `actions` / `form` | mixed | 点击、滚动、输入、等待、表单填写等交互 |

`device` 会在导航前通过 CDP 模拟 viewport、DPR、mobile/touch 和 User-Agent。`fingerprint` 中显式传入的非空字段会覆盖设备预设对应字段。

---

## 实战组合示例

```bash
# 生产级部署：指定 Key + 远程 Chrome + 高并发 + 黑名单
./snir api \
  --host 0.0.0.0 \
  --port 8080 \
  --api-key production-secret-key \
  --wss ws://chrome-pool:9222/devtools/browser/xxx \
  --max-concurrent 20 \
  --queue-size 200 \
  --blacklist-file /etc/snir/blacklist.txt

# 开发调试：本地访问 + 自动 Key
./snir api --host 127.0.0.1 --port 9090 -D

# 容器化部署：远程 Chrome + 忽略证书
./snir api \
  --host 0.0.0.0 \
  --port 8080 \
  --wss ws://chrome:9222/devtools/browser/xxx \
  --ignore-cert-errors
```
