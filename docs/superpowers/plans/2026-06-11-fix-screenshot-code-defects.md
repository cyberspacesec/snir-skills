# Fix Screenshot Code Defects

> **For agentic workers:** REQUIRED SUB-SKILL: `superpowers:subagent-driven-development`
> Steps use checkbox (`- [ ]`) syntax.

**Goal:** 修复 go-snir 截图模块中 3 个严重代码缺陷：Driver 接口参数冗余、死代码 `runner.Screenshot()` 空指针风险、`MemoryWriter.Write` 签名不匹配。

**Architecture:** 本修复不改变功能，纯粹清理死代码和修复类型签名。修复范围覆盖 `pkg/runner/` 和 `pkg/api/` 两个包。Driver 接口中 `Witness` 的 `*Runner` 参数是多余的（ChromeDP 从不使用它），移除后不影响现有调用链。`runner.Screenshot()` 依赖从未赋值的 `defaultRunner`，直接删除。`MemoryWriter.Write` 的 `*interface{}` 参数需修复为 `*models.Result`。

**Tech Stack:** Go 1.x, chromedp

**Risks:**
- Task 1 删除 Driver 接口的 `*Runner` 参数后，所有实现该接口的类型都要一致修改 → 缓解：当前只有一个实现 `ChromeDP`，修改影响面可控
- Task 3 修复 `MemoryWriter` 签名后，如果未来有人直接调用这个类型需同步更新 → 缓解：当前无调用方，纯死代码修复

---

### Task 1: 修复 Driver 接口 — 移除 Witness 方法中冗余的 runner 参数

**Depends on:** None
**Files:**
- Modify: `pkg/runner/driver.go:18-22` (Driver 接口)
- Modify: `pkg/runner/chromedp.go:83` (Witness 方法签名 + 函数体)
- Modify: `pkg/api/server_methods.go:47` (调用方)

- [ ] **Step 1: 修改 Driver 接口 — 移除 Witness 的 runner 参数**

`Witness` 方法的第二个参数 `runner *Runner` 从未在 `ChromeDP.Witness` 中被实际使用（chromedp.go 中无任何 `runner.` 引用）。移除它让接口更纯粹。

```go
// pkg/runner/driver.go:18-22（替换整个 Driver 接口）

// Driver is the interface browser drivers will implement.
type Driver interface {
	Witness(target string, opts *Options) (*models.Result, error)
	Close()
}
```

需要新增 import：

```go
// pkg/runner/driver.go（替换 import 块，新增 models 导入，移除未使用的 fmt）

import (
	"github.com/cyberspacesec/snir-skills/pkg/models"
)
```

- [ ] **Step 2: 修改 ChromeDP.Witness — 适配新接口签名**

文件: `pkg/runner/chromedp.go:83`（替换 `Witness` 方法签名，同时移除函数体内未使用的 `runner` 参数）

`ChromeDP.Witness` 内从未使用 `runner` 参数（没有 `runner.` 引用），直接删除即可。同时将原本从 `runner` 获取的 opts 改为从 `c.opts` 获取（已有 `c.opts` 字段）。

```go
// pkg/runner/chromedp.go:83（替换方法签名）

// Witness implements the Driver interface
func (c *ChromeDP) Witness(target string, opts *Options) (*models.Result, error) {
	result := &models.Result{
		URL:      target,
		ProbedAt: time.Now(),
	}
```

函数体其余部分不变（第 84-478 行保持不变），仅签名修改。

- [ ] **Step 3: 修改 ProcessScreenshot — 调用签名适配新接口**

文件: `pkg/api/server_methods.go:47`（修改 `driver.Witness` 调用行）

```go
// pkg/api/server_methods.go:47（替换调用行）
	result, err := driver.Witness(req.URL, &opts)
```

- [ ] **Step 4: 修改 chromedp_test.go — 测试签名适配新接口**

文件: `pkg/runner/chromedp_test.go:357`（修改 `driver.Witness` 调用行）

```go
// pkg/runner/chromedp_test.go:357（替换调用行）
			result, err := driver.Witness("https://example.com", opts)
```

- [ ] **Step 5: 验证编译通过**
Run: `go build ./...`
Expected:
  - Exit code: 0

- [ ] **Step 6: 验证测试通过**
Run: `go test ./pkg/runner/... ./pkg/api/... -short -count=1 2>&1`
Expected:
  - Exit code: 0
  - Output does NOT contain: "FAIL"

- [ ] **Step 7: 提交**
Run: `git add pkg/runner/driver.go pkg/runner/chromedp.go pkg/runner/chromedp_test.go pkg/api/server_methods.go && git commit -m "refactor(runner): remove unused runner param from Driver.Witness interface"`

---

### Task 2: 删除死代码 — runner.Screenshot() 函数

**Depends on:** None (可与 Task 1 并行)
**Files:**
- Modify: `pkg/runner/runner.go:47` (defaultRunner 变量)
- Modify: `pkg/runner/runner.go:123-126` (Screenshot 函数)

- [ ] **Step 1: 删除 runner.Screenshot() 函数**

`Screenshot` 函数依赖从未赋值的 `defaultRunner`，调用会导致 nil pointer dereference panic。该函数从未被任何代码调用（无 import 引用），属于死代码。

文件: `pkg/runner/runner.go:123-126`

```go
// pkg/runner/runner.go:123-126 — 删除 Screenshot 函数（4 行）
// 删除以下代码块（正式注释掉即可，因为函数从未被调用）：
```

由于 Go 编译器可能会检测到 unused function 报错，直接删除即可。

- [ ] **Step 2: 删除 defaultRunner 变量**

`defaultRunner` 从未被赋值，仅被已删除的 `Screenshot` 函数引用。

文件: `pkg/runner/runner.go:47`

```go
// pkg/runner/runner.go:47 — 删除该行
// var defaultRunner *Runner  // ← 删除此行
```

- [ ] **Step 3: 验证编译通过**
Run: `go build ./...`
Expected:
  - Exit code: 0

- [ ] **Step 4: 验证测试通过**
Run: `go test ./pkg/runner/... -short -count=1 2>&1`
Expected:
  - Exit code: 0
  - Output does NOT contain: "FAIL"

- [ ] **Step 5: 提交**
Run: `git add pkg/runner/runner.go && git commit -m "refactor(runner): remove dead code Screenshot() and defaultRunner"`

---

### Task 3: 修复 MemoryWriter.Write 签名 — 参数类型从 *interface{} 改为 *models.Result

**Depends on:** None (可与 Task 1、2 并行)
**Files:**
- Modify: `pkg/api/helpers.go:86-94` (MemoryWriter.Write 方法)
- Modify: `pkg/api/helpers.go:97-99` (MemoryWriter.Close 方法)

- [ ] **Step 1: 修复 MemoryWriter.Write 方法签名和实现**

`MemoryWriter` 声称实现了 `runner.Writer` 接口，但 `Write` 方法参数是 `*interface{}`，与接口要求的 `*models.Result` 不匹配。修复签名使其与接口一致。

```go
// pkg/api/helpers.go:86-94（替换整个 Write 方法）

// Write 实现 runner.Writer 接口的写入方法
func (w *MemoryWriter) Write(result *models.Result) error {
	w.mu.Lock()
	w.Results = append(w.Results, result)
	w.mu.Unlock()
	return nil
}
```

- [ ] **Step 2: 验证编译通过**
Run: `go build ./...`
Expected:
  - Exit code: 0

- [ ] **Step 3: 验证测试通过**
Run: `go test ./pkg/api/... -short -count=1 2>&1`
Expected:
  - Exit code: 0
  - Output does NOT contain: "FAIL"

- [ ] **Step 4: 提交**
Run: `git add pkg/api/helpers.go && git commit -m "fix(api): fix MemoryWriter.Write signature to match runner.Writer interface"`
```