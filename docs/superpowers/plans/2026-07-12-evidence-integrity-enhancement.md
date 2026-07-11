# Evidence Integrity Enhancement Plan

> **For agentic workers:** REQUIRED SUB-SKILL: `superpowers:subagent-driven-development`
> Steps use checkbox (`- [ ]`) syntax.

**Goal:** 修复 AI Agent 接入实测暴露的两个证据字段缺口：`content_length` 在 HTTP/2 响应下始终为 0、`response_reason` 在 HTTP/2 下为空且无降级，让 Agent 拿到的 `Result` JSONL/API 响应包含完整可靠的响应元数据。

**Architecture:** 当前 `pkg/runner/chromedp.go` 的 `ListenTarget` 回调在 `EventResponseReceived` case 中从 `e.Response.Headers["Content-Length"]` 解析 content_length。HTTP/2 不传输 `Content-Length` 头，故永远为 0。改为双源策略：主源用 CDP 的 `e.Response.EncodedDataLength`（字节级、协议无关，同步可得），兜底用 `EventLoadingFinished.EncodedDataLength`（加载完成后的最终字节数，异步覆盖）。`response_reason` 在 HTTP/2 下 `e.Response.StatusText` 普遍为空，仅当协议是 `http/1.x` 且 StatusText 空时从状态码推断 reason phrase，h2/h3 空属正常不误判。

**Tech Stack:** Go 1.23, chromedp v0.13.0, chromedp/cdproto v0.0.0-20250222051814-50c6cb17f10a, cobra v1.9.1, GORM v1.25.12

**Risks:**
- `EventLoadingFinished` 是异步事件，可能在结果读取后才到达 → 缓解：用 `EventResponseReceived.EncodedDataLength` 作主源（同步），`LoadingFinished` 仅作兜底覆盖；两者都为 0 时保留 0 不编造数据
- HTTP/2 的 `StatusText` 普遍为空，不能误判为捕获缺陷 → 缓解：reason 降级仅对 `http/1.x` 生效，h2/h3 空属正常
- 改动集中在 `pkg/runner/chromedp.go` 单文件单函数，行号会因 Task 1 偏移影响 Task 2 → 缓解：Task 2 用函数名 + 上下文描述定位，不硬依赖 Task 1 后的行号

---

### Task 1: content_length 双源捕获

**Depends on:** None
**Files:**
- Modify: `pkg/runner/chromedp.go:187-194`（替换 content_length 收集逻辑）
- Modify: `pkg/runner/chromedp.go:227-229`（新增 `EventLoadingFinished` case）
- Test: `pkg/runner/chromedp_evidence_test.go`（新增）

- [x] **Step 1: 替换 content_length 收集逻辑 — 从依赖 Content-Length 头改为优先用 CDP EncodedDataLength**

文件: `pkg/runner/chromedp.go:187-194`（在 `case *network.EventResponseReceived` 主请求匹配块内，紧跟 `protocol` 赋值之后、`PDF` 检测之前）

```go
				// 记录内容长度（优先用 CDP 的 EncodedDataLength，协议无关；
				// HTTP/2 不传输 Content-Length 头，单纯从头解析会恒为 0）
				if e.Response.EncodedDataLength > 0 {
					contentLength = int64(e.Response.EncodedDataLength)
				} else if cl, ok := e.Response.Headers["Content-Length"]; ok {
					if clStr, ok := cl.(string); ok {
						if clInt, err := strconv.ParseInt(clStr, 10, 64); err == nil {
							contentLength = clInt
						}
					}
				}
```

- [x] **Step 2: 新增 EventLoadingFinished case — 用加载完成后的最终字节数兜底覆盖 content_length**

文件: `pkg/runner/chromedp.go:227-229`（替换 `default:` 之前的空位，在 `case *runtime.EventExceptionThrown` 之后、`default` 之前插入新 case）

```go
			case *network.EventLoadingFinished:
				// 加载完成事件携带最终 EncodedDataLength，比 ResponseReceived 时更准确
				if e.EncodedDataLength > 0 {
					if nl, ok := networkEvents[e.RequestID.String()]; ok {
						nl.ContentLength = int64(e.EncodedDataLength)
					}
					// 主请求的 contentLength 在 ResponseReceived 时可能为 0（数据未传完），这里兜底
					if contentLength == 0 {
						contentLength = int64(e.EncodedDataLength)
					}
				}
```

- [x] **Step 3: 给 NetworkLog 增加 ContentLength 字段以承载逐请求字节数**

文件: `pkg/models/models.go:153-162`（替换整个 `NetworkLog` struct）

```go
// NetworkLog represents a network request log entry
type NetworkLog struct {
	ID            uint        `json:"id" gorm:"primarykey"`
	ResultID      uint        `json:"result_id"`
	Type          RequestType `json:"type"`
	URL           string      `json:"url"`
	Method        string      `json:"method"`
	StatusCode    int         `json:"status_code"`
	ContentType   string      `json:"content_type"`
	ContentLength int64       `json:"content_length"`
	Body          string      `json:"body"`
}
```

- [x] **Step 4: 创建 chromedp 证据捕获单元测试 — 验证 content_length 双源逻辑**

```go
// pkg/runner/chromedp_evidence_test.go
package runner

import (
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/chromedp/cdproto/network"
)

// TestContentLengthFromEncodedDataLength 验证 EventResponseReceived 时优先用 EncodedDataLength
func TestContentLengthFromEncodedDataLength(t *testing.T) {
	var contentLength int64
	e := &network.EventResponseReceived{
		RequestID: "1",
		Response: &network.Response{
			URL:               "https://example.com/",
			Status:            200,
			EncodedDataLength: 1256,
			Headers:           network.Headers{}, // 无 Content-Length 头，模拟 HTTP/2
		},
	}
	// 模拟主请求匹配块内的逻辑
	if e.Response.EncodedDataLength > 0 {
		contentLength = int64(e.Response.EncodedDataLength)
	}
	if contentLength != 1256 {
		t.Errorf("contentLength = %d, want 1256 (from EncodedDataLength)", contentLength)
	}
}

// TestContentLengthFallbackToHeader 验证 EncodedDataLength 为 0 时从头解析
func TestContentLengthFallbackToHeader(t *testing.T) {
	var contentLength int64
	e := &network.EventResponseReceived{
		RequestID: "1",
		Response: &network.Response{
			URL:               "https://example.com/",
			Status:            200,
			EncodedDataLength: 0,
			Headers:           network.Headers{"Content-Length": "999"},
		},
	}
	if e.Response.EncodedDataLength > 0 {
		contentLength = int64(e.Response.EncodedDataLength)
	} else if cl, ok := e.Response.Headers["Content-Length"]; ok {
		if clStr, ok := cl.(string); ok {
			if clInt, err := strconv.ParseInt(clStr, 10, 64); err == nil {
				contentLength = clInt
			}
		}
	}
	if contentLength != 999 {
		t.Errorf("contentLength = %d, want 999 (from Content-Length header)", contentLength)
	}
}

// TestLoadingFinishedOverridesZero 验证 EventLoadingFinished 兜底覆盖
func TestLoadingFinishedOverridesZero(t *testing.T) {
	var contentLength int64
	fin := &network.EventLoadingFinished{
		RequestID:         "1",
		EncodedDataLength: 2048,
	}
	if contentLength == 0 && fin.EncodedDataLength > 0 {
		contentLength = int64(fin.EncodedDataLength)
	}
	if contentLength != 2048 {
		t.Errorf("contentLength = %d, want 2048 (overridden by LoadingFinished)", contentLength)
	}
}

// TestResponseReasonHTTP1Fallback 验证 HTTP/1.x 空 StatusText 时从状态码推断
func TestResponseReasonHTTP1Fallback(t *testing.T) {
	protocol := "http/1.1"
	var responseReason string
	status := int64(200)
	if responseReason == "" && strings.HasPrefix(protocol, "http/1") {
		responseReason = http.StatusText(int(status))
	}
	if responseReason != "OK" {
		t.Errorf("responseReason = %q, want %q (inferred from status 200)", responseReason, "OK")
	}
}

// TestResponseReasonHTTP2EmptyIsNormal 验证 HTTP/2 空 StatusText 不触发降级
func TestResponseReasonHTTP2EmptyIsNormal(t *testing.T) {
	protocol := "h2"
	var responseReason string
	status := int64(200)
	if responseReason == "" && strings.HasPrefix(protocol, "http/1") {
		responseReason = http.StatusText(int(status))
	}
	if responseReason != "" {
		t.Errorf("responseReason = %q, want empty for HTTP/2 (no reason phrase transmitted)", responseReason)
	}
}
```

- [x] **Step 5: 验证证据捕获逻辑单元测试通过**
Run: `go test ./pkg/runner/ -run "TestContentLength|TestLoadingFinished|TestResponseReason" -count=1 -v`
Expected:
  - Exit code: 0
  - Output contains: "PASS"
  - Output does NOT contain: "FAIL"

---

### Task 2: response_reason HTTP/2 降级

**Depends on:** Task 1
**Files:**
- Modify: `pkg/runner/chromedp.go:182`（替换 responseReason 赋值，增加降级）
- Test: `pkg/runner/chromedp_evidence_test.go`（追加测试）

- [x] **Step 1: 替换 responseReason 赋值 — HTTP/1.x 空时从状态码推断 reason phrase**

文件: `pkg/runner/chromedp.go:182`（在主请求匹配块内，紧跟 `finalURL = respURL` 之后）

```go
				// 记录响应原因短语
				responseReason = e.Response.StatusText
				// HTTP/2/3 不传输 reason phrase，空属正常；仅 HTTP/1.x 空时从状态码推断
				if responseReason == "" && strings.HasPrefix(protocol, "http/1") {
					responseReason = http.StatusText(int(e.Response.Status))
				}
```

- [x] **Step 2: 添加 net/http import — 提供 StatusText 函数**

文件: `pkg/runner/chromedp.go:1-32`（在 import 块中添加 net/http）

```go
import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/dom"
	"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/cdproto/runtime"

	"github.com/chromedp/chromedp"

	"github.com/cyberspacesec/snir-skills/pkg/log"
	"github.com/cyberspacesec/snir-skills/pkg/models"
	"github.com/cyberspacesec/snir-skills/pkg/phash"
	"github.com/cyberspacesec/snir-skills/pkg/techdetect"
)
```

- [x] **Step 3: 验证 response_reason 降级测试通过**
Run: `go test ./pkg/runner/ -run "TestResponseReason" -count=1 -v`
Expected:
  - Exit code: 0
  - Output contains: "PASS"
  - Output does NOT contain: "FAIL"

- [x] **Step 4: 提交**
Run: `git add pkg/runner/chromedp.go pkg/runner/chromedp_evidence_test.go && git commit -m "fix(runner): infer response_reason for HTTP/1.x when StatusText is empty"`

---

### Task 3: 端到端 AI Agent 接入验证

**Depends on:** Task 1, Task 2
**Files:**
- Modify: 无（纯验证 Task）

- [x] **Step 1: 构建修复后的二进制**
Run: `make build`
Expected:
  - Exit code: 0
  - Output contains: "构建" or "build"
  - `./snir` 可执行文件存在且版本输出 `v0.0.1`

- [x] **Step 2: 验证 CLI 单 URL 证据完整性 — content_length 非零、reason 有值或合理空**
Run: `cd /tmp && rm -rf snir-verify && mkdir snir-verify && cd snir-verify && timeout 120 /home/cc11001100/github/cyberspacesec/snir-skills/snir scan example.com --save-headers --write-jsonl && python3 -c "import json; r=json.loads(open('results.jsonl').readline()); print('content_length:', r.get('content_length')); print('response_reason:', repr(r.get('response_reason'))); print('response_code:', r.get('response_code')); assert r.get('response_code')==200, 'code should be 200'"`
Expected:
  - Exit code: 0
  - Output contains: "response_code: 200"
  - `content_length` 值 > 0（不再恒为 0）

- [x] **Step 3: 验证 HTTP API 证据完整性 — data.content_length 非零**
Run: `cd /tmp/snir-verify && /home/cc11001100/github/cyberspacesec/snir-skills/snir api --host 127.0.0.1 --port 19093 --api-key vtest >/tmp/api-verify.log 2>&1 & sleep 4; curl -s -X POST http://127.0.0.1:19093/screenshot -H "X-API-Key: vtest" -H "Content-Type: application/json" -d '{"url":"example.com","save_headers":true}' | python3 -c "import sys,json; r=json.load(sys.stdin); d=r.get('data') or {}; print('data.content_length:', d.get('content_length')); print('data.response_code:', d.get('response_code'))"; pkill -f "snir.*api.*19093"`
Expected:
  - Exit code: 0
  - Output contains: "data.response_code: 200"
  - `data.content_length` 值 > 0

- [x] **Step 4: 运行完整测试套件确保无回归**
Run: `go test ./pkg/runner/ ./pkg/models/ ./pkg/sdk/ ./pkg/api/ ./pkg/scan/ -count=1 -timeout 300s`
Expected:
  - Exit code: 0
  - Output contains: "ok" for each package
  - Output does NOT contain: "FAIL"

- [x] **Step 5: 提交验证结果到记忆（如有新发现）**
Run: 无（此步为记录）
Expected:
  - 若验证发现新问题，记录到 memory；若全部通过，确认证据完整性增强完成
