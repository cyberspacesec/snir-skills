# scan 命令族

<p align="center">📸 snir scan — 扫描并截图网站。</p>

`scan` 是 snir 最核心的命令，负责扫描指定的 URL、文件或网段，并对网站截图与信息收集。

三种输入入口共享同一套扫描流水线：

```mermaid
flowchart LR
    I1["single [url]"] --> Expand[目标展开]
    I2["file -f"] --> Expand
    I3["cidr [cidr]"] --> Expand
    Direct["直接传 URL<br/>等价 single"] --> Expand
    Expand --> BL[黑名单过滤]
    BL --> Pool[并发池<br/>--threads]
    Pool --> Runner[runner 截图+证据]
    Runner --> Out[截图/jsonl/csv/db]

    style Expand fill:#3aa676,stroke:#2a7a56,color:#fff
    style Runner fill:#e6f4ea,stroke:#3aa676
    style Out fill:#e6f4ea,stroke:#3aa676
```

单个目标在扫描流水线内的执行时序：

```mermaid
sequenceDiagram
  participant CLI as scan 命令
  participant EX as 目标展开
  participant BL as 黑名单
  participant POOL as 并发池
  participant DR as Driver/Chrome
  participant W as Writer
  CLI->>EX: url/file/cidr
  EX-->>CLI: 候选 URL 列表
  CLI->>BL: 逐目标过黑名单
  BL-->>CLI: 通过的 URL
  CLI->>POOL: --threads 并发下发
  POOL->>DR: 导航+截图+证据
  DR-->>POOL: Result
  POOL->>W: 写 jsonl/csv/db/screenshots
  POOL-->>CLI: 完成（单点失败不中断）
```

## 子命令

| 子命令 | 用法 | 说明 |
|--------|------|------|
| `single [url]` | `snir scan single https://example.com` | 扫描单个 URL |
| `file` | `snir scan file -f urls.txt` | 从文件批量扫描 |
| `cidr [cidr]` | `snir scan cidr 192.168.1.0/24` | 扫描网段 |
| （直接 URL） | `snir scan example.com` | 等价于 single |

## 直接传 URL

`scan` 后直接给一个 URL 参数，视为单 URL 扫描：

```bash
snir scan example.com
snir scan https://example.com/path
```

## 设备预设

`--device <name>` 应用设备预设（视口/UA/像素比/触摸）：

```bash
snir scan example.com --device iphone-15
```

`--list-devices` 列出全部预设：

```bash
snir scan --list-devices
```

见 [设备模拟](./scan-device)。

## 公共标志

`scan` 的持久化标志对所有子命令生效，涵盖：截图、证据、Chrome、代理、Cookie、设备、JS、输出、数据库、黑名单、端口。完整表见 [CLI 标志全表](../reference/cli-flags)。

::: info 持久化标志 = 在 `scan` 或子命令上都可挂
`--save-html` 这类标志既可写 `snir scan --save-html file -f urls.txt`，也可写 `snir scan file --save-html -f urls.txt`——cobra 的持久化标志贯穿整棵子命令树。
:::

## 示例

::: tip 常用配方速查
| 配方 | 命令 |
|------|------|
| 单页快照 | `snir scan example.com` |
| 批量归档 | `snir scan file -f urls.txt --threads 10 --write-jsonl --db` |
| 全量证据 | `snir scan example.com --full-page --save-html --save-headers --save-cookies --save-console --save-network` |
| 移动端视角 | `snir scan example.com --device iphone-15` |
| 网段普查 | `snir scan cidr 192.168.1.0/24` |
| 走代理 | `snir scan example.com --proxy http://127.0.0.1:8080` |
| 高清桌面 | `snir scan example.com --resolution-x 1920 --resolution-y 1080` |
:::

## 下一步

- [scan single](./scan-single)
- [scan file](./scan-file)
- [scan cidr](./scan-cidr)
- 各选项专题：[截图](./scan-screenshot)、[证据](./scan-evidence)、[Chrome](./scan-chrome)、[代理](./scan-proxy)、[Cookie](./scan-cookie)、[设备](./scan-device)、[JS](./scan-js)、[输出](./scan-output)、[数据库](./scan-db)、[黑名单](./scan-blacklist)、[端口](./scan-ports)
