# snir scan — 网页截图与信息收集

> **渐进式披露**：[快速上手](#快速上手) → [常用选项](#常用选项) → [高级选项](#高级选项) → [完整参数参考](#完整参数参考)

---

## 快速上手

```bash
# 最简单的用法 — 直接传 URL
./snir scan example.com

# 等价写法
./snir scan single example.com

# 批量扫描 — 从文件读取 URL 列表
./snir scan file -f urls.txt

# 从 host/IP 列表按协议和端口展开 URL
./snir scan file -f hosts.txt --ports 80,443,8080,8443

# 网段扫描 — 自动展开 CIDR
./snir scan cidr 192.168.1.0/24

# 网段扫描常见 Web 端口
./snir scan cidr 192.168.1.0/24 --ports 80,443,8080,8443
```

---

## 子命令概览

| 子命令 | 用法 | 说明 |
|--------|------|------|
| *(直接传 URL)* | `snir scan <url>` | 单 URL 截图（快捷方式） |
| `single` | `snir scan single <url>` | 单 URL 截图（显式子命令） |
| `cidr` | `snir scan cidr <cidr>` | 按网段批量截图 |
| `file` | `snir scan file -f <path>` | 从文件读取 URL 批量截图 |

> **提示**：`snir scan <url>` 和 `snir scan single <url>` 效果完全相同。直接传 URL 是最便捷的用法。

---

## 常用选项

### 超时与延迟

```bash
# 页面加载超时（默认 30 秒），适用于慢加载网站
./snir scan example.com --timeout 60

# 截图前等待时间（默认 0 秒），确保动态内容加载完成
./snir scan example.com --delay 3

# 组合使用
./snir scan slow-site.com --timeout 60 --delay 5
```

| 标志 | 默认值 | 说明 |
|------|--------|------|
| `--timeout` | `30` | 页面加载超时时间（秒） |
| `--delay` | `0` | 截图前等待时间（秒） |

### 分辨率

```bash
# 高清截图
./snir scan example.com --resolution-x 1920 --resolution-y 1080

# 移动端模拟
./snir scan example.com --resolution-x 375 --resolution-y 812
```

| 标志 | 默认值 | 说明 |
|------|--------|------|
| `--resolution-x` | `1280` | 浏览器窗口宽度（像素） |
| `--resolution-y` | `800` | 浏览器窗口高度（像素） |

### 代理

```bash
# HTTP 代理
./snir scan example.com --proxy http://127.0.0.1:8080

# SOCKS5 代理
./snir scan example.com --proxy socks5://127.0.0.1:1080
```

| 标志 | 默认值 | 说明 |
|------|--------|------|
| `--proxy` | `""` | 代理服务器地址 |

### 数据收集

```bash
# 保存 HTML 源码
./snir scan example.com --save-html

# 保存 HTTP 响应头
./snir scan example.com --save-headers

# 保存 Cookie
./snir scan example.com --save-cookies

# 保存控制台日志
./snir scan example.com --save-console

# 保存网络请求日志
./snir scan example.com --save-network

# 全部收集
./snir scan example.com --save-html --save-headers --save-cookies --save-console --save-network
```

| 标志 | 默认值 | 说明 |
|------|--------|------|
| `--save-html` | `false` | 保存网页 HTML 内容 |
| `--save-headers` | `false` | 保存 HTTP 响应头 |
| `--save-cookies` | `false` | 保存 Cookie |
| `--save-console` | `false` | 保存控制台日志 |
| `--save-network` | `false` | 保存网络请求日志 |

### 输出格式

```bash
# JSONL 格式
./snir scan file -f urls.txt --write-jsonl --jsonl-file results.jsonl

# CSV 格式
./snir scan file -f urls.txt --write-csv --csv-file results.csv

# 存入数据库
./snir scan file -f urls.txt --db --db-path screenshots.db

# 禁用控制台输出（只写文件）
./snir scan file -f urls.txt --write-jsonl --write-stdout=false
```

| 标志 | 默认值 | 说明 |
|------|--------|------|
| `--write-jsonl` | `false` | 写入 JSONL 格式结果 |
| `--jsonl-file` | `results.jsonl` | JSONL 文件路径 |
| `--write-csv` | `false` | 写入 CSV 格式结果 |
| `--csv-file` | `results.csv` | CSV 文件路径 |
| `--write-stdout` | `true` | 输出结果到控制台 |
| `--db` | `false` | 启用数据库存储 |
| `--db-path` | `go-web-screenshot.db` | 数据库文件路径 |

---

## 高级选项

### 截图控制

```bash
# 全页截图（包括滚动区域）
./snir scan example.com --full-page

# CSS 选择器截图 — 仅截取匹配元素
./snir scan example.com --selector "#main-content"

# XPath 截图 — 仅截取匹配元素
./snir scan example.com --xpath "//div[@class='chart']"

# JPEG 格式 + 质量
./snir scan example.com --screenshot-format jpeg --screenshot-quality 85

# 自定义截图保存路径
./snir scan example.com --screenshot-path ./output/screenshots

# 跳过保存截图（仅收集信息，不写图片文件）
./snir scan example.com --skip-screenshot --save-html
```

| 标志 | 默认值 | 说明 |
|------|--------|------|
| `--full-page` | `false` | 截取完整页面（包括滚动区域） |
| `--selector` | `""` | CSS 选择器截图（仅截取匹配元素） |
| `--xpath` | `""` | XPath 截图（仅截取匹配元素） |
| `--screenshot-format` | `png` | 截图格式（`png` 或 `jpeg`） |
| `--screenshot-quality` | `90` | JPEG 截图质量（1-100，仅对 jpeg 有效） |
| `--screenshot-path` | `screenshots` | 截图保存路径 |
| `--skip-screenshot` | `false` | 跳过保存截图文件 |

### 设备预设

```bash
# 移动端截图，模拟 viewport、DPR、mobile/touch 和 User-Agent
./snir scan example.com --device iphone-15

# Android 设备预设
./snir scan example.com --device pixel-8-pro

# 桌面设备预设
./snir scan example.com --device desktop-1080p

# 列出所有可用设备预设
./snir scan --list-devices
```

| 标志 | 默认值 | 说明 |
|------|--------|------|
| `--device` | `""` | 设备预设名称，例如 `iphone-15`、`pixel-8-pro`、`ipad-pro-12`、`desktop-1080p` |
| `--list-devices` | `false` | 列出可用设备预设并退出 |

设备预设会在页面导航前通过 CDP 模拟窗口大小、device scale factor、mobile/touch 能力，并设置对应 User-Agent。CLI 中使用 `--device` 时，以设备预设中的窗口尺寸和 User-Agent 为准。

### JavaScript 执行

```bash
# 截图前执行 JavaScript（如移除弹窗）
./snir scan example.com --js "document.querySelectorAll('.popup').forEach(el => el.remove());"

# 从文件加载 JavaScript
./snir scan example.com --js-file inject.js

# 在页面加载前执行 JavaScript（注入早期脚本）
./snir scan example.com --js "window.__test = true" --run-js-before
```

| 标志 | 默认值 | 说明 |
|------|--------|------|
| `--js` | `""` | 在页面上执行的 JavaScript 代码 |
| `--js-file` | `""` | 包含 JavaScript 代码的文件路径 |
| `--run-js-before` | `false` | 在页面加载前执行 JavaScript（而非加载后） |

### Chrome 浏览器控制

```bash
# 自定义 Chrome 路径（适用于非标准安装位置）
./snir scan example.com --chrome-path /opt/google/chrome/chrome

# 自定义 User-Agent
./snir scan example.com --user-agent "Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/120.0.0.0"

# 连接远程 Chrome（避免本地启动，适合容器化部署）
./snir scan example.com --wss ws://chrome-server:9222/devtools/browser/xxx

# 忽略 SSL 证书错误（HTTPS 站点自签证书）
./snir scan example.com --ignore-cert-errors

# 非无头模式（显示浏览器界面，调试用）
./snir scan example.com --headless=false
```

| 标志 | 默认值 | 说明 |
|------|--------|------|
| `--chrome-path` | `""` | Chrome 可执行文件路径 |
| `--user-agent` | `""` | 自定义 User-Agent |
| `--wss` | `""` | 远程 Chrome WebSocket URL |
| `--ignore-cert-errors` | `false` | 忽略 SSL 证书错误 |
| `--headless` | `true` | 使用无头模式（`--headless=false` 显示浏览器界面） |

### 代理轮换（高级代理策略）

```bash
# 代理列表 — 多个代理轮换使用
./snir scan file -f urls.txt \
  --proxy-list http://proxy1:8080 \
  --proxy-list http://proxy2:8080 \
  --proxy-list http://proxy3:8080

# 代理文件 — 每行一个代理，支持热加载（文件修改后自动生效）
./snir scan file -f urls.txt --proxy-file proxies.txt

# 代理 API — 每次请求从 API 获取新代理
./snir scan file -f urls.txt --proxy-url http://proxy-api.example.com/get

# 代理轮换策略（默认 round-robin）
./snir scan file -f urls.txt \
  --proxy-list http://proxy1:8080 \
  --proxy-list http://proxy2:8080 \
  --proxy-strategy random

# 组合使用 — 代理列表 + 文件 + API 同时生效
./snir scan file -f urls.txt \
  --proxy-list http://proxy1:8080 \
  --proxy-file proxies.txt \
  --proxy-url http://proxy-api.example.com/get
```

| 标志 | 默认值 | 说明 |
|------|--------|------|
| `--proxy-list` | `[]` | 代理列表（可多次使用，轮换） |
| `--proxy-file` | `""` | 代理文件路径（每行一个代理，支持热加载） |
| `--proxy-url` | `""` | 代理 API URL（每次获取新代理） |
| `--proxy-strategy` | `round-robin` | 代理轮换策略：`round-robin` / `random` / `sequential` |

**代理策略说明：**

| 策略 | 行为 |
|------|------|
| `round-robin` | 按顺序循环使用每个代理（默认） |
| `random` | 随机选择代理 |
| `sequential` | 顺序使用，当前代理失败时切换到下一个 |

本地 Chrome 模式下，代理是 Chrome 进程级配置，工具会为不同代理隔离浏览器进程，避免请求间串用代理。远程 Chrome（`--wss`）无法在连接后为单个请求切换进程级代理，需在远程 Chrome 启动时配置代理。

### Cookie 管理

```bash
# 加载 Cookie 文件（JSON 格式，跨请求复用）
./snir scan example.com --cookie-file cookies.json

# 截图后将浏览器 Cookie 写回文件
./snir scan example.com --cookie-file cookies.json --cookie-write-back

# 导入 Netscape 格式 Cookie 文件
./snir scan example.com --cookie-import cookies.txt

# 导出 Cookie 为 Netscape 格式
./snir scan example.com --cookie-export exported-cookies.txt

# 内联 Cookie（可多次使用）
./snir scan example.com --cookie "session=abc123" --cookie "token=xyz789"
```

| 标志 | 默认值 | 说明 |
|------|--------|------|
| `--cookie-file` | `""` | Cookie 持久化文件路径（JSON 格式，跨请求复用） |
| `--cookie-write-back` | `false` | 截图后将浏览器 Cookie 写回 cookie-file |
| `--cookie-export` | `""` | 截图后导出 Cookie 到文件（Netscape 格式） |
| `--cookie-import` | `""` | 导入 Netscape 格式 Cookie 文件 |
| `--cookie` | `[]` | 内联 Cookie（`name=value` 格式，可多次使用） |

### 并发与重试

```bash
# 调整并发线程数（默认 2）
./snir scan file -f urls.txt --threads 10

# 最大重试次数（默认 1）
./snir scan file -f urls.txt --max-retries 3
```

| 标志 | 默认值 | 说明 |
|------|--------|------|
| `--threads` | `2` | 并发线程数 |
| `--max-retries` | `1` | 最大重试次数 |

### 协议控制

```bash
# 只扫描 HTTPS
./snir scan cidr 192.168.1.0/24 --http=false

# 只扫描 HTTP
./snir scan cidr 192.168.1.0/24 --https=false

# 同时扫描 HTTP 和 HTTPS（默认行为）
./snir scan cidr 192.168.1.0/24

# 对裸 host/IP 按端口展开；已有协议的 URL 会保持原样
./snir scan file -f hosts.txt --ports 80,443,8080,8443
```

| 标志 | 默认值 | 说明 |
|------|--------|------|
| `--http` | `true` | 使用 HTTP 协议 |
| `--https` | `true` | 使用 HTTPS 协议 |
| `--ports` | `[]` | 对无协议 host/IP 目标展开端口列表 |

`--ports` 只负责 URL 组合，不是 TCP/UDP 端口发现器。输入 `example.com/admin` 会展开成类似 `https://example.com:8443/admin`；输入 `https://example.com:9443/path` 会保持原样；输入 `example.com:9443/path` 会只补协议并保留显式端口。

### 黑名单

```bash
# 禁用黑名单检查
./snir scan file -f urls.txt --enable-blacklist=false

# 不使用默认黑名单规则
./snir scan file -f urls.txt --default-blacklist=false

# 添加自定义黑名单规则（可多次使用）
./snir scan file -f urls.txt --blacklist-pattern ".*\.jpg$" --blacklist-pattern ".*\.png$"

# 使用黑名单规则文件
./snir scan file -f urls.txt --blacklist-file blacklist.txt
```

| 标志 | 默认值 | 说明 |
|------|--------|------|
| `--enable-blacklist` | `true` | 启用 URL 黑名单检查 |
| `--default-blacklist` | `true` | 使用默认黑名单规则 |
| `--blacklist-pattern` | `[]` | 添加自定义黑名单规则（可多次使用） |
| `--blacklist-file` | `""` | 黑名单规则文件路径 |

---

## 完整参数参考

### 截图选项

| 标志 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `--screenshot-path` | string | `screenshots` | 截图保存路径 |
| `--screenshot-format` | string | `png` | 截图格式（`png` 或 `jpeg`） |
| `--screenshot-quality` | int | `90` | JPEG 截图质量（1-100，仅对 jpeg 有效） |
| `--skip-screenshot` | bool | `false` | 跳过保存截图 |
| `--full-page` | bool | `false` | 截取完整页面（包括滚动区域） |
| `--selector` | string | `""` | CSS 选择器截图 |
| `--xpath` | string | `""` | XPath 截图 |
| `--device` | string | `""` | 设备预设名称，例如 `iphone-15`、`pixel-8-pro` |
| `--list-devices` | bool | `false` | 列出可用设备预设并退出 |

### Chrome 选项

| 标志 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `--chrome-path` | string | `""` | Chrome 可执行文件路径 |
| `--user-agent` | string | `""` | 自定义 User-Agent |
| `--proxy` | string | `""` | 代理服务器地址 |
| `--timeout` | int | `30` | 页面加载超时时间（秒） |
| `--delay` | int | `0` | 截图前等待时间（秒） |
| `--resolution-x` | int | `1280` | 浏览器窗口宽度 |
| `--resolution-y` | int | `800` | 浏览器窗口高度 |
| `--headless` | bool | `true` | 使用无头模式 |
| `--ignore-cert-errors` | bool | `false` | 忽略 SSL 证书错误 |
| `--wss` | string | `""` | 远程 Chrome WebSocket URL |
| `--proxy-list` | stringSlice | `[]` | 代理列表（可多次使用） |
| `--proxy-file` | string | `""` | 代理文件路径 |
| `--proxy-url` | string | `""` | 代理 API URL |
| `--proxy-strategy` | string | `round-robin` | 代理轮换策略（`round-robin`/`random`/`sequential`） |

### 扫描选项

| 标志 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `--threads` | int | `2` | 并发线程数 |
| `--http` | bool | `true` | 使用 HTTP 协议 |
| `--https` | bool | `true` | 使用 HTTPS 协议 |
| `--ports` | intSlice | `[]` | 对无协议 host/IP 目标展开端口列表 |
| `--max-retries` | int | `1` | 最大重试次数 |
| `--js` | string | `""` | 在页面上执行的 JavaScript |
| `--js-file` | string | `""` | JavaScript 文件路径 |
| `--run-js-before` | bool | `false` | 在页面加载前执行 JS |
| `--selector` | string | `""` | CSS 选择器截图 |
| `--xpath` | string | `""` | XPath 截图 |
| `--full-page` | bool | `false` | 全页截图 |
| `--save-network` | bool | `false` | 保存网络请求日志 |

### Cookie 选项

| 标志 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `--cookie-file` | string | `""` | Cookie 持久化文件路径（JSON 格式） |
| `--cookie-write-back` | bool | `false` | 截图后将浏览器 Cookie 写回 cookie-file |
| `--cookie-export` | string | `""` | 截图后导出 Cookie 到文件（Netscape 格式） |
| `--cookie-import` | string | `""` | 导入 Netscape 格式 Cookie 文件 |
| `--cookie` | stringArray | `[]` | 内联 Cookie（`name=value`，可多次使用） |

### 数据收集选项

| 标志 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `--save-html` | bool | `false` | 保存 HTML 源码 |
| `--save-headers` | bool | `false` | 保存 HTTP 响应头 |
| `--save-console` | bool | `false` | 保存控制台日志 |
| `--save-cookies` | bool | `false` | 保存 Cookie |
| `--save-network` | bool | `false` | 保存网络请求日志 |

### 黑名单选项

| 标志 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `--enable-blacklist` | bool | `true` | 启用 URL 黑名单检查 |
| `--default-blacklist` | bool | `true` | 使用默认黑名单规则 |
| `--blacklist-pattern` | stringSlice | `[]` | 自定义黑名单规则（可多次使用） |
| `--blacklist-file` | string | `""` | 黑名单规则文件路径 |

### 数据库选项

| 标志 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `--db` | bool | `false` | 启用数据库存储 |
| `--db-path` | string | `go-web-screenshot.db` | 数据库文件路径 |

SQLite 会保存标准化的 `schema_version`、`scheme`、`host`、`port`、`endpoint` 字段，并以 JSON 字段保存 TLS、Headers、Technologies、Network、Console、Cookies 等 Web 证据。

### 输出选项

| 标志 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `--write-jsonl` | bool | `false` | 写入 JSONL 格式结果 |
| `--jsonl-file` | string | `results.jsonl` | JSONL 结果文件路径 |
| `--write-csv` | bool | `false` | 写入 CSV 格式结果 |
| `--csv-file` | string | `results.csv` | CSV 结果文件路径 |
| `--write-stdout` | bool | `true` | 输出结果到控制台 |

---

## 子命令特定参数

### snir scan file

| 标志 | 短写 | 类型 | 默认值 | 说明 | 必填 |
|------|------|------|--------|------|------|
| `--file` | `-f` | string | `""` | URL 列表文件路径 | ✅ 是 |

```bash
# 必须指定 -f 参数
./snir scan file -f urls.txt

# 带并发和输出
./snir scan file -f urls.txt --threads 10 --write-jsonl
```

### snir scan single

| 位置参数 | 说明 | 必填 |
|---------|------|------|
| `url` | 要截图的 URL | ✅ 是 |

```bash
./snir scan single https://example.com
```

### snir scan cidr

| 位置参数 | 说明 | 必填 |
|---------|------|------|
| `cidr` | CIDR 网段（如 `192.168.1.0/24`） | ✅ 是 |

```bash
./snir scan cidr 192.168.1.0/24 --threads 20
```

---

## 实战组合示例

```bash
# 大规模网段扫描 + 高并发 + 代理轮换 + JSONL 输出
./snir scan cidr 10.0.0.0/16 \
  --threads 20 \
  --proxy-file proxies.txt \
  --proxy-strategy random \
  --timeout 60 \
  --write-jsonl --jsonl-file full-scan.jsonl \
  --save-html --save-headers

# 精确元素截图 + JavaScript 交互 + Cookie 注入
./snir scan dashboard.example.com \
  --cookie "session=abc123" \
  --js "document.querySelector('#login-form').submit();" \
  --delay 3 \
  --selector "#dashboard-panel" \
  --save-html

# 信息收集模式（不保存截图，只收集数据）
./snir scan file -f urls.txt \
  --skip-screenshot \
  --save-html --save-headers --save-cookies --save-console --save-network \
  --write-jsonl --jsonl-file intel.jsonl \
  --db --db-path intel.db

# 远程 Chrome + 全页截图 + 导出 Cookie
./snir scan example.com \
  --wss ws://chrome-server:9222/devtools/browser/xxx \
  --full-page \
  --cookie-file cookies.json --cookie-write-back
```
