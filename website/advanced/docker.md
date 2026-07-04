# Docker 部署

<p align="center">🐳 用 Docker 运行 snir，内置 Chromium。</p>

snir 提供多阶段 Dockerfile，运行时镜像基于 alpine + chromium。

构建与运行阶段的关系：

```mermaid
flowchart LR
    subgraph Build [构建阶段]
        B1[golang:1.23-alpine] --> B2[go build 交叉编译] --> B3[/app/snir 二进制]
    end
    subgraph Run [运行阶段]
        R1[alpine:3.21] --> R2[安装 chromium] --> R3[复制 snir 二进制] --> R4[ENTRYPOINT snir CMD serve]
    end
    B3 --> R3
    R4 --> P[EXPOSE 8080<br/>HEALTHCHECK /health]
    P --> Vol[/app/data 挂载卷<br/>截图/SQLite/JSONL]

    style B2 fill:#e6f4ea,stroke:#3aa676
    style R4 fill:#3aa676,stroke:#2a7a56,color:#fff
    style Vol fill:#e6f4ea,stroke:#3aa676
```

## Dockerfile 概览

```dockerfile
# 构建阶段：golang:1.23-alpine → 交叉编译 snir
# 运行阶段：alpine:3.21 + chromium
```

关键点：

- 🏗️ 多阶段构建，最终镜像小
- 🌐 内置 `chromium`，无需另装 Chrome
- 🌍 `ENV CHROME_BIN=/usr/bin/chromium-browser`、`CHROME_PATH=/usr/lib/chromium/`
- 🚪 `EXPOSE 8080`（API）
- ❤️ `HEALTHCHECK` 调 `/health`
- 🚀 `ENTRYPOINT ["/app/snir"]`，`CMD ["serve"]`

## docker-compose

```yaml
services:
  snir:
    build: .
    container_name: go-snir
    ports:
      - "8080:8080"
    volumes:
      - ./data:/app/data
    restart: unless-stopped
    command: ["serve"]
```

## 使用

```bash
# 构建并启动
docker compose up -d

# 或手动
docker build -t snir .
docker run --rm -p 8080:8080 -v $(pwd)/data:/app/data snir serve

# 跑截图
docker run --rm -v $(pwd)/data:/app/data snir scan example.com

# 启动 API
docker run --rm -p 8080:8080 -v $(pwd)/data:/app/data snir api --api-key secret
```

## 数据卷

挂载 `./data:/app/data` 持久化截图、SQLite、JSONL。

## 注意

::: warning 容器内 Chrome 必须关沙箱
alpine 容器内默认以 root 运行，Chromium 的 sandbox 在此环境下会报 `No usable sandbox!` 而崩溃。snir 在容器内已自动追加 `--no-sandbox`。

- ✅ 容器内可放心使用 `--no-sandbox`（容器本身就是隔离边界）
- ❌ **宿主机直跑切勿 `--no-sandbox`**，会降低安全性，请改用专用用户 + sandbox
:::

::: info 容器网络与资源
- 🌐 访问内网目标时，容器默认 bridge 网络可能无法到达宿主机内网——用 `--network host` 或自定义网络
- 🧠 大批量采集（数百/上千 URL）调大 `mem_limit` 与 `--threads`，否则容器 OOM 被杀
- 💾 数据卷 `./data:/app/data` 必须挂载，否则容器重建后截图/SQLite 全丢
:::

## 下一步

- [CI/CD 集成](./cicd)
- [安装](../guide/installation)
- [HTTP API](../api/overview)
