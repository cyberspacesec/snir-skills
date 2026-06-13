# snir CLI SKILLS 文档

> **渐进式披露结构**：从快速上手 → 常用场景 → 高级选项 → 完整参数参考，逐层深入。

---

## 工具概述

`snir` 是一个基于 Chrome DevTools Protocol (CDP) 的网页截图与信息收集工具，支持四种集成方式：

| 方式 | 适用场景 | 文档链接 |
|------|---------|---------|
| **CLI** | 命令行一次性/批量截图 | ← 你在这里 |
| **Go SDK** | Go 程序内调用 | [skills.md](../skills.md) |
| **HTTP API** | 其他语言/微服务调用 | [skills.md](../skills.md#三http-api-集成) |
| **CDP Provider** | 跨进程共享 Chrome | [provider.md](provider.md) |

---

## 安装

### 方式一：下载预编译二进制（推荐，无需 Go SDK）

从 [GitHub Releases](https://github.com/cyberspacesec/snir-skills/releases) 下载对应平台的可执行文件。

#### AI Agent 一键安装脚本（推荐）

自动检测操作系统和 CPU 架构，下载最新版本并安装：

```bash
# 自动检测平台并安装 — 适用于 Linux/macOS/FreeBSD/OpenBSD/NetBSD
LATEST=$(curl -s https://api.github.com/repos/cyberspacesec/snir-skills/releases/latest | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/')
OS=$(uname -s)
ARCH=$(uname -m)
case "$OS" in
  Linux)   OS_NAME="Linux" ;;
  Darwin)  OS_NAME="Darwin" ;;
  FreeBSD) OS_NAME="FreeBSD" ;;
  OpenBSD) OS_NAME="OpenBSD" ;;
  NetBSD)  OS_NAME="NetBSD" ;;
  *)       echo "Unsupported OS: $OS"; exit 1 ;;
esac
case "$ARCH" in
  x86_64)         ARCH_NAME="x86_64" ;;
  aarch64|arm64)  ARCH_NAME="arm64" ;;
  i386|i686)      ARCH_NAME="i386" ;;
  armv5*)         ARCH_NAME="armv5" ;;
  armv6*)         ARCH_NAME="armv6" ;;
  armv7*)         ARCH_NAME="armv7" ;;
  mips)           ARCH_NAME="mips" ;;
  mipsel)         ARCH_NAME="mipsle" ;;
  mips64)         ARCH_NAME="mips64" ;;
  mips64el)       ARCH_NAME="mips64le" ;;
  ppc64le)        ARCH_NAME="ppc64le" ;;
  riscv64)        ARCH_NAME="riscv64" ;;
  s390x)          ARCH_NAME="s390x" ;;
  *)              echo "Unsupported arch: $ARCH"; exit 1 ;;
esac
curl -L -o snir.tar.gz "https://github.com/cyberspacesec/snir-skills/releases/download/${LATEST}/snir-skills_${OS_NAME}_${ARCH_NAME}.tar.gz"
tar xzf snir.tar.gz snir
chmod +x snir
sudo mv snir /usr/local/bin/
snir version
```

#### Windows 安装脚本

```powershell
# PowerShell — Windows amd64
$LATEST = (Invoke-RestMethod https://api.github.com/repos/cyberspacesec/snir-skills/releases/latest).tag_name
$arch = if ([Environment]::Is64BitOperatingSystem) { "x86_64" } else { "i386" }
$url = "https://github.com/cyberspacesec/snir-skills/releases/download/$LATEST/snir-skills_Windows_$arch.zip"
Invoke-WebRequest -Uri $url -OutFile snir.zip
Expand-Archive snir.zip -DestinationPath .
# 将 snir.exe 移动到 PATH 中的目录
```

#### 手动下载

| 操作系统 | 架构 | 下载链接 |
|---------|------|---------|
| **Linux** | x86_64 | `snir-skills_Linux_x86_64.tar.gz` |
| **Linux** | arm64 | `snir-skills_Linux_arm64.tar.gz` |
| **Linux** | i386 | `snir-skills_Linux_i386.tar.gz` |
| **Linux** | armv7 | `snir-skills_Linux_armv7.tar.gz` |
| **Linux** | armv6 | `snir-skills_Linux_armv6.tar.gz` |
| **Linux** | armv5 | `snir-skills_Linux_armv5.tar.gz` |
| **Linux** | mips | `snir-skills_Linux_mips.tar.gz` |
| **Linux** | mipsle | `snir-skills_Linux_mipsle.tar.gz` |
| **Linux** | mips64 | `snir-skills_Linux_mips64.tar.gz` |
| **Linux** | mips64le | `snir-skills_Linux_mips64le.tar.gz` |
| **Linux** | ppc64le | `snir-skills_Linux_ppc64le.tar.gz` |
| **Linux** | riscv64 | `snir-skills_Linux_riscv64.tar.gz` |
| **Linux** | s390x | `snir-skills_Linux_s390x.tar.gz` |
| **macOS** | x86_64 (Intel) | `snir-skills_Darwin_x86_64.tar.gz` |
| **macOS** | arm64 (Apple Silicon) | `snir-skills_Darwin_arm64.tar.gz` |
| **Windows** | x86_64 | `snir-skills_Windows_x86_64.zip` |
| **Windows** | arm64 | `snir-skills_Windows_arm64.zip` |
| **Windows** | i386 | `snir-skills_Windows_i386.zip` |
| **FreeBSD** | x86_64 | `snir-skills_FreeBSD_x86_64.tar.gz` |
| **FreeBSD** | arm64 | `snir-skills_FreeBSD_arm64.tar.gz` |
| **FreeBSD** | i386 | `snir-skills_FreeBSD_i386.tar.gz` |
| **FreeBSD** | armv6/v7 | `snir-skills_FreeBSD_armvX.tar.gz` |
| **FreeBSD** | mips | `snir-skills_FreeBSD_mips.tar.gz` |
| **FreeBSD** | mipsle | `snir-skills_FreeBSD_mipsle.tar.gz` |
| **FreeBSD** | mips64 | `snir-skills_FreeBSD_mips64.tar.gz` |
| **FreeBSD** | mips64le | `snir-skills_FreeBSD_mips64le.tar.gz` |
| **FreeBSD** | ppc64le | `snir-skills_FreeBSD_ppc64le.tar.gz` |
| **FreeBSD** | riscv64 | `snir-skills_FreeBSD_riscv64.tar.gz` |
| **OpenBSD** | x86_64 | `snir-skills_OpenBSD_x86_64.tar.gz` |
| **OpenBSD** | arm64 | `snir-skills_OpenBSD_arm64.tar.gz` |
| **OpenBSD** | i386 | `snir-skills_OpenBSD_i386.tar.gz` |
| **OpenBSD** | armv6/v7 | `snir-skills_OpenBSD_armvX.tar.gz` |
| **OpenBSD** | riscv64 | `snir-skills_OpenBSD_riscv64.tar.gz` |
| **NetBSD** | x86_64 | `snir-skills_NetBSD_x86_64.tar.gz` |
| **NetBSD** | arm64 | `snir-skills_NetBSD_arm64.tar.gz` |
| **NetBSD** | i386 | `snir-skills_NetBSD_i386.tar.gz` |
| **NetBSD** | armv6/v7 | `snir-skills_NetBSD_armvX.tar.gz` |
| **NetBSD** | riscv64 | `snir-skills_NetBSD_riscv64.tar.gz` |

> 下载链接格式：`https://github.com/cyberspacesec/snir-skills/releases/download/{版本号}/{文件名}`

### 方式二：Linux 包管理器

GoReleaser 自动构建 deb/rpm/archlinux 包，从 [Releases](https://github.com/cyberspacesec/snir-skills/releases) 下载：

```bash
# Debian/Ubuntu
sudo dpkg -i snir_1.0.0_linux_amd64.deb

# RHEL/CentOS/Fedora
sudo rpm -i snir-1.0.0_linux_amd64.rpm

# Arch Linux
sudo pacman -U snir-1.0.0_linux_amd64.pkg.tar.zst
```

包管理器安装后：
- 二进制文件：`/usr/bin/snir`
- SKILLS 文档：`/usr/share/doc/snir/superpowers/`

### 方式三：Homebrew（macOS/Linux）

```bash
brew tap cyberspacesec/homebrew-tap
brew install snir
```

### 方式四：Docker

```bash
# 拉取镜像
docker pull ghcr.io/cyberspacesec/snir:latest

# 运行截图
docker run --rm ghcr.io/cyberspacesec/snir:latest scan example.com

# 启动 API 服务
docker run -p 8080:8080 ghcr.io/cyberspacesec/snir:latest api --port 8080

# 使用 docker-compose
docker compose up -d
```

### 方式五：从源码编译（需要 Go 1.23+）

```bash
# 克隆仓库
git clone https://github.com/cyberspacesec/snir-skills.git
cd snir-skills

# 编译安装
make build
# 或直接安装到 GOPATH/bin
make install

# 验证
./snir version
```

### 前置依赖

截图功能需要 Chrome/Chromium 浏览器：

```bash
# Debian/Ubuntu
sudo apt install chromium-browser

# Fedora/RHEL
sudo dnf install chromium

# macOS
brew install --cask google-chrome

# FreeBSD
sudo pkg install chromium

# 也可以通过 --chrome-path 指定非标准路径
./snir scan example.com --chrome-path /opt/google/chrome/chrome

# 或通过 --wss 连接远程 Chrome（无需本机安装浏览器）
./snir scan example.com --wss ws://chrome-server:9222/devtools/browser/xxx
```

> **无 Chrome 的平台**：arm/mips/ppc64le/riscv64/s390x 等平台可能无法安装 Chrome，
> 此时可以通过 `--wss` 参数连接到另一台有 Chrome 的机器上运行的 Provider。
> 参见 [provider.md](provider.md)。

---

## CLI 命令树

```
snir
├── scan [url]              ← 直接传 URL 即可截图
│   ├── single [url]        ← 单 URL 截图
│   ├── cidr [cidr]         ← 网段批量截图
│   └── file -f <path>      ← 文件批量截图
├── api                     ← 启动 HTTP API 服务
├── provider                ← 启动 CDP Provider（共享 Chrome）
├── report                  ← 报告操作
│   ├── convert             ← 格式转换
│   ├── merge               ← 合并报告
│   └── html                ← 生成 HTML 报告
├── webserve / serve        ← Web 查看服务器
└── version                 ← 显示版本信息
```

---

## 快速上手（30 秒入门）

```bash
# 单 URL 截图 — 最简单的用法
./snir scan example.com

# 批量截图 — 从文件读取 URL
./snir scan file -f urls.txt

# 网段扫描 — 自动展开 CIDR
./snir scan cidr 192.168.1.0/24
```

---

## 命令详细文档

每个命令都有独立的渐进式文档，按 **快速上手 → 常用选项 → 高级选项 → 完整参数** 四层组织：

| 命令 | 文档 | 核心用途 |
|------|------|---------|
| `scan` | [scan.md](scan.md) | 网页截图与信息收集（含 single/cidr/file 子命令） |
| `api` | [api.md](api.md) | 启动 HTTP API 截图服务 |
| `provider` | [provider.md](provider.md) | 启动 CDP Provider 共享 Chrome 进程 |
| `report` | [report.md](report.md) | 报告转换、合并、生成 HTML（含 convert/merge/html 子命令） |
| `webserve` | [webserve.md](webserve.md) | Web 服务器查看截图结果 |
| `version` | [version.md](version.md) | 版本信息与全局调试标志 |

---

## 全局标志（所有命令通用）

以下标志继承到所有子命令：

| 标志 | 短写 | 默认值 | 说明 |
|------|------|--------|------|
| `--debug-log` | `-D` | `false` | 启用调试日志，输出 ChromeDP 内部通信细节 |
| `--quiet` | `-q` | `false` | 静默模式，几乎不输出任何日志 |

```bash
# 调试模式 — 排查截图失败原因
./snir scan example.com -D

# 静默模式 — 仅输出最终结果
./snir scan file -f urls.txt -q
```

---

## 常见场景速查

| 场景 | 命令 |
|------|------|
| 单 URL 截图 | `./snir scan example.com` |
| 批量文件扫描 | `./snir scan file -f urls.txt` |
| 网段扫描 | `./snir scan cidr 192.168.1.0/24` |
| 超时/慢加载 | `./snir scan example.com --timeout 60 --delay 3` |
| 代理访问 | `./snir scan example.com --proxy http://127.0.0.1:8080` |
| 全页截图 | `./snir scan example.com --full-page` |
| 元素截图 | `./snir scan example.com --selector "#chart"` |
| 保存 HTML/头/Cookie | `./snir scan example.com --save-html --save-headers --save-cookies` |
| 输出 JSONL/CSV | `./snir scan file -f urls.txt --write-jsonl --write-csv` |
| 存数据库 | `./snir scan file -f urls.txt --db --db-path screenshots.db` |
| 启动 API 服务 | `./snir api --port 8080 --api-key secret` |
| 共享 Chrome | `./snir provider --port 9223` |
| 查看结果 | `./snir serve --port 8080` |