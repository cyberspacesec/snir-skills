# snir version — 版本信息与全局调试标志

> **渐进式披露**：[快速上手](#快速上手) → [全局标志详解](#全局标志详解)

---

## 快速上手

```bash
# 显示版本信息
./snir version
```

输出示例：
```
snir version: 1.0.0
```

---

## 全局标志详解

以下标志继承到 **所有命令**（`scan`、`api`、`provider`、`report`、`webserve`、`version`）：

### --debug-log / -D

启用调试日志，输出 ChromeDP 内部通信细节。适用于排查截图失败原因。

```bash
# 调试模式
./snir scan example.com -D

# 等价写法
./snir scan example.com --debug-log

# 在 API 服务中调试
./snir api --port 8080 -D
```

**什么时候用：**
- 截图失败需要查看详细错误信息
- 页面加载异常需要追踪 Chrome 通信
- 提交 Bug 报告时收集诊断信息

**输出内容：**
- ChromeDP WebSocket 通信日志
- 页面加载事件序列
- JavaScript 执行结果
- 网络请求详情

### --quiet / -q

静默模式，几乎不输出任何日志。适用于脚本自动化、管道操作。

```bash
# 静默模式
./snir scan file -f urls.txt -q

# 等价写法
./snir scan file -f urls.txt --quiet

# 静默 + JSONL 输出（纯数据，无日志干扰）
./snir scan file -f urls.txt -q --write-jsonl
```

**什么时候用：**
- 脚本自动化，只需要最终结果
- 管道操作，避免日志污染数据流
- 批量任务，减少终端噪音

**注意事项：**
- `--quiet` 和 `--debug-log` 同时使用时，`--quiet` 优先级更高
- 静默模式下，错误信息仍会输出到 stderr

---

## 完整参数参考

| 标志 | 短写 | 类型 | 默认值 | 说明 |
|------|------|------|--------|------|
| `--debug-log` | `-D` | bool | `false` | 启用调试日志 |
| `--quiet` | `-q` | bool | `false` | 静默模式 |

---

## 实战组合示例

```bash
# 调试截图失败
./snir scan problematic-site.com -D --timeout 60

# 脚本自动化（静默 + 输出文件）
./snir scan file -f urls.txt -q --write-jsonl --jsonl-file results.jsonl

# 调试 API 服务
./snir api --port 8080 -D

# 调试 Provider
./snir provider -D --headless=false
```