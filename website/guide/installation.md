# 安装

<p align="center">📦 在各平台安装 snir。</p>

## 前置条件

- **OS**：Linux / macOS / FreeBSD / OpenBSD / NetBSD
- **Chrome/Chromium**：截图所需（或用远程 CDP，见 [远程 Chrome](../advanced/remote-chrome)）

## 选择安装方式

不同环境推荐的安装路径不同，按下图对号入座：

```mermaid
flowchart TD
    Start([安装 snir]) --> Q1{已克隆仓库?}
    Q1 -- 是 --> S1[脚本安装<br/>./scripts/install-snir.sh]
    Q1 -- 否 --> Q2{需要源码定制?}
    Q2 -- 是 --> S3[源码构建<br/>make build]
    Q2 -- 否 --> Q3{容器化部署?}
    Q3 -- 是 --> S4[Docker<br/>docker compose up -d]
    Q3 -- 否 --> S2[预编译二进制<br/>curl 下载 tar 包]
    S1 --> V[snir version]
    S2 --> V
    S3 --> V
    S4 --> V
    V --> Q4{本地有 Chrome?}
    Q4 -- 是 --> Done([✅ 就绪])
    Q4 -- 否 --> R[远程 CDP<br/>--wss ws://host:9222/...]
    R --> Done

    classDef pick fill:#e6f4ea,stroke:#3aa676,stroke-width:2px,color:#1a5d3a;
    class S1,S2,S3,S4 pick;
```

::: tip 💡 推荐路径
绝大多数用户走 **脚本安装 → 验证 → 就绪** 即可。源码构建与 Docker 适合二次开发与生产容器化场景。
:::

## 方式一：脚本安装（推荐）

仓库已克隆时，用自带脚本：

```bash
./scripts/install-snir.sh
snir version
```

`install-snir.sh` 是跨平台（Linux/macOS/BSD）安装助手，自动判断 OS/ARCH 下载对应二进制。

## 方式二：预编译二进制

```bash
LATEST=$(curl -s https://api.github.com/repos/cyberspacesec/snir-skills/releases/latest | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/')
OS=$(uname -s)
ARCH=$(uname -m | sed 's/aarch64/arm64/')
curl -L -o snir.tar.gz "https://github.com/cyberspacesec/snir-skills/releases/download/${LATEST}/snir-skills_${OS}_${ARCH}.tar.gz"
tar xzf snir.tar.gz snir
chmod +x snir
sudo mv snir /usr/local/bin/
snir version
```

## 方式三：源码构建

需 Go 1.23+：

```bash
git clone https://github.com/cyberspacesec/snir-skills.git
cd snir-skills
make build
./snir version
```

## 方式四：Docker

```bash
docker compose up -d   # 见 docker-compose.yml
# 或
docker build -t snir .
docker run --rm snir version
```

详见 [Docker 部署](../advanced/docker)。

## 安装 Chrome

**Linux (Debian/Ubuntu)**：

```bash
sudo apt install chromium-browser
# 或安装 Google Chrome
```

**macOS**：

```bash
brew install --cask chromium
```

::: info 没有 Chrome？
可连接**远程 CDP** 免本地浏览器：`snir scan example.com --wss ws://host:9222/...`，详见下方 [远程 Chrome 替代](#远程-chrome-替代)。
:::

## 验证

```bash
snir version
```

输出 Logo 与版本信息即成功。

::: details 预期输出
```text
   _____ _    _____
  / ___(_)  / ___/___  ___
 / /__  / / / /__/ __ \/ _/
 \___/_/_/  \___/_/ /_/\_/

snir vX.Y.Z
https://github.com/cyberspacesec/snir-skills
```
:::

## 平台注意

| 平台 | 注意 |
|------|------|
| 🐧 Linux | 主力平台，Chrome 通常用 chromium 包 |
| 🍎 macOS | 需安装 Chrome/Chromium 到 Applications |
| 🐡 BSD | 二进制可用；Chrome 需自行解决 |
| 🪟 Windows | 可从源码构建，需指定 `--chrome-path` |

## 远程 Chrome 替代

::: tip 无本地 Chrome 的最佳选择
连接远程 CDP，免本地浏览器，可被多 worker 复用：
:::

```bash
snir scan example.com --wss ws://host:9222/devtools/browser/<id>
```

见 [远程 Chrome](../advanced/remote-chrome)。

## 下一步

- [快速开始](./quick-start)
- [Docker 部署](../advanced/docker)
- [CI/CD 集成](../advanced/cicd)
