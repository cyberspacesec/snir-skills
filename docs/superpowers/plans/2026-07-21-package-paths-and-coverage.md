# Go 包路径一致性 + 单元测试覆盖率补全 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: `superpowers:subagent-driven-development`
> Steps use checkbox (`- [ ]`) syntax.

**Goal:** (1) 确保所有 Go 包的 import 路径与 GitHub 仓库 `github.com/cyberspacesec/snir-skills` 保持一致并清理死目录/产物；(2) 把纯逻辑包的单元测试覆盖率补全到 100%，整体覆盖率提升到 ≥90%，并对架构受限包（浏览器/CLI 入口）的纯函数分支补测，透明说明不可达部分。

**Architecture:** 诉求 1 的调研结论是 module 路径已一致，真正待修的是空目录 `pkg/api/screenshots/`、未忽略的 `results.jsonl`、工作区残留 build 产物——通过删除空目录 + 完善 .gitignore + 加 `go list` 守卫解决。诉求 2 按包分层补测：纯逻辑包（database/ascii/scan/techdetect/phash/models/report/sdk/api）的 0% 与低覆盖函数逐个补表驱动测试到 100%；架构受限包（provider/runner）的 HTTP handler 与纯函数用 `httptest`/纯输入补测，浏览器交互路径用 `SKIP_BROWSER_TESTS` 守卫并明确标注不可达；cmd 包新增首个测试文件覆盖 `proxyStrategyFlag`/`inc`/`generateRandomAPIKey` 等纯函数。所有新测试必须 `go test -race` 通过。

**Tech Stack:** Go 1.26.5, 标准 `testing` + 表驱动测试, `net/http/httptest`, `go tool cover`, `github.com/cyberspacesec/snir-skills/pkg/*`

**Risks:**
- provider/runner 浏览器交互路径在 CI 因 `SKIP_BROWSER_TESTS` 跳过，无法达 100% → 缓解：Plan 明确标注受限函数，目标设为"纯逻辑包 100% + 整体 ≥90%"，不假称全包 100%
- 补测可能暴露既有 data race（[[snir-ci-browser-tests-race]] 已记录同类问题）→ 缓解：每个 Task 验证 Step 强制 `go test -race`
- cmd 包首次引入测试，`init()` 副作用可能干扰 → 缓解：只测纯函数，不触发 cobra Execute
- 修改 .gitignore 可能影响其他流程 → 缓解：只追加 `results.jsonl`，不动现有条目

---

### Task 1: 清理包路径死目录与产物，加 CI 守卫

**Depends on:** None
**Files:**
- Delete: `pkg/api/screenshots/`（空目录）
- Modify: `.gitignore`（追加 `results.jsonl`）
- Modify: `.github/workflows/ci.yml`（quality job 加包路径一致性校验）
- Modify: `Makefile`（补 `coverage-check` 阈值目标）

- [ ] **Step 1: 删除空目录 pkg/api/screenshots/ — Go 不允许空包，死目录应清理**

```bash
# 空目录，无文件，直接删除
rmdir pkg/api/screenshots
```

验证目录确为空（已在调研确认 `ls` 仅返回 `.` 和 `..`）。

- [ ] **Step 2: 完善 .gitignore 追加 results.jsonl — 扫描产物不应被跟踪**

文件: `.gitignore`（在 `coverage.txt` 行之后追加）

```text

# 扫描产物（运行时生成，不应入库）
results.jsonl
```

- [ ] **Step 3: 修改 ci.yml quality job 加包路径一致性校验 — 防止未来 import 路径与目录脱节**

文件: `.github/workflows/ci.yml:58-59`（在 `Run go vet` 步骤之后插入新步骤）

```yaml
      - name: Verify package paths match directory structure
        run: |
          # 校验：每个 .go 文件的 package 声明必须与其所在目录名一致（根包为 main）
          # 校验：无相对路径 import，所有内部 import 必须以 module 路径为前缀
          mismatches=0
          while IFS= read -r f; do
            pkg=$(grep -m1 '^package ' "$f" | awk '{print $2}')
            dir=$(dirname "$f")
            if [ "$dir" = "." ]; then expected="main"; else expected=$(basename "$dir"); fi
            if [ "$pkg" != "$expected" ]; then
              echo "MISMATCH: $f package=$pkg expected=$expected"
              mismatches=$((mismatches+1))
            fi
          done < <(find . -name '*.go' ! -name '*_test.go' -not -path './vendor/*')
          if [ "$mismatches" -gt 0 ]; then
            echo "::error::Found $mismatches package/directory mismatches"
            exit 1
          fi
          # 校验：无相对路径 import
          if grep -rn 'import' --include='*.go' . | grep -E '"\.\.?/'; then
            echo "::error::Found relative import path"
            exit 1
          fi
          echo "All package paths consistent with directory structure."
```

- [ ] **Step 4: 修改 Makefile 补 coverage-check 阈值目标 — 让覆盖率有可执行门槛**

文件: `Makefile`（在 `coverage:` 目标之后追加）

```makefile
coverage-check: coverage
	@echo "检查覆盖率阈值（整体 >= 90%）..."
	@total=$$(go tool cover -func=coverage.out | tail -1 | awk '{print $$3}' | tr -d '%'); \
	if [ -z "$$total" ]; then echo "::error::无法解析覆盖率"; exit 1; fi; \
	echo "当前整体覆盖率: $$total%"; \
	if [ "$$total" -lt 90 ]; then \
		echo "::error::覆盖率 $$total% 低于阈值 90%"; exit 1; \
	else \
		echo "覆盖率达标"; \
	fi
```

- [ ] **Step 5: 验证 Task 1 变更**

Run: `rmdir pkg/api/screenshots 2>/dev/null; git status --short && SKIP_BROWSER_TESTS=1 go build ./... && go vet ./...`
Expected:
  - Exit code: 0
  - Output does NOT contain: "error" or "MISMATCH"
  - `git status` 显示 `.gitignore`、`.github/workflows/ci.yml`、`Makefile` 已修改

Run: `make coverage-check 2>&1 | tail -5`
Expected:
  - Output contains: "当前整体覆盖率:"
  - （此时覆盖率约 75-80%，会显示"低于阈值 90%"——这是 Task 2-6 补测前的预期状态，Task 6 后将达标）

- [ ] **Step 6: 提交**
Run: `git add pkg/api/screenshots .gitignore .github/workflows/ci.yml Makefile && git commit -m "chore(repo): clean dead dir + ignore results.jsonl + add package-path CI guard"`

---

### Task 2: 补全 pkg/phash 与 pkg/models 纯函数测试到 100%

**Depends on:** Task 1
**Files:**
- Modify: `pkg/phash/phash_test.go`（补 Distance / HexToHashValue / ComputeHash 等分支）
- Modify: `pkg/models/models_test.go`（补包级 EnrichEndpoint / DefaultPortForScheme）

- [ ] **Step 1: 修改 pkg/phash/phash_test.go 补 Distance 测试 — 覆盖 0% 的 Distance 函数**

文件: `pkg/phash/phash_test.go`（在文件末尾追加）

```go
func TestDistance_ValidHashes(t *testing.T) {
	// 用 ComputePerceptionHash 产出真实可解析的 hash 字符串，确保 Distance 走通
	h1 := ComputeAverageHash(image.NewRGBA(image.Rect(0, 0, 8, 8)))
	h2 := ComputeAverageHash(image.NewRGBA(image.Rect(0, 0, 8, 8)))
	dist, err := Distance(h1, h2)
	if err != nil {
		t.Fatalf("Distance 返回错误: %v", err)
	}
	if dist < 0 {
		t.Fatalf("Distance 不应为负: %d", dist)
	}
	// 相同图像 ahash 应距离 0
	if dist != 0 {
		t.Fatalf("相同图像 Distance 应为 0, 实际 %d", dist)
	}
}

func TestDistance_InvalidHash(t *testing.T) {
	// 非法 hash 字符串应返回错误
	_, err := Distance("not-a-valid-hash", "also-invalid")
	if err == nil {
		t.Fatal("非法 hash 应返回错误")
	}
}

func TestDistanceFromValues(t *testing.T) {
	tests := []struct {
		name string
		a, b uint64
		want int
	}{
		{"相同值", 0x1234, 0x1234, 0},
		{"全异", 0x00FF, 0xFF00, 16},
		{"单bit差异", 0x0001, 0x0000, 1},
		{"全零", 0, 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DistanceFromValues(tt.a, tt.b); got != tt.want {
				t.Fatalf("DistanceFromValues(%#x,%#x) = %d, want %d", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestIsSimilar(t *testing.T) {
	if !IsSimilar(0x1234, 0x1234, 5) {
		t.Fatal("相同 hash 应判为相似")
	}
	if IsSimilar(0x00FF, 0xFF00, 5) {
		t.Fatal("距离 16 超过阈值 5 不应判为相似")
	}
	if !IsSimilar(0x0001, 0x0000, 5) {
		t.Fatal("距离 1 在阈值 5 内应判为相似")
	}
}

func TestHexToHashValue(t *testing.T) {
	v, err := HexToHashValue("0000000000000001")
	if err != nil {
		t.Fatalf("HexToHashValue 错误: %v", err)
	}
	if v != 1 {
		t.Fatalf("HexToHashValue = %#x, want 1", v)
	}
	// 非法输入
	if _, err := HexToHashValue("xyz"); err == nil {
		t.Fatal("非法 hex 应返回错误")
	}
	if _, err := HexToHashValue(""); err == nil {
		t.Fatal("空字符串应返回错误")
	}
}
```

- [ ] **Step 2: 修改 pkg/models/models_test.go 补 EnrichEndpoint 与 DefaultPortForScheme — 覆盖 0% 的包级 EnrichEndpoint**

文件: `pkg/models/models_test.go`（在文件末尾追加）

```go
func TestEnrichEndpoint_PackageLevel_NilSafe(t *testing.T) {
	// 包级 EnrichEndpoint 对 nil 应安全无 panic
	EnrichEndpoint(nil) // 不应 panic

	r := &Result{URL: "http://example.com:8080/path"}
	EnrichEndpoint(r)
	// 方法级 EnrichEndpoint 应已填充 Host/Port/Scheme
	if r.Host == "" {
		t.Fatal("EnrichEndpoint 后 Host 不应为空")
	}
}

func TestDefaultPortForScheme(t *testing.T) {
	tests := []struct {
		scheme string
		want   int
	}{
		{"http", 80},
		{"https", 443},
		{"HTTP", 80},   // 大小写不敏感
		{"HTTPS", 443},
		{"ftp", 0},     // 未知 scheme
		{"", 0},        // 空 scheme
	}
	for _, tt := range tests {
		t.Run(tt.scheme, func(t *testing.T) {
			if got := DefaultPortForScheme(tt.scheme); got != tt.want {
				t.Fatalf("DefaultPortForScheme(%q) = %d, want %d", tt.scheme, got, tt.want)
			}
		})
	}
}

func TestResult_HeaderMap(t *testing.T) {
	r := &Result{
		Headers: []Header{{Name: "X-A", Value: "1"}, {Name: "X-A", Value: "2"}, {Name: "X-B", Value: "3"}},
	}
	m := r.HeaderMap()
	if len(m["X-A"]) != 2 || m["X-A"][0] != "1" || m["X-A"][1] != "2" {
		t.Fatalf("HeaderMap X-A = %v", m["X-A"])
	}
	if len(m["X-B"]) != 1 || m["X-B"][0] != "3" {
		t.Fatalf("HeaderMap X-B = %v", m["X-B"])
	}
}
```

- [ ] **Step 3: 验证 phash 与 models 覆盖率**

Run: `SKIP_BROWSER_TESTS=1 go test -race -cover ./pkg/phash/ ./pkg/models/`
Expected:
  - Exit code: 0
  - Output contains: "ok" for both packages
  - pkg/phash coverage >= 95%
  - pkg/models coverage >= 95%

- [ ] **Step 4: 提交**
Run: `git add pkg/phash/phash_test.go pkg/models/models_test.go && git commit -m "test(phash,models): cover Distance/EnrichEndpoint/HeaderMap pure functions"`

---

### Task 3: 补全 pkg/sdk builder 与 BatchScreenshot 测试

**Depends on:** Task 1
**Files:**
- Modify: `pkg/sdk/builders_test.go`（补 WithProxyStrategy/WithHeaders/WithConsole/WithNetwork）
- Modify: `pkg/sdk/client_test.go`（补 BatchScreenshot error path）

- [ ] **Step 1: 修改 pkg/sdk/builders_test.go 补 4 个 0% builder — 覆盖 WithProxyStrategy/WithHeaders/WithConsole/WithNetwork**

文件: `pkg/sdk/builders_test.go`（在文件末尾追加）

```go
func TestWithProxyStrategy(t *testing.T) {
	tests := []runner.ProxyStrategy{
		runner.ProxyRoundRobin,
		runner.ProxyRandom,
		runner.ProxySequential,
	}
	for _, s := range tests {
		opts := NewScreenshotOptions()
		WithProxyStrategy(s)(opts)
		if opts.ProxyStrategy != s {
			t.Fatalf("WithProxyStrategy(%s) = %s", s, opts.ProxyStrategy)
		}
	}
}

func TestWithHeaders(t *testing.T) {
	opts := NewScreenshotOptions()
	WithHeaders()(opts)
	if !opts.SaveHeaders {
		t.Fatal("WithHeaders 应设置 SaveHeaders=true")
	}
}

func TestWithConsole(t *testing.T) {
	opts := NewScreenshotOptions()
	WithConsole()(opts)
	if !opts.SaveConsole {
		t.Fatal("WithConsole 应设置 SaveConsole=true")
	}
}

func TestWithNetwork(t *testing.T) {
	opts := NewScreenshotOptions()
	WithNetwork()(opts)
	if !opts.SaveNetwork {
		t.Fatal("WithNetwork 应设置 SaveNetwork=true")
	}
}

func TestWithEvidence_SetsAll(t *testing.T) {
	// 顺带加固 WithEvidence 的全部字段断言（已有间接覆盖，此处显式）
	opts := NewScreenshotOptions()
	WithEvidence()(opts)
	if !opts.SaveHTML || !opts.SaveHeaders || !opts.SaveConsole || !opts.SaveCookies || !opts.SaveNetwork {
		t.Fatalf("WithEvidence 应设置全部证据字段, got %+v", opts)
	}
}
```

- [ ] **Step 2: 修改 pkg/sdk/client_test.go 补 BatchScreenshot error path — 覆盖 0% 的 BatchScreenshot 与并发聚合**

文件: `pkg/sdk/client_test.go`（在文件末尾追加）

说明：`BatchScreenshot` 是 `BatchScreenshotWithContext` 的薄包装，内部对每个 URL 调 `ScreenshotWithContext`。用一个未配置 pool 的 `Client`（pool 为 nil 或返回 error）触发 error path，验证 `BatchResult` 聚合正确。先读取现有 client_test.go 的 mock/fake 构造方式确认可注入方式。

```go
func TestBatchScreenshot_EmptyURLs(t *testing.T) {
	// 空 URL 列表：应返回空切片，不 panic
	c := &Client{}
	results := c.BatchScreenshot(nil, NewScreenshotOptions())
	if len(results) != 0 {
		t.Fatalf("空列表应返回 0 结果, got %d", len(results))
	}
}

func TestBatchScreenshot_ErrorAggregation(t *testing.T) {
	// 无可用 provider 的 Client：每个 URL 应聚合为带 Error 的 BatchResult
	c := &Client{}
	urls := []string{"http://invalid-a.test", "http://invalid-b.test"}
	results := c.BatchScreenshot(urls, NewScreenshotOptions())
	if len(results) != 2 {
		t.Fatalf("应返回 2 结果, got %d", len(results))
	}
	for i, r := range results {
		if r.URL != urls[i] {
			t.Fatalf("results[%d].URL = %q, want %q", i, r.URL, urls[i])
		}
		if r.Error == nil {
			t.Fatalf("无 provider 时 results[%d] 应有 Error", i)
		}
	}
}

func TestBatchScreenshotWithContext_PreservesOrder(t *testing.T) {
	// 并发执行但结果顺序应与输入一致（results[idx] 按 idx 写入）
	c := &Client{}
	urls := []string{"http://a.test", "http://b.test", "http://c.test"}
	results := c.BatchScreenshotWithContext(context.Background(), urls, NewScreenshotOptions())
	if len(results) != len(urls) {
		t.Fatalf("结果数 %d != 输入 %d", len(results), len(urls))
	}
	for i, r := range results {
		if r.URL != urls[i] {
			t.Fatalf("顺序错乱: results[%d].URL=%q want %q", i, r.URL, urls[i])
		}
	}
}
```

- [ ] **Step 3: 验证 sdk 覆盖率**

Run: `SKIP_BROWSER_TESTS=1 go test -race -cover ./pkg/sdk/`
Expected:
  - Exit code: 0
  - Output contains: "ok"
  - pkg/sdk coverage >= 80%（从 75.1% 提升）

- [ ] **Step 4: 提交**
Run: `git add pkg/sdk/builders_test.go pkg/sdk/client_test.go && git commit -m "test(sdk): cover builder options + BatchScreenshot error aggregation"`

---

### Task 4: 补全 pkg/provider HTTP handler 与纯函数测试

**Depends on:** Task 1
**Files:**
- Modify: `pkg/provider/provider_test.go`（补 handleHealth/handleStats 的 httptest 测试）

- [ ] **Step 1: 修改 pkg/provider/provider_test.go 补 handleHealth 与 handleStats — 用 httptest 覆盖 0% handler**

文件: `pkg/provider/provider_test.go`（在文件末尾追加）

说明：`handleHealth`/`handleStats` 是纯 HTTP handler，用 `httptest.NewRecorder` + `http.NewRequest` 直接调用，无需真实 Chrome。先读取 provider.go 中 Provider 的 `ServeHTTP` 路由或 handler 暴露方式（是否可通过构造未启动 pool 的 Provider 直接调 handler）。若 handler 未导出，通过 `ServeHTTP` + 路径 `/health`、`/stats` 触发。

```go
func TestProvider_HealthHandler_NotInitialized(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过 handler 测试")
	}
	if os.Getenv("SKIP_BROWSER_TESTS") != "" {
		// handler 测试本身不需要浏览器，但若整个包被跳过则一并跳过
	}
	p := &Provider{} // pool == nil 分支
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()
	p.handleHealth(rr, req)
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("未初始化 health 应返回 503, got %d", rr.Code)
	}
}

func TestProvider_StatsHandler_NotInitialized(t *testing.T) {
	p := &Provider{}
	req := httptest.NewRequest(http.MethodGet, "/stats", nil)
	rr := httptest.NewRecorder()
	p.handleStats(rr, req)
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("未初始化 stats 应返回 503, got %d", rr.Code)
	}
}

func TestProvider_ScreenshotHandler_MethodNotAllowed(t *testing.T) {
	p := &Provider{}
	// GET /screenshot 应返回 405（handleScreenshot 校验 Method != POST）
	req := httptest.NewRequest(http.MethodGet, "/screenshot?url=http://x.test", nil)
	rr := httptest.NewRecorder()
	p.handleScreenshot(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("非 POST 应返回 405, got %d", rr.Code)
	}
}
```

- [ ] **Step 2: 验证 provider 覆盖率提升**

Run: `SKIP_BROWSER_TESTS=1 go test -race -cover ./pkg/provider/`
Expected:
  - Exit code: 0
  - Output contains: "ok"
  - pkg/provider coverage >= 55%（从 43.9% 提升，handler 分支被覆盖）
  - （Start/StartWithContext/handleScreenshot 真实截图路径仍受 SKIP_BROWSER_TESTS 限制，不可达 100%——这是预期）

- [ ] **Step 3: 提交**
Run: `git add pkg/provider/provider_test.go && git commit -m "test(provider): cover health/stats/screenshot HTTP handlers via httptest"`

---

### Task 5: 补全 pkg/runner 纯函数与 cmd 包首个测试文件

**Depends on:** Task 1
**Files:**
- Modify: `pkg/runner/blacklist_test.go`（补 loadPatternsFromFile）
- Create: `cmd/cmd_test.go`（cmd 包首个测试，覆盖 proxyStrategyFlag/inc/generateRandomAPIKey）

- [ ] **Step 1: 修改 pkg/runner/blacklist_test.go 补 loadPatternsFromFile — 覆盖 0% 的文件加载函数**

文件: `pkg/runner/blacklist_test.go`（在文件末尾追加）

```go
func TestLoadPatternsFromFile(t *testing.T) {
	// 写一个临时黑名单文件，每行一个模式
	tmp := t.TempDir()
	path := tmp + "/blacklist.txt"
	content := "*.evil.com\n# 这是注释\nspam.test\n\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("写临时文件失败: %v", err)
	}
	patterns, err := loadPatternsFromFile(path)
	if err != nil {
		t.Fatalf("loadPatternsFromFile 错误: %v", err)
	}
	// 应包含两个非空非注释行（具体过滤逻辑以 parsePatterns 为准，此处至少验证读取成功）
	if len(patterns) == 0 {
		t.Fatal("应至少读取到 1 个模式")
	}
}

func TestLoadPatternsFromFile_NotExist(t *testing.T) {
	_, err := loadPatternsFromFile("/nonexistent/path/blacklist.txt")
	if err == nil {
		t.Fatal("文件不存在应返回错误")
	}
}
```

- [ ] **Step 2: 创建 cmd/cmd_test.go — cmd 包首个测试文件，覆盖纯函数**

```go
// cmd/cmd_test.go
package cmd

import (
	"net"
	"testing"

	"github.com/cyberspacesec/snir-skills/pkg/runner"
)

func TestProxyStrategyFlag_Set(t *testing.T) {
	tests := []struct {
		input string
		want  runner.ProxyStrategy
	}{
		{"round-robin", runner.ProxyRoundRobin},
		{"random", runner.ProxyRandom},
		{"sequential", runner.ProxySequential},
		{"", runner.ProxyRoundRobin},          // 空串默认 round-robin
		{"unknown", runner.ProxyRoundRobin},   // 未知值默认 round-robin
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			var s runner.ProxyStrategy
			f := &proxyStrategyFlag{value: &s}
			if err := f.Set(tt.input); err != nil {
				t.Fatalf("Set(%q) 错误: %v", tt.input, err)
			}
			if s != tt.want {
				t.Fatalf("Set(%q) => %s, want %s", tt.input, s, tt.want)
			}
		})
	}
}

func TestProxyStrategyFlag_StringAndType(t *testing.T) {
	s := runner.ProxyRandom
	f := &proxyStrategyFlag{value: &s}
	if f.String() != "random" {
		t.Fatalf("String() = %q, want random", f.String())
	}
	if f.Type() != "string" {
		t.Fatalf("Type() = %q, want string", f.Type())
	}
	// nil value 分支
	var nilF proxyStrategyFlag
	if nilF.String() != "" {
		t.Fatalf("nil value String() 应为空串, got %q", nilF.String())
	}
}

func TestInc_IPIncrement(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"末字节+1", "192.168.1.1", "192.168.1.2"},
		{"末字节进位", "192.168.1.255", "192.168.2.0"},
		{"全进位", "10.0.0.255", "10.0.1.0"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := net.ParseIP(tt.in).To4()
			if ip == nil {
				t.Fatalf("非法 IP: %s", tt.in)
			}
			inc(ip)
			if got := ip.String(); got != tt.want {
				t.Fatalf("inc(%s) = %s, want %s", tt.in, got, tt.want)
			}
		})
	}
}

func TestGenerateRandomAPIKey(t *testing.T) {
	// 正常路径：返回 hex 编码，长度 = length/2*2
	key := generateRandomAPIKey(16)
	if len(key) != 16 {
		t.Fatalf("长度 16 请求应返回 16 字符 hex, got %d", len(key))
	}
	// 两次调用应不同（随机性）
	key2 := generateRandomAPIKey(16)
	if key == key2 {
		t.Fatal("两次随机密钥不应相同")
	}
}

func TestPrintResult_DoesNotPanic(t *testing.T) {
	// printResult 只调用 log.Success，验证不 panic
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("printResult panic: %v", r)
		}
	}()
	printResult("any")
}
```

- [ ] **Step 3: 验证 runner 与 cmd 覆盖率**

Run: `SKIP_BROWSER_TESTS=1 go test -race -cover ./pkg/runner/ ./cmd/`
Expected:
  - Exit code: 0
  - Output contains: "ok" for both
  - pkg/runner coverage >= 56%（从 54.5% 提升）
  - cmd coverage > 0%（从 0% 提升，纯函数被覆盖）

- [ ] **Step 4: 提交**
Run: `git add pkg/runner/blacklist_test.go cmd/cmd_test.go && git commit -m "test(runner,cmd): cover loadPatternsFromFile + cmd pure functions (first cmd test)"`

---

### Task 6: 补全 pkg/api / pkg/report / pkg/database / pkg/scan / pkg/techdetect / pkg/ascii 残余 gap 并达整体阈值

**Depends on:** Task 1
**Files:**
- Modify: `pkg/api/helpers_test.go`（补 SendJSONResponse 错误分支）
- Modify: `pkg/report/html_test.go`（补 getStatusClass）
- Modify: `pkg/database/host_query_test.go`（补 GetScreenshotsByHost/ByURL 边界）
- Modify: `pkg/scan/scan_test.go`（补 hasExplicitPort / NewPooledScanner 边界）
- Modify: `pkg/techdetect/detector_test.go`（补 matchFingerprint 边界）
- Modify: `pkg/ascii/ascii_test.go`（补 Markdown 边界）

- [ ] **Step 1: 修改 pkg/api/helpers_test.go 补 SendJSONResponse — 覆盖错误分支**

文件: `pkg/api/helpers_test.go`（在文件末尾追加）

```go
func TestSendJSONResponse_Success(t *testing.T) {
	rr := httptest.NewRecorder()
	SendJSONResponse(rr, http.StatusOK, map[string]string{"k": "v"})
	if rr.Code != http.StatusOK {
		t.Fatalf("状态码 = %d, want 200", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", ct)
	}
	body := strings.TrimSpace(rr.Body.String())
	if !strings.Contains(body, `"k":"v"`) && !strings.Contains(body, `"k": "v"`) {
		t.Fatalf("响应体应包含 k:v, got %s", body)
	}
}

func TestSendJSONResponse_EncodeError(t *testing.T) {
	// 传入无法被 json.Marshal 的值（如 channel）触发 encode 错误分支
	rr := httptest.NewRecorder()
	SendJSONResponse(rr, http.StatusOK, map[string]interface{}{"ch": make(chan int)})
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("编码失败应返回 500, got %d", rr.Code)
	}
}
```

- [ ] **Step 2: 修改 pkg/report/html_test.go 补 getStatusClass — 覆盖状态码分类分支**

文件: `pkg/report/html_test.go`（在文件末尾追加）

```go
func TestGetStatusClass(t *testing.T) {
	tests := []struct {
		status int
		want   string
	}{
		{200, "success"},
		{299, "success"},
		{301, "redirect"},
		{399, "redirect"},
		{400, "client-error"},
		{404, "client-error"},
		{499, "client-error"},
		{500, "server-error"},
		{503, "server-error"},
		{0, "unknown"},
		{999, "unknown"},
	}
	for _, tt := range tests {
		t.Run(itoa(tt.status), func(t *testing.T) {
			if got := getStatusClass(tt.status); got != tt.want {
				t.Fatalf("getStatusClass(%d) = %q, want %q", tt.status, got, tt.want)
			}
		})
	}
}

// itoa 辅助，避免在测试名中引入额外 import
func itoa(n int) string {
	return fmt.Sprintf("%d", n)
}
```

注意：若 `getStatusClass` 的具体返回值（"success"/"redirect" 等字符串）与上方预期不符，执行时需先 `grep -n 'func getStatusClass' pkg/report/html.go` 读取实际实现并对齐 want 值。

- [ ] **Step 3: 修改 pkg/database/host_query_test.go 补 GetScreenshotsByHost/ByURL 边界**

文件: `pkg/database/host_query_test.go`（在文件末尾追加）

```go
func TestGetScreenshotsByHost_Empty(t *testing.T) {
	// 用内存 sqlite 构造空库，查询不存在的 host 应返回空切片无错误
	db := setupTestDB(t) // 复用现有测试辅助；若无则用 database.New(":memory:")
	rows, err := db.GetScreenshotsByHost("nonexistent.test")
	if err != nil {
		t.Fatalf("空库查询不应报错: %v", err)
	}
	if len(rows) != 0 {
		t.Fatalf("空库应返回 0 行, got %d", len(rows))
	}
}

func TestGetScreenshotsByURL_Empty(t *testing.T) {
	db := setupTestDB(t)
	rows, err := db.GetScreenshotsByURL("http://nonexistent.test/")
	if err != nil {
		t.Fatalf("空库查询不应报错: %v", err)
	}
	if len(rows) != 0 {
		t.Fatalf("空库应返回 0 行, got %d", len(rows))
	}
}
```

注意：`setupTestDB` 若在 host_query_test.go 或 database_test.go 中不存在，执行时需读取现有测试辅助函数名（`grep -n 'func.*[Dd]b\|func.*[Tt]est.*[Dd]ata' pkg/database/*_test.go`）并替换为实际可用的构造方式（如直接 `database.New(":memory:")`）。

- [ ] **Step 4: 修改 pkg/scan/scan_test.go 补 hasExplicitPort 与 NewPooledScanner 边界**

文件: `pkg/scan/scan_test.go`（在文件末尾追加）

```go
func TestHasExplicitPort(t *testing.T) {
	tests := []struct {
		target string
		want   bool
	}{
		{"example.com:8080", true},
		{"example.com", false},
		{"http://example.com:443/path", true},
		{"http://example.com/path", false},
		{"192.168.1.1:22", true},
		{"192.168.1.1", false},
	}
	for _, tt := range tests {
		t.Run(tt.target, func(t *testing.T) {
			if got := hasExplicitPort(tt.target); got != tt.want {
				t.Fatalf("hasExplicitPort(%q) = %v, want %v", tt.target, got, tt.want)
			}
		})
	}
}
```

注意：`hasExplicitPort` 若为未导出函数，测试与实现同包可直接调用。若签名不同，执行时先 `grep -n 'func hasExplicitPort' pkg/scan/scan.go` 对齐。

- [ ] **Step 5: 修改 pkg/techdetect/detector_test.go 补 matchFingerprint 边界 + pkg/ascii/ascii_test.go 补 Markdown**

文件: `pkg/techdetect/detector_test.go`（末尾追加）

```go
func TestDetector_NoMatches(t *testing.T) {
	d := NewDetector()
	// 空指纹集 + 任意内容应返回空结果，不 panic
	results := d.Detect(map[string]string{"X-Powered-By": "nothing"}, "")
	if len(results) != 0 {
		t.Fatalf("空指纹应返回 0 结果, got %d", len(results))
	}
}
```

文件: `pkg/ascii/ascii_test.go`（末尾追加）

```go
func TestMarkdown_NonEmpty(t *testing.T) {
	out := Markdown()
	if len(out) == 0 {
		t.Fatal("Markdown() 不应返回空串")
	}
}
```

注意：`NewDetector`、`Detect` 的确切签名与 `Markdown` 的存在性，执行时需 `grep -n 'func NewDetector\|func.*Detect\|func Markdown' pkg/techdetect/detector.go pkg/ascii/ascii.go` 对齐；若签名不符，按实际调整。

- [ ] **Step 6: 验证全包覆盖率与整体阈值**

Run: `SKIP_BROWSER_TESTS=1 go test -race -coverprofile=coverage.out ./... 2>&1 | tail -20`
Expected:
  - Exit code: 0
  - Output contains: "ok" for all packages
  - 各纯逻辑包 coverage: pkg/database >= 97%, pkg/ascii >= 95%, pkg/scan >= 90%, pkg/techdetect >= 90%, pkg/models >= 95%, pkg/report >= 88%, pkg/phash >= 95%, pkg/api >= 82%, pkg/sdk >= 80%

Run: `go tool cover -func=coverage.out | tail -1`
Expected:
  - Output contains: "total:" 且百分比 >= 90%
  - 若未达 90%，继续补测 gap 最大的函数直到达标

Run: `make coverage-check`
Expected:
  - Exit code: 0
  - Output contains: "覆盖率达标"

- [ ] **Step 7: 提交**
Run: `git add pkg/api/helpers_test.go pkg/report/html_test.go pkg/database/host_query_test.go pkg/scan/scan_test.go pkg/techdetect/detector_test.go pkg/ascii/ascii_test.go && git commit -m "test(*): cover residual gaps in api/report/database/scan/techdetect/ascii"`

---

## 受限包透明说明（不假称可达 100%）

以下函数因架构限制**无法在 CI 达 100%**，Plan 不掩盖此事实：

| 包 | 函数 | 不可达原因 |
|---|---|---|
| pkg/provider | Start / StartWithContext / handleScreenshot 截图主路径 | 需真实 Chrome 浏览器，CI 设 `SKIP_BROWSER_TESTS=1` 跳过 |
| pkg/runner | chromedp.go Witness / 真实导航截图路径 | 同上，依赖 chromedp 连接 Chrome |
| main | main() | 程序入口，无测试 |
| cmd | cobra `init()` + `RunE` | 副作用注册，难单测（纯函数已在 Task 5 覆盖） |

本 Plan 的目标 therefore 是：**纯逻辑包 100% + 整体 ≥90% + 受限包纯函数分支尽力覆盖**，而非字面全包 100%。
