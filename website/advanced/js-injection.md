# JS 注入

<p align="center">☕ 在页面执行自定义 JavaScript。</p>

## 注入方式

| 方式 | CLI | SDK | 时机 |
|------|-----|-----|------|
| 内联 | `--js` | `WithJS` / `WithJSAfter` | 加载后 |
| 文件 | `--js-file` | `WithJSFile` | 加载后 |
| 加载前 | `--run-js-before` | `WithJSBefore` | 加载前 |

## 执行时序

```mermaid
flowchart LR
  J1[JSBefore 加载前] --> N[导航加载]
  N --> A[Actions 交互]
  A --> J2[JSAfter 加载后]
  J2 --> S[截图]
```

## 典型用例

```bash
# 滚动到底触发懒加载
snir scan example.com --js "window.scrollTo(0, document.body.scrollHeight)"

# 关闭 cookie 弹窗
snir scan example.com --js "document.querySelector('.consent')?.remove()"

# 加载前 hook
snir scan example.com --js-file preload.js --run-js-before
```

## SDK 示例

```go
opts := sdk.NewScreenshotOptions(
    sdk.WithJSBefore("document.querySelector('.consent')?.remove()"),
    sdk.WithJS("window.scrollTo(0, document.body.scrollHeight)"),
)
```

## 与交互动作的区别

- `--js`：自由 JS，灵活但需自处理异步
- `WithActions`：结构化动作（点击/输入/滚动/等待/hover），见 [表单与交互](./forms)

## 注意

- JS 错误不必然中断截图，但可能影响结果
- 复杂异步逻辑建议用 `ActionWaitVisible` 等待完成再截图

## 下一步

- [JS 注入 CLI](../cli/scan-js)
- [JS 与交互构建器](../sdk/builder-js)
- [表单与交互](./forms)
