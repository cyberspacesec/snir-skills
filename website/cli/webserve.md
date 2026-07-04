# webserve 命令

<p align="center">🌐 `snir webserve` — 本地 Web 服务器查看结果。</p>

启动一个本地静态 Web 服务器，托管截图、报告等生成产物，便于浏览器查看。

## 用法

```bash
snir webserve [flags]
```

## 标志

| 标志 | 默认 | 说明 |
|------|------|------|
| `--host` | `0.0.0.0` | Web 服务器监听地址 |
| `--port` | `8080` | Web 服务器监听端口 |

（目录参数从 `Options.Report` 或位置参数指定）

## 示例

```bash
# 生成报告并查看
snir scan file -f urls.txt --write-jsonl
snir report html -i results.jsonl -o report.html
snir webserve --dir .

# 浏览器访问 http://localhost:8080
```

## 适用场景

::: tip 三步看报告，无需部署
```bash
snir scan file -f urls.txt --write-jsonl      # 1. 扫
snir report html -i results.jsonl -o report.html  # 2. 生成报告
snir webserve --dir .                          # 3. 浏览器打开 localhost:8080
```
适合本地浏览 HTML 报告、查看截图目录、临时内网分享——无需 Nginx，一条命令起服务。
:::

## 与 provider 的区别

- `webserve`：托管**静态文件**（报告/截图）
- `provider`：提供 **CDP 浏览器连接**

两者职责对比：

```mermaid
flowchart LR
    subgraph webserve [webserve :8080]
        S1[扫描产物目录] --> Srv[静态文件服务]
        Srv --> B1[浏览器查看报告/截图]
    end
    subgraph provider [provider :9223]
        Chr[(常驻 Chrome)] --> Cdp[CDP 端点]
        Cdp --> W1[worker --wss 接入扫描]
    end

    style Srv fill:#e6f4ea,stroke:#3aa676
    style Cdp fill:#e6f4ea,stroke:#3aa676
```

`webserve` 从启动到浏览器访问的时序：

```mermaid
sequenceDiagram
  participant U as 用户
  participant WS as snir webserve
  participant FS as 扫描产物目录
  participant HTTP as 静态服务器
  participant BR as 浏览器
  U->>WS: webserve --dir . --host --port
  WS->>FS: 扫描产物目录
  WS->>HTTP: 绑定 :8080 监听
  HTTP-->>WS: 就绪
  WS-->>U: 打印访问地址
  BR->>HTTP: GET http://localhost:8080/
  HTTP->>FS: 读取 report.html / 截图
  FS-->>HTTP: 静态资源
  HTTP-->>BR: 返回页面
  BR-->>U: 浏览报告/截图
```

## 下一步

- [report 总览](./report)
- [报告生成](../advanced/reports)
- [内部 pkg/report/server](../internals/report)
