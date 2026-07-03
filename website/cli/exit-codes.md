# 退出码

<p align="center">🚪 snir 进程退出码语义。</p>

snir 遵循 Unix 惯例：成功退出 `0`，失败退出非零。

## 退出码

| 退出码 | 含义 |
|--------|------|
| `0` | 成功完成 |
| 非 `0` | 执行过程出错 |

退出码与单目标失败的语义需区分：

```mermaid
flowchart TD
    Run[snir scan] --> Per{逐目标执行}
    Per -->|部分失败| Rec[记 Result.Failed=true]
    Per -->|整体异常| Fatal[进程退出非 0]
    Per -->|全部成功| OK[退出 0]
    Rec --> Check[查 results.jsonl 的 failed 字段]
    Fatal --> Script["$? 非 0<br/>脚本分支]
    OK --> Script

    style OK fill:#e6f4ea,stroke:#3aa676
    style Fatal fill:#fde8e8,stroke:#d23a3a
    style Rec fill:#fff4e6,stroke:#e8a317,color:#1a1a1a
```

## 错误处理

`scan` 命令对常见错误做了美化，给出可操作建议而非原始堆栈：

| 错误特征 | 美化建议 |
|---------|---------|
| `Could not find node with given id` | 增加超时/延迟，可能反爬或结构不符 |
| `timeout` | 增加 `--timeout`，检查网络 |
| `net::ERR_*` | 网络错误，检查可达性/DNS/代理 |

详见 [错误码](../reference/error-codes) 与 [故障排查](../advanced/troubleshooting)。

## 在脚本中使用

```bash
if snir scan file -f urls.txt --write-jsonl; then
  echo "✅ 扫描成功"
else
  echo "❌ 扫描失败，退出码 $?"
fi
```

## 注意

- 部分目标失败不必然使整个进程退出非零；具体取决于命令实现。批量扫描中单个失败通常记录在 `Result.Failed` 而非中断。
- 检查 `results.jsonl` 中 `failed` 字段以确认每条结果状态。

## 下一步

- [全局选项](./global-options)
- [错误码](../reference/error-codes)
- [故障排查](../advanced/troubleshooting)
