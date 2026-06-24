# snir — 网页截图与信息收集工具

<p align="center">
  <strong>基于 Chrome DevTools Protocol 的网页截图工具，支持多种接入方式</strong>
</p>

<p align="center">
  <a href="https://github.com/cyberspacesec/snir-skills/releases/latest"><img src="https://img.shields.io/github/v/release/cyberspacesec/snir-skills?style=flat-square" alt="Release"></a>
  <img src="https://img.shields.io/github/go-mod/go-version/cyberspacesec/snir-skills?style=flat-square" alt="Go Version">
  <img src="https://img.shields.io/github/license/cyberspacesec/snir-skills?style=flat-square" alt="License">
  <img src="https://img.shields.io/github/actions/workflow/status/cyberspacesec/snir-skills/ci.yml?branch=main&style=flat-square" alt="CI">
</p>

[English](README.md) | 简体中文

---

## 接入方式

### 1. 🤖 SKILLS（AI Agent 集成）— 推荐方式

SKILLS 提供渐进式披露文档，AI Agent 可自主安装和操作 snir，无需预装 Go SDK。

**AI Agent 一键安装：**

```bash
# 自动检测平台并安装最新版本
LATEST=$(curl -s https://api.github.com/repos/cyberspacesec/snir-skills/releases/latest | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/')
OS=$(uname -s | sed 's/Linux/Linux/;s/Darwin/Darwin/;s/FreeBSD/Freebsd/;s/OpenBSD/Openbsd/;s/NetBSD/Netbsd/')
ARCH=$(uname -m | sed 's/x86_64/x86_64/;s/aarch64/arm64/;s/arm64/arm64/')
curl -L -o snir.tar.gz "https://github.com/cyberspacesec/snir-skills/releases/download/${LATEST}/snir-skills_${OS}_${ARCH}.tar.gz"
tar xzf snir.tar.gz snir && chmod +x snir && sudo mv snir /usr/local/bin/
snir version
```

📖 **Anthropic Skills 入口：** [`SKILL.md`](SKILL.md)

标准 skill bundle 资源：

| 资源 | 用途 |
|------|------|
| [`references/`](references/) | Agent 按需加载的任务参考 |
| [`scripts/install-snir.sh`](scripts/install-snir.sh) | 可复用的跨平台 release 安装脚本 |
| [`evals/evals.json`](evals/evals.json) | Skill 评测提示词和预期行为 |

📖 **完整 SKILLS 文档：** [`docs/superpowers/SKILLS.md`](docs/superpowers/SKILLS.md)

每个命令的渐进式文档（快速上手 → 常用选项 → 高级选项 → 完整参数参考）：

| 命令 | 文档 |
|------|------|
| `scan` | [`scan.md`](docs/superpowers/scan.md) |
| `api` | [`api.md`](docs/superpowers/api.md) |
| `provider` | [`provider.md`](docs/superpowers/provider.md) |
| `report` | [`report.md`](docs/superpowers/report.md) |
| `webserve` | [`webserve.md`](docs/superpowers/webserve.md) |
| `version` | [`version.md`](docs/superpowers/version.md) |

### 2. 🖥️ CLI

```bash
# 单 URL 截图
snir scan example.com

# 从文件批量截图
snir scan file -f urls.txt

# 对裸 host/IP 展开常见 Web 端口
snir scan file -f hosts.txt --ports 80,443,8080,8443

# 网段扫描
snir scan cidr 192.168.1.0/24

# 全页截图 + 数据收集
snir scan example.com --full-page --save-html --save-headers --save-cookies
```

### 3. 📦 Go SDK

```go
import "github.com/cyberspacesec/snir-skills/pkg/sdk"

client, _ := sdk.NewClient(sdk.DefaultClientOptions())
defer client.Close()

result, _ := client.Screenshot("https://example.com", nil)
fmt.Println(result.Title)
```

📖 [Go SDK 文档](docs/skills.md#二go-sdk-集成)

### 4. 🌐 HTTP API

```bash
# 启动 API 服务
snir api --port 8080 --api-key secret

# 通过 API 截图
curl -X POST http://localhost:8080/screenshot \
  -H "X-API-Key: secret" \
  -H "Content-Type: application/json" \
  -d '{"url": "https://example.com"}'
```

📖 [HTTP API 文档](docs/superpowers/api.md)

### 5. 🔌 CDP Provider（跨进程共享 Chrome）

```bash
snir provider --port 9223 --idle-timeout 5m
# 其他工具通过 --wss ws://host:9222/devtools/browser/xxx 连接
```

📖 [Provider 文档](docs/superpowers/provider.md)

---

## 功能特点

- **截图** — 全页/元素级（CSS 选择器 / XPath），支持 PNG/JPEG、质量控制、落盘或内存字节返回
- **信息收集** — HTML 源码、HTTP 头、Cookie、控制台日志、网络请求、TLS/最终 URL/状态码元数据
- **浏览器交互** — JavaScript 执行、表单填写、点击/滚动/输入操作
- **设备与指纹** — CDP 设备预设、移动端/touch/DPR 模拟、自定义 User-Agent、WebGL、平台、语言、WebRTC 禁用
- **Chrome 复用** — 连接池、单例池、远程连接、自动发现
- **代理轮换** — 代理列表、代理文件（热加载）、代理 API、轮换策略、本地 Chrome 按代理隔离进程
- **Cookie 管理** — 持久化 JSON Cookie Jar、Netscape 格式导入/导出、内联 Cookie
- **库能力** — Go SDK、HTTP API、pHash、技术栈识别、流式/回调式批量截图
- **输出** — JSONL、CSV、SQLite 数据库、控制台
- **跨平台** — 43 个平台组合（Linux/Windows/macOS/FreeBSD/OpenBSD/NetBSD × amd64/arm64/386/arm/mips/ppc64le/riscv64/s390x）

用于网络空间测绘系统时，snir 更适合作为 Web 资产采集、截图、指纹与页面证据子模块。完整能力边界和缺口见 [网络空间测绘底层库支撑性评估](docs/cyberspace-mapping-assessment.md)。

---

## 安装

### 预编译二进制（无需 Go）

从 [GitHub Releases](https://github.com/cyberspacesec/snir-skills/releases/latest) 下载：

| 平台 | 命令 |
|------|------|
| **Linux x86_64** | `curl -L https://github.com/cyberspacesec/snir-skills/releases/latest/download/snir-skills_Linux_x86_64.tar.gz \| tar xz snir` |
| **macOS arm64** | `curl -L https://github.com/cyberspacesec/snir-skills/releases/latest/download/snir-skills_Darwin_arm64.tar.gz \| tar xz snir` |
| **Windows x86_64** | 从 [Releases](https://github.com/cyberspacesec/snir-skills/releases/latest) 下载 `snir-skills_Windows_x86_64.zip` |

### Linux 包管理器（deb/rpm/archlinux）

每个 [Release](https://github.com/cyberspacesec/snir-skills/releases/latest) 中提供：

```bash
sudo dpkg -i snir_*.deb        # Debian/Ubuntu
sudo rpm -i snir-*.rpm         # RHEL/Fedora
sudo pacman -U snir-*.pkg.tar.zst  # Arch Linux
```

### Docker

```bash
docker pull ghcr.io/cyberspacesec/snir:latest
docker run --rm ghcr.io/cyberspacesec/snir:latest scan example.com
```

### 从源码编译（需要 Go 1.23+）

```bash
git clone https://github.com/cyberspacesec/snir-skills.git
cd snir-skills && make build
```

### 前置依赖

截图功能需要 Chrome/Chromium。也可以通过 `--wss` 连接远程 Chrome 实例。

```bash
sudo apt install chromium-browser   # Debian/Ubuntu
brew install --cask google-chrome   # macOS
```

---

## 常用示例

```bash
# 单 URL 截图
snir scan example.com

# 指定超时和代理
snir scan example.com --timeout 60 --proxy http://127.0.0.1:8080

# 全页截图 + 收集所有信息
snir scan example.com --full-page --save-html --save-headers --save-cookies --save-network

# 元素截图（CSS 选择器）
snir scan example.com --selector "#dashboard-panel"

# 截图前执行 JavaScript
snir scan example.com --js "document.querySelectorAll('.popup').forEach(el => el.remove());"

# 批量扫描 + 代理轮换
snir scan file -f urls.txt --threads 10 --proxy-file proxies.txt --proxy-strategy random

# 输出为 JSONL + 数据库
snir scan file -f urls.txt --write-jsonl --db --db-path results.db
```

---

## 文档

| 文档 | 说明 |
|------|------|
| [SKILLS 索引](docs/superpowers/SKILLS.md) | AI Agent 集成 — 安装、命令、全部 70 个 CLI 标志 |
| [完整能力文档](docs/skills.md) | CLI + Go SDK + HTTP API + Provider 完整参考 |
| [测绘底层库评估](docs/cyberspace-mapping-assessment.md) | 作为网络空间测绘系统底层库时的支撑边界、缺口和补齐优先级 |
| [快速示例](docs/quick_examples.md) | 常见场景的复制粘贴示例 |
| [使用示例](docs/usage_examples.md) | 带解释的详细示例 |

---

## 许可证

[MIT](LICENSE)
