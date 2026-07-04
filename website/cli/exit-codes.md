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

按退出码类别归类的心智图：

```mermaid
mindmap
  root((退出码))
    0 成功
      全部目标完成
      脚本判断 $? 为 0
    非 0 致命
      参数错误
      Chrome 找不到
      配置非法
      进程级异常
    单目标 failed
      记入 results.jsonl
      failed=true
      不影响进程退出码
      需 jq 查询
    脚本友好
      $? 反映进程整体
      failed 字段反映单点
      两者结合判断
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

::: warning 退出码 ≠ 单目标成败
这是最容易误解的点：

- **进程退出码**：反映 snir 进程**整体**是否正常结束（参数错、Chrome 找不到等致命错误才非零）
- **单目标 `failed` 字段**：批量扫描中某个 URL 失败**通常不会**让进程退出非零，而是记在 `results.jsonl` 的 `failed=true`

→ 脚本里判断"扫没扫好"不能只看 `$?`，**必须查 `failed` 字段**：
```bash
snir scan file -f urls.txt --write-jsonl
# $? 为 0 不代表每个 URL 都成功
jq 'select(.failed == true)' results.jsonl   # 这才是真正的失败列表
```
:::

## 下一步

- [全局选项](./global-options)
- [错误码](../reference/error-codes)
- [故障排查](../advanced/troubleshooting)
