# go-snir 网页截图工具 — 能力文档

> 本文档描述 go-snir 的全部截图能力，以及如何通过 CLI、SDK、API、Provider 四种方式集成使用。

---

## 概述

go-snir 是一个基于 Chrome DevTools Protocol (CDP) 的网页截图与信息收集工具，也可以作为 Go 截图库集成到业务系统。核心能力包括：

- **网页截图**：全页/元素级截图，支持 PNG/JPEG、JPEG 质量控制、保存到磁盘或直接返回图片字节
- **信息收集**：HTML 源码、HTTP 头、Cookie、控制台日志、网络请求、TLS 信息、最终 URL/状态码
- **浏览器交互**：JavaScript 执行、表单填写、点击/滚动/输入操作
- **设备与指纹**：内置移动端/桌面设备预设，CDP 级 viewport/DPR/mobile/touch 模拟，自定义 User-Agent、WebGL、平台、语言等
- **CDP 复用**：连接池、单例池、远程连接、自动发现
- **代理轮换**：代理列表、代理文件热加载、代理 API，按代理隔离本地 Chrome 进程
- **图片分析**：感知哈希 pHash、技术栈识别，便于去重、聚类和资产画像
- **多种集成方式**：CLI、Go SDK、HTTP API、CDP Provider

如果用于网络空间测绘系统，当前库更适合作为 Web 资产采集、截图、指纹与证据采集子模块；完整底座能力边界见 [网络空间测绘底层库支撑性评估](cyberspace-mapping-assessment.md)。

---

## 一、CLI 使用

### 1.1 单 URL 截图

```bash
# 基本用法
./snir scan example.com

# 指定超时和延迟
./snir scan example.com --timeout 60 --delay 3

# 高分辨率
./snir scan example.com --resolution-x 1920 --resolution-y 1080

# 移动端设备预设（会模拟 viewport、DPR、mobile、touch 和 User-Agent）
./snir scan example.com --device iphone-15

# 查看可用设备预设
./snir scan --list-devices

# 使用代理
./snir scan example.com --proxy http://127.0.0.1:8080

# 忽略证书错误（HTTPS 站点）
./snir scan example.com --ignore-cert-errors

# 连接远程 Chrome（避免本地启动）
./snir scan example.com --wss ws://chrome-server:9222/devtools/browser/xxx

# 执行 JavaScript
./snir scan example.com --js "document.querySelectorAll('.popup').forEach(el => el.remove());"

# 页面加载前执行 JS
./snir scan example.com --js "window.__test = true" --run-js-before

# CSS 选择器截图（仅截取特定元素）
./snir scan example.com --selector "#main-content"

# XPath 截图
./snir scan example.com --xpath "//div[@class='chart']"

# 全页截图（包括滚动区域）
./snir scan example.com --full-page
```

### 1.2 批量扫描

```bash
# 从文件读取 URL 列表
./snir scan file -f urls.txt

# 从 host/IP 列表按协议和端口展开 URL
./snir scan file -f hosts.txt --ports 80,443,8080,8443

# 扫描网段
./snir scan cidr 192.168.1.0/24

# 扫描网段的常见 Web 端口
./snir scan cidr 192.168.1.0/24 --ports 80,443,8080,8443

# 调整并发数
./snir scan file -f urls.txt --threads 10

# 最大重试次数
./snir scan file -f urls.txt --max-retries 3
```

### 1.3 数据收集

```bash
# 保存 HTML 源码
./snir scan example.com --save-html

# 保存 HTTP 头
./snir scan example.com --save-headers

# 保存 Cookie
./snir scan example.com --save-cookies

# 保存控制台日志
./snir scan example.com --save-console

# 保存网络请求日志
./snir scan example.com --save-network

# 全部保存
./snir scan example.com --save-html --save-headers --save-cookies --save-console --save-network

# 输出到 JSONL/CSV
./snir scan file -f urls.txt --write-jsonl --jsonl-file results.jsonl
./snir scan file -f urls.txt --write-csv --csv-file results.csv

# 存入数据库
./snir scan file -f urls.txt --db --db-path screenshots.db
```

SQLite 会保存标准化的 `schema_version`、`scheme`、`host`、`port`、`endpoint` 字段，并以 JSON 字段保存 TLS、Headers、Technologies、Network、Console、Cookies 等 Web 证据。JSONL 仍会输出完整 `models.Result`。

### 1.4 API 服务

```bash
# 启动 API 服务
./snir api --port 8080

# 指定 API Key
./snir api --api-key my-secret-key

# 连接远程 Chrome
./snir api --wss ws://chrome-server:9222/devtools/browser/xxx

# 最大并发
./snir api --max-concurrent 20

# 忽略证书错误
./snir api --ignore-cert-errors
```

### 1.5 CDP Provider（共享 Chrome 给其他工具）

```bash
# 启动 Provider（默认端口 9223）
./snir provider

# 自定义配置
./snir provider --port 8090 --chrome-port 9222 --max-concurrent 20

# 设置空闲超时（5 分钟不用自动关闭浏览器）
./snir provider --idle-timeout 5m

# 使用代理
./snir provider --proxy http://127.0.0.1:8080

# 非无头模式（调试用）
./snir provider --headless=false
```

### 1.6 全部 CLI 标志

| 标志 | 默认值 | 说明 |
|------|--------|------|
| `--screenshot-path` | screenshots | 截图保存路径 |
| `--screenshot-format` | png | 截图格式 (png/jpeg) |
| `--screenshot-quality` | 90 | JPEG 截图质量 |
| `--skip-screenshot` | false | 跳过保存截图 |
| `--save-html` | false | 保存 HTML 源码 |
| `--save-headers` | false | 保存 HTTP 头 |
| `--save-console` | false | 保存控制台日志 |
| `--save-cookies` | false | 保存 Cookie |
| `--save-network` | false | 保存网络请求日志 |
| `--chrome-path` | "" | Chrome 可执行文件路径 |
| `--user-agent` | "" | 自定义 User-Agent |
| `--proxy` | "" | 代理服务器 |
| `--timeout` | 30 | 页面加载超时（秒） |
| `--delay` | 0 | 截图前等待（秒） |
| `--resolution-x` | 1280 | 窗口宽度 |
| `--resolution-y` | 800 | 窗口高度 |
| `--headless` | true | 无头模式 |
| `--ignore-cert-errors` | false | 忽略证书错误 |
| `--wss` | "" | 远程 Chrome WebSocket URL |
| `--threads` | 2 | 并发线程数 |
| `--max-retries` | 1 | 最大重试次数 |
| `--js` | "" | 页面上执行的 JavaScript |
| `--js-file` | "" | JavaScript 文件路径 |
| `--run-js-before` | false | 在页面加载前执行 JS |
| `--selector` | "" | CSS 选择器截图 |
| `--xpath` | "" | XPath 截图 |
| `--full-page` | false | 全页截图 |
| `--http` | true | 使用 HTTP 协议 |
| `--https` | true | 使用 HTTPS 协议 |
| `--ports` | [] | 对无协议 host/IP 目标展开端口列表，例如 `80,443,8080` |
| `--enable-blacklist` | true | 启用 URL 黑名单 |
| `--default-blacklist` | true | 使用默认黑名单 |
| `--blacklist-pattern` | [] | 自定义黑名单规则 |
| `--blacklist-file` | "" | 黑名单文件 |
| `--db` | false | 启用数据库存储 |
| `--db-path` | go-web-screenshot.db | 数据库路径 |
| `--write-jsonl` | false | 写入 JSONL |
| `--write-csv` | false | 写入 CSV |

---

## 二、Go SDK 集成

### 2.1 安装

```bash
go get github.com/cyberspacesec/snir-skills/pkg/sdk
```

### 2.2 基本使用

```go
package main

import (
    "fmt"
    "github.com/cyberspacesec/snir-skills/pkg/sdk"
)

func main() {
    // 创建客户端
    opts := sdk.DefaultClientOptions()
    opts.ScreenshotPath = "screenshots"
    opts.MaxConcurrent = 4
    
    client, err := sdk.NewClient(opts)
    if err != nil {
        panic(err)
    }
    defer client.Close()
    
    // 截图
    result, err := client.Screenshot("https://www.baidu.com", nil)
    if err != nil {
        panic(err)
    }
    fmt.Printf("标题: %s\n", result.Title)
}
```

### 2.3 截图字节数据（不写磁盘）

```go
imgBytes, result, err := client.ScreenshotBytes("https://example.com", nil)
if err != nil {
    panic(err)
}
// imgBytes 是 PNG/JPEG 的原始字节数据
fmt.Printf("图片大小: %d bytes\n", len(imgBytes))
```

### 2.4 带取消的截图

```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

result, err := client.ScreenshotWithContext(ctx, "https://example.com", nil)
```

### 2.5 设备预设截图

```go
opts := sdk.DefaultClientOptions()
opts.Device = "iphone-15" // 也支持 pixel-8-pro、ipad-pro-12、desktop-1080p 等

client, err := sdk.NewClient(opts)
if err != nil {
    panic(err)
}
defer client.Close()

result, err := client.Screenshot("https://example.com", &sdk.ScreenshotOptions{
    Device: "pixel-8-pro", // 单次截图可覆盖客户端默认设备
})
```

设备预设会在导航前通过 CDP 模拟 viewport、device scale factor、mobile/touch 能力，并设置匹配的 User-Agent。显式传入的 `UserAgent`、`Platform`、屏幕尺寸等指纹字段会覆盖设备预设。

### 2.6 批量截图

```go
urls := []string{"https://a.com", "https://b.com", "https://c.com"}
results := client.BatchScreenshot(urls, nil)

for _, r := range results {
    if r.Error != nil {
        fmt.Printf("❌ %s: %v\n", r.URL, r.Error)
    } else {
        fmt.Printf("✅ %s: %s\n", r.URL, r.Result.Title)
    }
}
```

### 2.7 流式批量截图（实时获取结果）

```go
// 每完成一个截图立即返回，不用等全部完成
ch := client.BatchScreenshotStreaming(ctx, urls, nil)
for result := range ch {
    if result.Error != nil {
        fmt.Printf("❌ %s: %v\n", result.URL, result.Error)
    } else {
        fmt.Printf("✅ %s: %s\n", result.URL, result.Result.Title)
    }
}
```

### 2.8 回调式批量截图

```go
// 每完成一个截图调用回调
client.BatchScreenshotCallback(ctx, urls, nil, func(r sdk.BatchResult) {
    fmt.Printf("完成: %s\n", r.URL)
})
```

### 2.9 连接远程 Chrome

```go
// 连接 Provider 或其他 Chrome 实例
wsURL := "ws://chrome-server:9222/devtools/browser/xxxx"
client, err := sdk.NewRemoteClient(wsURL, 4)
```

### 2.10 自动发现并连接

```go
// AutoConnect: 优先级：远程 > 发现本地 > 启动新实例
client, mode, err := sdk.AutoConnectClient(sdk.DefaultClientOptions())
fmt.Printf("连接模式: %s\n", mode) // "remote" | "discovered" | "local"
```

### 2.11 进程内共享池（多模块复用）

```go
// 任意包/模块调用，自动复用同一 Chrome 进程
result, _ := sdk.SharedScreenshot("https://example.com", nil)

// 查看统计
stats, _ := sdk.SharedStats()
fmt.Printf("总截图: %d, 失败: %d\n", stats.TotalScreenshots, stats.FailedScreenshots)

// 程序退出时关闭
defer sdk.CloseSharedPool()
```

### 2.12 事件监听

```go
client.OnEvent(func(event runner.PoolEvent) {
    switch event.Type {
    case runner.EventScreenshotComplete:
        fmt.Printf("✅ %s (%.2fs)\n", event.URL, event.Duration.Seconds())
    case runner.EventScreenshotFailed:
        fmt.Printf("❌ %s: %v\n", event.URL, event.Error)
    case runner.EventReconnect:
        fmt.Printf("🔄 浏览器重连 (第%d次)\n", event.ReconnectCount)
    case runner.EventIdleClose:
        fmt.Printf("💤 浏览器空闲关闭")
    }
})
```

### 2.13 空闲超时

```go
// 5分钟不用自动关闭浏览器，下次截图自动重启
client.SetIdleTimeout(5 * time.Minute)
```

### 2.14 统计信息

```go
stats := client.Stats()
fmt.Printf("活跃: %d, 总计: %d, 失败: %d, 重连: %d\n",
    stats.ActiveCount,
    stats.TotalScreenshots,
    stats.FailedScreenshots,
    stats.ReconnectCount,
)
```

### 2.15 便捷截图方法

#### 全页截图

```go
result, err := client.ScreenshotFullPage("https://example.com", nil)
```

#### 元素截图

```go
// CSS 选择器
result, err := client.ScreenshotElement("https://example.com", "#main-content", nil)
```

#### 执行 JavaScript 后截图

```go
result, err := client.ScreenshotWithJS("https://example.com",
    "window.scrollTo(0, document.body.scrollHeight)", nil)
```

#### 交互动作后截图

```go
actions := []runner.InteractionAction{
    {Type: "type", Selector: "#search", Value: "go-snir"},
    {Type: "click", Selector: "#search-btn"},
    {Type: "wait", WaitTime: 2},
}
result, err := client.ScreenshotWithActions("https://example.com", actions, nil)
```

#### 表单填写后截图

```go
form := runner.Form{
    Fields: []runner.FormField{
        {Selector: "#username", Value: "admin"},
        {Selector: "#password", Value: "pass123"},
    },
    SubmitSelector: "#login-btn",
    WaitAfterSubmit: 3,
}
result, err := client.ScreenshotWithForm("https://example.com/login", form, nil)
```

#### 注入 Cookie 后截图

```go
cookies := []runner.CustomCookie{
    {Name: "session", Value: "abc123", Domain: "example.com"},
}
result, err := client.ScreenshotWithCookies("https://example.com/dashboard", cookies, nil)
```

#### 截图并获取 HTML

```go
html, result, err := client.ScreenshotHTML("https://example.com", nil)
fmt.Printf("HTML 长度: %d\n", len(html))
```

#### 全证据采集

```go
result, err := client.ScreenshotEvidence("https://example.com", &sdk.ScreenshotOptions{
    CaptureFullPage: true,
})

imgBytes, result, err := client.ScreenshotEvidenceBytes("https://example.com", nil)
```

`ScreenshotEvidence` 会同时打开 `SaveHTML`、`SaveHeaders`、`SaveCookies`、`SaveConsole`、`SaveNetwork`，适合资产画像、网页证据归档、AI Agent 后续分析等场景。

#### XPath、设备和 viewport 截图

```go
result, err := client.ScreenshotXPath("https://example.com", "//main", nil)
imgBytes, result, err := client.ScreenshotElementBytes("https://example.com", "#chart", nil)

result, err = client.ScreenshotDevice("https://example.com", "iphone-15", nil)
result, err = client.ScreenshotViewport("https://example.com", 1440, 900, nil)
```

#### 页面加载前或 JS 文件注入

```go
result, err := client.ScreenshotWithJSBefore(
    "https://example.com",
    "window.__snir_probe = true",
    nil,
)

result, err = client.ScreenshotWithJSFile("https://example.com", "preload.js", true, nil)
```

### 2.16 结果便捷访问（ResultWrapper）

```go
w := sdk.WrapResult(result)

// 状态判断
w.IsSuccess()    // 截图是否成功
w.IsFailed()     // 截图是否失败
w.HasScreenshot() // 是否有截图文件
w.HasHTML()      // 是否包含 HTML

// 便捷取值
w.TitleOrDefault("无标题")     // 标题，空则返回默认值
w.ResponseCodeOrDefault(0)     // 状态码
w.HeaderValue("Content-Type")  // 获取指定 HTTP 头
w.CookieMap()                  // Cookie 的 map 形式
w.ConsoleErrors()              // 控制台错误日志
w.NetworkErrors()              // 失败的网络请求
w.TechnologyNames()            // 检测到的技术名称
w.TLSInfo()                    // TLS 信息
```

### 2.17 自定义截图选项

```go
opts := &sdk.ScreenshotOptions{
    Timeout:          60 * time.Second,              // 超时 60 秒
    Delay:            3 * time.Second,               // 延迟 3 秒
    WindowWidth:      1440,                          // 单次截图窗口宽度
    WindowHeight:     900,                           // 单次截图窗口高度
    UserAgent:        "Mozilla/5.0 Custom",          // 自定义 UA
    Proxy:            "http://127.0.0.1:8080",       // 代理
    ProxyList:        []string{"http://a:8080"},      // 代理轮换列表
    ProxyStrategy:    runner.ProxyRoundRobin,         // 轮换策略
    Device:           "iphone-15",                   // 设备预设
    IgnoreCertErrors: true,                           // 忽略证书错误
    Selector:         "#main-content",               // CSS 选择器
    CaptureFullPage:  true,                           // 全页截图
    ScreenshotFormat: "jpeg",                        // 截图格式
    ScreenshotQuality: 95,                            // JPEG 质量
    Ports:            []int{80, 443, 8080},           // host/IP 展开端口
    JavaScript:       "window.scrollTo(0, 500)",      // 执行 JS
    JavaScriptFile:   "inject.js",                    // JS 文件
    RunJSBefore:      true,                           // 页面加载前执行
    RunJSAfter:       false,                          // 页面加载后执行
    SaveHTML:         true,                           // 保存 HTML
    SaveHeaders:      true,                           // 保存 HTTP 头
    SaveConsole:      true,                           // 保存控制台
    SaveCookies:      true,                           // 保存 Cookie
    SaveNetwork:      true,                           // 保存网络请求
    SkipSave:         false,                          // 不跳过保存
    Cookies: []runner.CustomCookie{                   // 注入 Cookie
        {Name: "auth", Value: "xxx", Domain: ".example.com"},
    },
    CookieHeader:    "sid=abc; tenant=demo",          // Cookie Header 注入
    CookieImport:    "cookies.txt",                   // Netscape Cookie 导入
    CookieExport:    "out-cookies.txt",               // 结果 Cookie 导出
    CookieWriteBack: true,                            // 写回 SDK CookieJar
    Actions: []runner.InteractionAction{              // 交互动作
        {Type: "click", Selector: "#accept"},
        {Type: "wait", WaitTime: 2},
    },
    MaxRetries: 3,                                    // 重试次数
}

result, err := client.Screenshot("https://example.com", opts)
```

### 2.18 场景化 Capture API

`Capture` 和 `CaptureBytes` 支持函数式选项，更适合组合复杂截图场景：

```go
result, err := client.Capture(
    "https://example.com",
    sdk.WithFullPage(),
    sdk.WithEvidence(),
    sdk.WithDevice("iphone-15"),
    sdk.WithViewport(390, 844),
    sdk.WithDeviceEmulation(390, 844, 3, true, true),
    sdk.WithIgnoreCertErrors(),
    sdk.WithCustomHeaders(map[string]string{
        "X-Agent": "snir",
    }),
)
```

返回图片字节，不写磁盘：

```go
imgBytes, result, err := client.CaptureBytes(
    "https://example.com/dashboard",
    sdk.WithElement("#chart"),
    sdk.WithFormat("jpeg", 85),
    sdk.WithEvidence(),
)
```

常用组合：

```go
// 移动端全页证据采集
mobileEvidence := sdk.NewScreenshotOptions(
    sdk.WithDevice("pixel-8-pro"),
    sdk.WithFullPage(),
    sdk.WithEvidence(),
)
result, err := client.Screenshot("https://example.com", mobileEvidence)

// 登录态页面截图
result, err = client.Capture(
    "https://example.com/dashboard",
    sdk.WithInjectedCookies(runner.CustomCookie{
        Name: "session", Value: "abc", Domain: "example.com",
    }),
    sdk.WithActions(
        runner.InteractionAction{Type: "wait", WaitTime: 1000},
    ),
)

// 单次截图覆盖指纹
result, err = client.Capture(
    "https://example.com",
    sdk.WithUserAgent("Mozilla/5.0 Custom"),
    sdk.WithAcceptLanguage("zh-CN,zh;q=0.9"),
    sdk.WithFingerprint("Win32", "Google Inc.", "Intel Inc.", "Intel Iris"),
    sdk.WithMobileEmulation(3),
    sdk.WithTouchEmulation(true),
    sdk.WithDisableWebRTC(),
)

// 单次截图使用代理池和 Cookie Header
result, err = client.Capture(
    "https://example.com/dashboard",
    sdk.WithProxyList(runner.ProxyRoundRobin, "http://a:8080", "http://b:8080"),
    sdk.WithCookieHeader("session=abc; tenant=demo"),
    sdk.WithCookieWriteBack(),
    sdk.WithCookies(),
)

// Netscape Cookie 文件导入/导出
result, err = client.Capture(
    "https://example.com",
    sdk.WithCookieImport("cookies.txt"),
    sdk.WithCookieExport("out-cookies.txt"),
    sdk.WithCookies(),
)

// 单次截图覆盖 URL 黑名单
result, err = client.Capture(
    "https://example.com",
    sdk.WithBlacklist("*.internal.*", "metadata.google.internal"),
)
result, err = client.Capture(
    "https://example.com",
    sdk.WithNoBlacklist(),
)
```

### 2.19 浏览器指纹配置（反检测）

```go
opts := sdk.DefaultClientOptions()

// 基本指纹
opts.UserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) ..."
opts.AcceptLanguage = "zh-CN,zh;q=0.9,en;q=0.8"
opts.Platform = "Win32"
opts.Vendor = "Google Inc."

// WebGL 指纹
opts.WebGLVendor = "Intel Inc."
opts.WebGLRenderer = "Intel Iris OpenGL Engine"

// 插件列表
opts.Plugins = []string{
    "Chrome PDF Plugin",
    "Chrome PDF Viewer",
    "Native Client",
}

// 自定义 HTTP 头
opts.CustomHeaders = map[string]string{
    "X-Custom-Header": "value",
}

// 高级选项
opts.DisableWebRTC = true       // 禁用 WebRTC（防止泄漏真实 IP）
opts.SpoofScreenSize = true     // 伪造屏幕尺寸
opts.ScreenWidth = 1920
opts.ScreenHeight = 1080

client, _ := sdk.NewClient(opts)
```

同样的指纹能力也可以按单次截图覆盖：

```go
result, err := client.Capture(
    "https://example.com",
    sdk.WithAcceptLanguage("en-US,en;q=0.9"),
    sdk.WithCustomHeaders(map[string]string{"X-Trace": "asset-001"}),
    sdk.WithSpoofedScreen(1920, 1080),
)
```

## 三、HTTP API 集成

### 3.1 单张截图

```bash
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
    "javascript": "",
    "selector": "",
    "xpath": "",
    "capture_full_page": false,
    "https": true,
    "http": true
  }'
```

### 3.2 批量截图

```bash
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
```

### 3.3 查看统计

```bash
curl http://localhost:8080/stats
```

### 3.4 健康检查

```bash
curl http://localhost:8080/health
```

### 3.5 API 端点一览

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

---

## 四、CDP Provider（跨进程共享 Chrome）

### 4.1 启动 Provider

```bash
./snir provider --port 9223 --max-concurrent 20 --idle-timeout 5m
```

### 4.2 Provider HTTP API

| 端点 | 方法 | 说明 |
|------|------|------|
| `/` | GET | Provider 信息 |
| `/ws` | GET | 获取 WebSocket URL |
| `/health` | GET | 健康检查 |
| `/stats` | GET | 连接池统计 |
| `/screenshot?url=` | POST | 直接截图 |

### 4.3 其他工具连接 Provider

```bash
# 查询 WebSocket URL
curl http://provider-host:9223/ws

# 返回: {"ws_url": "ws://provider-host:9222/devtools/browser/xxx", ...}
```

```go
// Go SDK 连接
client, _ := sdk.NewRemoteClient("ws://provider-host:9222/devtools/browser/xxx", 4)
```

### 4.4 自动发现

```go
// 自动发现本地 Chrome 实例（扫描 9222/9223/9224）
client, mode, _ := sdk.AutoConnectClient(sdk.DefaultClientOptions())
// mode: "discovered" = 找到了 Provider 或其他 Chrome 实例
```

---

## 五、Chrome 复用架构

### 5.1 三层复用

```
层级 3: Provider 服务（跨进程/跨机器）
  多个独立进程 → 1个 Provider → 1个 Chrome 实例

层级 2: Singleton Pool（同进程内多模块）
  多个模块 → GetSharedPool() → 1个 Chrome 实例

层级 1: AutoConnect（自动发现）
  AutoConnectClient() → 优先连接已有 Chrome → 没有则启动新的
```

### 5.2 连接池特性

| 特性 | 说明 |
|------|------|
| 健康检查 | 每次截图前验证浏览器进程可用 |
| 自动恢复 | 浏览器崩溃时自动重启 |
| 优雅关闭 | 等待进行中截图完成 |
| 空闲超时 | 长时间不用自动关闭浏览器 |
| 智能重试 | 区分可重试/不可重试错误，指数退避 |
| 生命周期事件 | 6 种事件类型，异步回调 |
| 远程连接 | 通过 WebSocket URL 连接远程 Chrome |
| 代理隔离 | 本地 Chrome 下不同代理使用独立浏览器进程，避免进程级代理串用 |

远程 Chrome（`WSSURL`/`--wss`）的代理必须在远程浏览器启动时配置；CDP 连接后不能为单个请求动态切换进程级代理。

### 5.3 智能重试分类

| 不可重试（立即失败） | 可重试（指数退避重试） |
|---|---|
| `ERR_NAME_NOT_RESOLVED` | `ERR_CONNECTION_RESET` |
| `ERR_CONNECTION_REFUSED` | `ERR_TIMED_OUT` |
| `ERR_ADDRESS_UNREACHABLE` | `ERR_NETWORK_CHANGED` |
| `ERR_ACCESS_DENIED` | `Could not find node with given id` |
| | `context deadline exceeded` |
| | 浏览器进程不可用 |

### 5.4 生命周期事件

| 事件 | 触发时机 | 可用字段 |
|------|---------|---------|
| `screenshot_start` | 截图开始 | URL |
| `screenshot_complete` | 截图成功 | URL, Duration, Result |
| `screenshot_failed` | 截图失败 | URL, Duration, Error |
| `reconnect` | 浏览器重连 | ReconnectCount |
| `idle_close` | 空闲关闭浏览器 | — |
| `pool_closed` | 连接池关闭 | — |

---

## 六、SDK ClientOptions 完整参考

```go
type ClientOptions struct {
    // Chrome 浏览器配置
    ChromePath       string        // Chrome 可执行文件路径
    Headless         bool          // 无头模式（默认 true）
    WindowWidth      int           // 窗口宽度（默认 1280）
    WindowHeight     int           // 窗口高度（默认 800）
    UserAgent        string        // 自定义 User-Agent
    Proxy            string        // 代理服务器
    ProxyList        []string      // 代理轮换列表
    ProxyFile        string        // 代理文件
    ProxyURL         string        // 动态代理 API
    ProxyStrategy    runner.ProxyStrategy
    Device           string        // 设备预设名称
    DeviceScaleFactor float64      // 设备像素比
    IsMobile          bool         // 启用移动端仿真
    HasTouch          bool         // 启用触摸仿真
    WSSURL           string        // 远程 Chrome WebSocket URL
    IgnoreCertErrors bool          // 忽略证书错误

    // 浏览器指纹（反检测）
    AcceptLanguage   string            // Accept-Language 头
    Platform         string            // 平台标识
    Vendor           string            // 浏览器厂商
    Plugins          []string          // 插件列表
    WebGLVendor      string            // WebGL 厂商
    WebGLRenderer    string            // WebGL 渲染器
    CustomHeaders    map[string]string // 自定义 HTTP 头
    DisableWebRTC    bool              // 禁用 WebRTC
    SpoofScreenSize  bool              // 伪造屏幕尺寸
    ScreenWidth      int               // 伪造屏幕宽度
    ScreenHeight     int               // 伪造屏幕高度

    // 截图配置
    MaxConcurrent     int    // 最大并发截图数（默认 2）
    ScreenshotPath    string // 截图保存路径
    ScreenshotFormat  string // 截图格式 png/jpeg
    ScreenshotQuality int    // JPEG 质量（1-100，默认 90）
    SkipSave          bool   // 跳过保存到磁盘
    CaptureFullPage   bool   // 全页截图
    Selector          string // CSS 选择器截图
    XPath             string // XPath 截图
    Ports             []int  // 扫描端口列表

    // 超时配置
    Timeout time.Duration // 页面加载超时
    Delay   time.Duration // 截图前等待

    // JavaScript 执行
    JavaScript     string // 在页面上执行的 JavaScript
    JavaScriptFile string // JavaScript 文件路径
    RunJSBefore    bool   // 在页面加载前执行 JS
    RunJSAfter     bool   // 在页面加载后执行 JS

    // 数据收集
    SaveHTML    bool // 保存 HTML 源码
    SaveHeaders bool // 保存 HTTP 头
    SaveConsole bool // 保存控制台日志
    SaveCookies bool // 保存 Cookie
    SaveNetwork bool // 保存网络请求日志

    // 重试配置
    MaxRetries int // 最大重试次数（默认 1）

    // 自定义 Cookie
    Cookies         []runner.CustomCookie
    CookieHeader    string
    CookieStrings   []string
    CookieImport    string
    CookieExport    string
    CookieWriteBack bool

    // 浏览器交互
    Actions []runner.InteractionAction
    Form    runner.Form

    // 黑名单
    EnableBlacklist   bool     // 启用 URL 黑名单
    DefaultBlacklist  bool     // 使用默认黑名单
    BlacklistPatterns []string // 自定义黑名单规则
    BlacklistFile     string   // 黑名单文件
}

type ScreenshotOptions struct {
    // 超时覆盖
    Timeout time.Duration
    Delay   time.Duration

    // 浏览器覆盖
    WindowWidth      int
    WindowHeight     int
    UserAgent        string
    Proxy            string
    ProxyList        []string
    ProxyFile        string
    ProxyURL         string
    ProxyStrategy    runner.ProxyStrategy
    Device           string
    DeviceScaleFactor float64
    IsMobile          *bool
    HasTouch          *bool
    IgnoreCertErrors bool

    // 浏览器指纹覆盖
    AcceptLanguage  string
    Platform        string
    Vendor          string
    Plugins         []string
    WebGLVendor     string
    WebGLRenderer   string
    CustomHeaders   map[string]string
    DisableWebRTC   bool
    SpoofScreenSize bool
    ScreenWidth     int
    ScreenHeight    int

    // 截图覆盖
    Selector          string
    XPath             string
    CaptureFullPage   bool
    ScreenshotFormat  string
    ScreenshotQuality int
    Ports             []int

    // JavaScript
    JavaScript     string
    JavaScriptFile string
    RunJSBefore    bool
    RunJSAfter     bool

    // 数据收集覆盖
    SaveHTML    bool
    SaveHeaders bool
    SaveConsole bool
    SaveCookies bool
    SaveNetwork bool
    SkipSave    bool

    // 自定义 Cookie（注入）
    Cookies         []runner.CustomCookie
    CookieHeader    string
    CookieStrings   []string
    CookieImport    string
    CookieExport    string
    CookieWriteBack bool

    // 浏览器交互
    Actions []runner.InteractionAction
    Form    runner.Form

    // 黑名单覆盖
    EnableBlacklist   *bool
    DefaultBlacklist  *bool
    BlacklistPatterns []string
    BlacklistFile     string

    // 重试覆盖
    MaxRetries int
}
```

## 七、PoolStats 完整参考

```go
type PoolStats struct {
    ActiveCount      int       // 当前活跃截图数
    MaxConcurrent    int       // 最大并发数
    TotalScreenshots int64     // 总截图次数
    FailedScreenshots int64    // 失败截图次数
    ReconnectCount   int64     // 浏览器重连次数
    LastActive       time.Time // 最后活跃时间
    CreatedAt        time.Time // 池创建时间
    Closed           bool      // 是否已关闭
}
```

---

## 八、快速选择指南

| 场景 | 推荐方式 | 示例 |
|------|---------|------|
| 命令行一次性截图 | `snir scan` | `./snir scan example.com` |
| 批量扫描 | `snir scan file` | `./snir scan file -f urls.txt` |
| 网段扫描 | `snir scan cidr` | `./snir scan cidr 192.168.1.0/24` |
| 提供 HTTP API | `snir api` | `./snir api --port 8080` |
| 共享 Chrome 给其他工具 | `snir provider` | `./snir provider` |
| Go 程序内截图 | SDK `NewClient` | `sdk.NewClient(opts)` |
| Go 程序连远程 Chrome | SDK `NewRemoteClient` | `sdk.NewRemoteClient(wsURL, 4)` |
| Go 程序零配置 | SDK `AutoConnectClient` | `sdk.AutoConnectClient(opts)` |
| 同进程多模块共享 | SDK `SharedScreenshot` | `sdk.SharedScreenshot(url, nil)` |
| 监控截图状态 | SDK `OnEvent` | `client.OnEvent(handler)` |
