# Go Web Screenshot

一个用Go语言编写的网页截图工具，可以对网站进行截图并收集相关信息。

## 功能特点

- 对单个URL进行截图
- 批量处理URL列表文件
- 支持扫描CIDR网段
- 支持从Nmap和Nessus扫描结果导入目标
- 自定义截图分辨率和格式
- 可选择保存网页内容和响应头
- 支持多种输出格式（JSON、CSV、数据库）
- 内置Web服务器查看截图结果

## 安装

### 从源码安装

```bash
git clone https://github.com/cyberspacesec/snir-skills.git
cd go-web-screenshot
go build
```

### 使用Docker

#### 使用预构建镜像

```bash
docker pull cyberspacesec/go-web-screenshot
docker run -it --rm cyberspacesec/go-web-screenshot scan single https://example.com
```

#### 使用项目中的Dockerfile构建

```bash
# 构建镜像
docker build -t go-snir .

# 运行容器 - Web服务模式
docker run -p 8080:8080 -it --rm go-snir

# 运行容器 - 扫描单个URL
docker run -it --rm go-snir scan single -u https://example.com

# 运行容器 - 指定输出目录
docker run -v $(pwd)/data:/app/data -it --rm go-snir scan file -f /app/data/urls.txt
```

#### 使用Docker Compose

```bash
# 启动服务
docker-compose up -d

# 查看日志
docker-compose logs -f

# 停止服务
docker-compose down
```

## 使用方法

### 对单个URL截图

```bash
go-web-screenshot scan single https://example.com
```

### 从文件批量截图

```bash
go-web-screenshot scan file -f urls.txt
```

### 扫描CIDR网段

```bash
go-web-screenshot scan cidr -c 192.168.1.0/24 --port 80,443,8080
```

### 从Nmap XML文件导入

```bash
go-web-screenshot scan nmap -f scan.xml
```

### 启动Web服务器查看结果

```bash
go-web-screenshot report serve
```

## 详细使用示例

工具的选项很多，可能会让新用户感到困惑。我们提供了一系列常见使用场景的示例，您可以直接复制使用：

- [常见使用示例文档](docs/usage_examples.md) - 包含了基础扫描、批量扫描、结果输出和高级使用的各种示例

下面是最常用的几个简单例子：

```bash
# 扫描单个网站（最简单用法）
./snir scan example.com

# 批量扫描文件中的网站
./snir scan file -f urls.txt

# 对加载较慢的网站增加超时和延迟
./snir scan slow-website.com --timeout 60 --delay 3
```

## 配置选项

可以通过命令行参数自定义工具的行为：

- `--screenshot-path`: 截图保存路径
- `--resolution`: 截图分辨率，格式为"宽x高"
- `--timeout`: 页面加载超时时间
- `--user-agent`: 自定义User-Agent
- `--chrome-path`: 自定义Chrome路径
- `--delay`: 截图前等待时间
- `--save-html`: 保存网页HTML内容
- `--save-headers`: 保存HTTP响应头

## 许可证

本项目采用MIT许可证。详见[LICENSE](LICENSE)文件。