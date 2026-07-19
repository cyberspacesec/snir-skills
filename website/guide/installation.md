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

四条安装路径一览：

```mermaid
mindmap
  root((安装 snir))
    脚本
      ./scripts/install-snir.sh
      已克隆仓库
      最快推荐
    预编译二进制
      curl 下载 tar 包
      无需源码
      服务器直装
    源码构建
      make build
      二次开发定制
      需 Go 工具链
    Docker
      docker compose up -d
      生产容器化
      内置 Chromium
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

需 Go 1.26+（从 [go.dev/dl](https://go.dev/dl/) 下载安装）：

```bash
git clone https://github.com/cyberspacesec/snir-skills.git
cd snir-skills
make build                     # 构建 ./snir，注入版本/提交 ldflags
./snir version

# 可选：安装到 PATH
make install                   # go install 到 $GOPATH/bin

# 可选：交叉编译（纯 Go，构建机无需 Chrome）
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o snir-arm64 .
```

::: details make 目标说明
| 目标 | 作用 |
|------|------|
| `make build` | 构建本地平台二进制，注入 version/commit/date ldflags |
| `make install` | `go install` 到 `$GOPATH/bin` |
| `make test` | 运行全部测试 |
| `make release-test` | 本地预演 goreleaser 发布（`--snapshot --skip=publish`） |
:::

::: warning Go 版本
`go.mod` 声明 `go 1.26.0`，低于 1.26 的工具链会触发 toolchain 自动下载。建议直接安装 1.26.5+ 避免额外下载。
:::

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
