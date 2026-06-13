# 第一阶段：构建环境
FROM --platform=$BUILDPLATFORM golang:1.23-alpine3.21 AS build

# 设置工作目录
WORKDIR /app

# 安装构建依赖
RUN apk add --no-cache git make

# 设置环境变量
ENV GOTOOLCHAIN=auto

# 接收目标平台参数
ARG TARGETOS
ARG TARGETARCH

# 复制 Go 模块定义文件并下载依赖
COPY go.mod go.sum ./
RUN go mod download

# 复制所有源代码
COPY . .

# 执行交叉编译构建
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -ldflags "-s -w" -o snir .

# 第二阶段：运行环境
FROM alpine:3.21

# 安装运行时依赖
RUN apk add --no-cache ca-certificates chromium

# 设置工作目录
WORKDIR /app

# 创建配置和数据目录
RUN mkdir -p /app/data

# 从构建阶段复制编译好的二进制文件
COPY --from=build /app/snir /app/snir

# 复制必要的资源文件
COPY --from=build /app/webpage /app/webpage

# 设置环境变量
ENV PATH="/app:${PATH}"
ENV CHROME_BIN="/usr/bin/chromium-browser"
ENV CHROME_PATH="/usr/lib/chromium/"

# 暴露 API 端口
EXPOSE 8080

# 健康检查
HEALTHCHECK --interval=30s --timeout=5s --retries=3 \
    CMD wget -qO- http://localhost:8080/health || exit 1

# 设置容器启动命令
ENTRYPOINT ["/app/snir"]
CMD ["serve"]
