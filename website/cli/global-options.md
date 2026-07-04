# 全局选项

<p align="center">🌍 适用于所有 snir 子命令的持久化标志。</p>

全局标志通过 cobra `PersistentFlags` 注册，对所有子命令生效。

## 标志

| 标志 | 简写 | 默认 | 说明 |
|------|------|------|------|
| `--debug-log` | `-D` | `false` | 启用调试日志，输出详细 CDP 交互过程 |
| `--quiet` | `-q` | `false` | 静默（几乎所有）日志 |

## 用法

```bash
# 调试模式
snir scan example.com -D
snir scan example.com --debug-log

# 静默模式（仅输出结果）
snir scan example.com -q

# 组合：调试但静默冲突时静默优先
```

## 行为细节

::: info 静默优先于调试
`PersistentPreRunE` 中：`--quiet` 启用 → `log.EnableSilence()`；`--debug-log` 且未静默 → `log.EnableDebug()`。

同时指定 `-D -q` 时**静默优先**——不会有 CDP 细节输出，只有结果。要调试就别加 `-q`。
:::

```mermaid
flowchart TD
    Start([命令启动]) --> Q{--quiet?}
    Q -- 是 --> Silence[EnableSilence<br/>仅结果]
    Q -- 否 --> D{--debug-log?}
    D -- 是 --> Debug[EnableDebug<br/>含 CDP 细节]
    D -- 否 --> Normal[正常日志]

    style Silence fill:#3aa676,stroke:#2a7a56,color:#fff
    style Debug fill:#e6f4ea,stroke:#3aa676
```

全局标志经 `PersistentPreRunE` 注入，对任意子命令统一生效的时序：

```mermaid
sequenceDiagram
  participant U as 用户
  participant CLI as snir 根命令
  participant PRE as PersistentPreRunE
  participant LOG as 日志层
  participant SUB as 子命令执行
  U->>CLI: snir scan example.com -D -q
  CLI->>CLI: cobra 解析持久化标志
  CLI->>PRE: 进入 PreRun
  PRE->>PRE: 判断 --quiet 与 --debug-log
  alt --quiet 启用
  PRE->>LOG: EnableSilence（静默优先）
  else --debug-log 且未静默
  PRE->>LOG: EnableDebug（含 CDP 细节）
  else 两者皆无
  PRE->>LOG: 默认日志
  end
  PRE-->>SUB: 继续 RunE
  SUB-->>U: 结果输出（按日志级别）
```

## 何时用

- 🐛 **`-D` 调试**：排查失败、看 CDP 细节、定位超时原因
- 🤫 **`-q` 静默**：脚本/管线中只想要结果，减少日志噪声

## 示例：管线友好输出

```bash
snir scan file -f urls.txt -q --write-jsonl --write-stdout=false
```

## 下一步

- [CLI 总览](./overview)
- [退出码](./exit-codes)
- [故障排查](../advanced/troubleshooting)
