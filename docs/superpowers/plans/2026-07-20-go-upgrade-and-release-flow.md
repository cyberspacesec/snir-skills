# Go 工具链升级 + 发布流程验证 + Skills 安装文档补全 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: `superpowers:subagent-driven-development`
> Steps use checkbox (`- [ ]`) syntax.

**Goal:** 将本机 Go 从 1.25.0 升级到最新 1.26.5、项目 go.mod 同步到 go 1.26.0，本地验证 goreleaser 发布链路在升级后仍可用，并触发一次真实新版本（v0.1.1）发布跑通 GitHub Actions 全链路，同时把 SKILL.md / README / installation.md 的源码编译教程补完整并同步 Go 版本要求。

**Architecture:**
- **数据怎么流：** 本机下载 go1.26.5 官方 tarball → 备份并替换 `~/go`（GOROOT）→ 恢复 goreleaser 等工具二进制 → 改 go.mod 的 `go` 指令为 1.26.0、删 toolchain 行 → `go mod tidy` + `go build/test` 验证 → `goreleaser check` + `--snapshot` 本地验证发布配置 → 改文档同步 Go 版本 + 补源码教程 → 提交 → 打 v0.1.1 tag 推送 → release.yml 自动触发 preflight→goreleaser→docker 全链路 → GitHub Release 产出多平台 assets + Docker 镜像推送到 ghcr.io。
- **关键组件：** `~/go`（GOROOT，替换内容）、`go.mod`（go 指令+toolchain）、`SKILL.md`、`README.md`、`website/guide/installation.md`（文档）、`.goreleaser.yml`+`.github/workflows/release.yml`（已存在，仅验证不改动）。
- **为什么这样做：** 项目 release.yml/ci.yml/.goreleaser.yml 已完整且 v0.1.0 已成功发布过，发布基础设施无需重建，只需升级 Go 后验证不破 + 用新 tag 触发一次全链路即可；SKILL.md 已有 release 下载段和 `make build` 一行，只需把源码段补成完整教程并同步 Go 版本号。

**Tech Stack:** Go 1.26.5（最新稳定版，go.dev 2026-07 确认）、GoReleaser v2、GitHub Actions（actions/checkout@v4、actions/setup-go@v5、goreleaser/goreleaser-action@v6）、cobra v1.9.1、gorilla/mux v1.8.1、mark3labs/mcp-go（本次不动）

**Risks:**
- Task 1 替换 `~/go`（GOROOT）可能丢失 `~/go/bin/` 下的 goreleaser、golangci-lint、staticcheck、govulncheck、actionlint 等工具二进制 → 缓解：备份 `~/go` 到 `~/go-1.25.bak`，新 Go 解压后把备份的 `bin/` 工具二进制（除 go/gofmt）拷回新 `~/go/bin/`
- Task 2 升级 go.mod 到 `go 1.26.0` 后 `go mod tidy` 可能改动 go.sum（间接依赖若声明更高 go 版本）→ 缓解：tidy 前后对比 `git diff go.sum`，仅接受与版本声明相关的改动，`go build ./...` + `go test ./...` 全绿才提交
- Task 4 push tag v0.1.1 是 outward-facing 不可逆操作——会真实发布 GitHub Release（多平台 binary）并推送 Docker 镜像到 ghcr.io → 缓解：用 patch 版本 v0.1.1（非大版本）降低影响面；用户明确要求"跑通发布新版本的流程"构成 durable authorization；Task 3 已用 `--snapshot --skip=publish` 本地预演确保配置无误后才打 tag
- GoReleaser 用 `version: latest`（release.yml:112）可能拉到与本地不一致的新版 → 缓解：Task 3 本地 goreleaser 先 `check` + `--snapshot`，若本地与 CI 版本行为不一致以 CI 为准（CI 是发布真相源）

---

### Task 1: 升级本机 Go 至 1.26.5

**Depends on:** None
**Files:**
- Modify: `~/go`（GOROOT，替换二进制与标准库）

- [ ] **Step 1: 备份现有 ~/go 到 ~/go-1.25.bak — 保留旧 Go 与工具二进制以便回滚**

`~/go` 当前是 GOROOT（`go env GOROOT` = `/home/cc11001100/go`），同时 `~/go/bin/` 下有 goreleaser、golangci-lint、staticcheck、govulncheck、actionlint 等 `go install` 装的工具二进制。直接解压新 Go 会覆盖这些。先整体备份。

Run: `cp -a ~/go ~/go-1.25.bak && ls ~/go-1.25.bak/bin/ && du -sh ~/go-1.25.bak`
Expected:
  - Exit code: 0
  - Output contains: "go" and "goreleaser" and "gofmt"
  - 备份目录约 460M

- [ ] **Step 2: 下载 go1.26.5 官方 tarball — 从 go.dev 拉取最新稳定版**

go.dev 确认最新稳定版是 go1.26.5（2026-07）。下载 linux-amd64 tarball 到 /tmp。

Run: `curl -fL -o /tmp/go1.26.5.tar.gz https://go.dev/dl/go1.26.5.linux-amd64.tar.gz && ls -lh /tmp/go1.26.5.tar.gz`
Expected:
  - Exit code: 0
  - Output contains: "go1.26.5.tar.gz"
  - 文件大小约 70-80MB

- [ ] **Step 3: 校验 tarball 完整性 — 防止下载损坏导致升级后 Go 异常**

从 go.dev 的 mode=json 接口取官方 sha256，与下载文件对比。

Run: `EXPECTED=$(curl -s https://go.dev/dl/?mode=json | grep -A4 '"go1.26.5.linux-amd64.tar.gz"' | grep sha256 | sed -E 's/.*"sha256"[[:space:]]*:[[:space:]]*"([^"]+)".*/\1/') && ACTUAL=$(sha256sum /tmp/go1.26.5.tar.gz | awk '{print $1}') && echo "expected=$EXPECTED" && echo "actual=$ACTUAL" && [ "$EXPECTED" = "$ACTUAL" ] && echo "SHA256 OK" || (echo "SHA256 MISMATCH" && exit 1)`
Expected:
  - Exit code: 0
  - Output contains: "SHA256 OK"

- [ ] **Step 4: 清空 ~/go 内容并解压新 Go — 替换 GOROOT**

先删 `~/go` 下所有内容（已备份），再解压新 Go 到 `~/`（官方 tarball 顶层是 `go/` 目录，解压到 `~/` 即生成 `~/go`）。

Run: `rm -rf ~/go && tar -C ~ -xzf /tmp/go1.26.5.tar.gz && ls ~/go/bin/`
Expected:
  - Exit code: 0
  - Output contains: "go" and "gofmt"（新 GOROOT 只含这两个 + go1.26.5 toolchain 链接）

- [ ] **Step 5: 恢复 go install 装的工具二进制 — 让 goreleaser 等工具在新 GOROOT 下可用**

从备份 `~/go-1.25.bak/bin/` 把非 go/gofmt 的工具二进制拷回新 `~/go/bin/`。这些是独立 ELF，与 GOROOT 版本无关，可直接复用。

Run: `cd ~/go-1.25.bak/bin && for t in goreleaser golangci-lint staticcheck govulncheck actionlint goimports; do [ -f "$t" ] && cp "$t" ~/go/bin/ && echo "restored $t"; done`
Expected:
  - Exit code: 0
  - Output contains: "restored goreleaser"

- [ ] **Step 6: 验证新 Go 版本与工具链 — 确认升级生效**

Run: `hash -r 2>/dev/null; export PATH=~/go/bin:$PATH; go version && go env GOROOT GOTOOLCHAIN && goreleaser --version 2>&1 | head -3`
Expected:
  - Exit code: 0
  - `go version` 输出: "go version go1.26.5 linux/amd64"
  - `GOROOT` = "/home/cc11001100/go"
  - `goreleaser --version` 正常输出版本号

- [ ] **Step 7: 验证现有项目在新 Go 下仍可构建 — 确认升级不破坏现状**

在升级 go.mod 之前（Task 2 才改 go.mod），先用新 Go 验证现有代码能编译。

Run: `cd /home/cc11001100/github/cyberspacesec/snir-skills && export PATH=~/go/bin:$PATH && go build ./... 2>&1 | tail -5`
Expected:
  - Exit code: 0
  - 无 error 输出

- [ ] **Step 8: 提交（本 Task 无代码变更，无需 git commit）**

Task 1 仅改本机环境，不产生仓库变更。跳过提交。

---

### Task 2: 升级项目 go.mod 至 go 1.26.0

**Depends on:** Task 1（需新 Go 1.26.5 工具链）
**Files:**
- Modify: `go.mod:3-5`（`go 1.23.0` → `go 1.26.0`，删除 `toolchain go1.23.2` 行）
- Modify: `go.sum`（tidy 后可能调整）

- [ ] **Step 1: 修改 go.mod 的 go 指令并删除 toolchain 行 — 升级项目 Go 版本要求**

文件: `go.mod:3-5`（替换前 5 行头部）

```text
module github.com/cyberspacesec/snir-skills

go 1.26.0

require (
```

说明：删除 `toolchain go1.23.2` 行——升到 go 1.26.0 后，本机已是 1.26.5，不再需要 toolchain 兜底机制下载旧版。GitHub Actions 的 `actions/setup-go@v5` 配合 `go-version-file: go.mod` 会自动用 ≥1.26.0 的最新 Go。

- [ ] **Step 2: 运行 go mod tidy 同步依赖 — 让 go.sum 与新 go 指令一致**

Run: `cd /home/cc11001100/github/cyberspacesec/snir-skills && export PATH=~/go/bin:$PATH && go mod tidy && git diff --stat go.mod go.sum`
Expected:
  - Exit code: 0
  - `git diff --stat` 显示 go.mod 改动（go 指令行）+ 可能的 go.sum 调整
  - 无 "requires go" 报错

- [ ] **Step 3: 验证构建与 vet 全绿 — 确认升级后代码可编译、无静态错误**

Run: `export PATH=~/go/bin:$PATH && go build ./... && go vet ./... && echo "BUILD+VET OK"`
Expected:
  - Exit code: 0
  - Output contains: "BUILD+VET OK"
  - 无 error 输出

- [ ] **Step 4: 验证测试全绿 — 确认升级不破坏现有测试**

Run: `export PATH=~/go/bin:$PATH && go test ./... 2>&1 | tail -15`
Expected:
  - Exit code: 0
  - Output contains: "ok" for all packages
  - 无 "FAIL"

- [ ] **Step 5: 检查 go.mod 与 go.sum 是否 tidy — CI 的 tidy 验证步骤能通过**

release.yml 与 ci.yml 都有 `go mod tidy && git diff --exit-code -- go.mod go.sum` 步骤，本地必须先保证 tidy。

Run: `export PATH=~/go/bin:$PATH && go mod tidy && git diff --exit-code -- go.mod go.sum && echo "TIDY OK"`
Expected:
  - Exit code: 0
  - Output contains: "TIDY OK"

- [ ] **Step 6: 提交**
Run: `cd /home/cc11001100/github/cyberspacesec/snir-skills && git add go.mod go.sum && git commit -m "chore(go): upgrade go directive to 1.26.0, drop toolchain pin"`

---

### Task 3: 本地验证 goreleaser 发布配置在升级后不破

**Depends on:** Task 2（go.mod 已升级）
**Files:**
- 无修改（仅验证 `.goreleaser.yml` 与 Makefile）

- [ ] **Step 1: 运行 goreleaser check 验证配置语法 — 确认 .goreleaser.yml 在新 Go 下仍有效**

Run: `cd /home/cc11001100/github/cyberspacesec/snir-skills && export PATH=~/go/bin:$PATH && goreleaser check 2>&1 | tail -10`
Expected:
  - Exit code: 0
  - Output contains: "ok" or "configuration is valid" or "configuration satisfied"
  - 无 "error" or "fail"

- [ ] **Step 2: 运行 goreleaser snapshot 本地预演发布 — 确认跨平台构建在升级后仍工作**

用 `--snapshot --clean --skip=publish` 本地构建所有平台二进制但不发布，与 CI 的 goreleaser-snapshot job（ci.yml:222-230）行为一致。需设 BUILD_TIME 环境变量（.goreleaser.yml:116 ldflags 引用）。

Run: `export PATH=~/go/bin:$PATH && export BUILD_TIME=$(date +%H:%M:%S) && goreleaser release --snapshot --clean --skip=publish 2>&1 | tail -20`
Expected:
  - Exit code: 0
  - Output contains: "success" or "done" or "build succeeded"
  - `dist/` 目录生成多个平台 snir 二进制
  - 无 "error" or "fatal"

- [ ] **Step 3: 验证 snapshot 产物存在 — 确认跨平台构建实际产出二进制**

Run: `ls -la dist/ 2>&1 | head -20 && echo "---" && ls dist/*.tar.gz dist/*.zip 2>/dev/null | head -5 && echo "---" && file dist/snir_linux_amd64*/snir 2>/dev/null || ls dist/linux_amd64/ 2>/dev/null`
Expected:
  - Exit code: 0
  - `dist/` 含多平台构建目录或归档文件
  - 至少有 linux_amd64 产物

- [ ] **Step 4: 清理 snapshot 产物 — 不把 dist/ 提交到仓库**

`.gitignore` 已含 `/dist/`（.gitignore:78），但本地清理避免误提交。

Run: `rm -rf dist/ && git status --short`
Expected:
  - Exit code: 0
  - `git status` 无 dist/ 相关改动

- [ ] **Step 5: 提交（本 Task 仅验证，无代码变更，无需 git commit）**

Task 3 验证性质，跳过提交。

---

### Task 4: 补全源码编译教程 + 同步 Go 版本要求

**Depends on:** Task 2（go.mod 已升 1.26，文档需同步版本号）
**Files:**
- Modify: `SKILL.md:32-37`（源码段扩展为完整教程）
- Modify: `README.md:392-396`（Requires Go 1.23+ → 1.26+，补完整源码教程）
- Modify: `website/guide/installation.md:87-96`（需 Go 1.23+ → 1.26+，方式三源码构建段补完整）

- [ ] **Step 1: 扩展 SKILL.md 源码段为完整编译教程 — 补齐从源码编译的完整步骤**

文件: `SKILL.md:32-37`（替换 "If working from this repository:" 区块）

当前只有 `make build` + `./snir version` 两行，扩展为含 Go 安装、clone、构建、全局安装的完整教程，并把二进制 release 安装段也补一句"也可 go install"。

```markdown
If working from this repository (build from source):

```bash
# 1. Install Go 1.26+ from https://go.dev/dl/
# 2. Clone and build
git clone https://github.com/cyberspacesec/snir-skills.git
cd snir-skills
make build                     # builds ./snir with version/commit metadata
./snir version

# 3. (Optional) Install to a directory on PATH
make install                   # go install into $GOPATH/bin
# or manually:
sudo install -m 0755 snir /usr/local/bin/snir

# 4. (Optional) Build for a different platform (no Chrome needed on build host)
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o snir-arm64 .
```

`make build` 注入版本信息 via ldflags（见 `Makefile`）；`make release-test` runs `goreleaser --snapshot` to validate the full release pipeline locally without publishing.
```

- [ ] **Step 2: 更新 README From Source 段 — 同步 Go 版本要求并补完整教程**

文件: `README.md:392-396`（替换 "### From Source" 区块）

当前是：

```markdown
### From Source

Requires Go 1.23+.

```bash
git clone https://github.com/cyberspacesec/snir-skills.git
cd snir-skills
make build
./snir version
```
```

替换为：

```markdown
### From Source

Requires Go 1.26+ (download from https://go.dev/dl/).

```bash
git clone https://github.com/cyberspacesec/snir-skills.git
cd snir-skills
make build                     # build ./snir with version/commit ldflags
./snir version

# optional: install to PATH
make install                   # go install into $GOPATH/bin

# optional: cross-compile (CGO-free, no Chrome on build host needed)
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o snir-arm64 .
```

See [installation guide](https://cyberspacesec.github.io/snir-skills/guide/installation) for platform-specific notes.
```

- [ ] **Step 3: 更新 website/guide/installation.md 方式三 — 同步 Go 版本并补完整源码教程**

文件: `website/guide/installation.md:87-96`（替换 "## 方式三：源码构建" 区块）

当前是：

```markdown
## 方式三：源码构建

需 Go 1.23+：

```bash
git clone https://github.com/cyberspacesec/snir-skills.git
cd snir-skills
make build
./snir version
```
```

替换为：

```markdown
## 方式三：源码构建

需 Go 1.26+（从 [go.dev/dl](https://go.dev/dl/) 下载安装）：

```bash
git clone https://github.com/cyberspacesec/snir-skills.git
cd snir-skills
make build                     # 构建 ./snir，注入版本/提交 ldflags
./snir version

# 可选：安装到 PATH
make install                   # go install 到 $GOPATH/bin

# 可选：交叉编译（纯 Go，构建机无需 Chrome）
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o snir-arm64 .
```

::: details make 目标说明
| 目标 | 作用 |
|------|------|
| `make build` | 构建本地平台二进制，注入 version/commit/date ldflags |
| `make install` | `go install` 到 `$GOPATH/bin` |
| `make test` | 运行全部测试 |
| `make release-test` | 本地预演 goreleaser 发布（`--snapshot --skip=publish`） |
:::

::: warning Go 版本
`go.mod` 声明 `go 1.26.0`，低于 1.26 的工具链会触发 toolchain 自动下载。建议直接安装 1.26.5+ 避免额外下载。
:::
```

- [ ] **Step 4: 验证文档站构建 — 确认 installation.md 改动不破坏 VitePress 构建**

Run: `cd /home/cc11001100/github/cyberspacesec/snir-skills/website && npx vitepress build 2>&1 | tail -5 && cd ..`
Expected:
  - Exit code: 0
  - Output does NOT contain: "error" or "doesn't exist"
  - Output contains: "✓ built" 或 "building"

- [ ] **Step 5: 验证 SKILL.md 仍通过 CI 的 skill bundle 校验 — release.yml:56-63 有结构校验**

Run: `cd /home/cc11001100/github/cyberspacesec/snir-skills && test -f SKILL.md && grep -q '^name: ' SKILL.md && grep -q '^description: ' SKILL.md && test -x scripts/install-snir.sh && sh -n scripts/install-snir.sh && jq empty evals/evals.json && echo "SKILL BUNDLE OK"`
Expected:
  - Exit code: 0
  - Output contains: "SKILL BUNDLE OK"

- [ ] **Step 6: 提交文档变更**
Run: `cd /home/cc11001100/github/cyberspacesec/snir-skills && git add SKILL.md README.md website/guide/installation.md && git commit -m "docs: expand source-build tutorial + sync Go 1.26 requirement across SKILL/README/installation"`

---

### Task 5: 触发新版本发布并验证全链路

**Depends on:** Task 2（go.mod 已升）、Task 3（goreleaser 已验证）、Task 4（文档已提交）
**Files:** 无修改（通过 git tag 触发已存在的 release.yml）

- [ ] **Step 1: 推送所有提交到 origin/main — 让 GitHub Actions 在最新代码上触发**

Run: `cd /home/cc11001100/github/cyberspacesec/snir-skills && git push origin main`
Expected:
  - Exit code: 0
  - 输出显示 push 成功，无 rejected

- [ ] **Step 2: 创建并推送 v0.1.1 tag — 触发 release.yml 全链路发布**

用 patch 版本 v0.1.1（v0.1.0 之后的增量发布），降低影响面。tag message 说明这是 Go 升级后的首次发布验证。

Run: `cd /home/cc11001100/github/cyberspacesec/snir-skills && git tag -a v0.1.1 -m "Release v0.1.1: Go 1.26 toolchain upgrade, first release on new toolchain" && git push origin v0.1.1`
Expected:
  - Exit code: 0
  - `git push origin v0.1.1` 成功，输出含 "new tag" 或 "[new tag]"

- [ ] **Step 3: 监控 release.yml 工作流执行 — 确认 preflight→goreleaser→docker 全链路成功**

tag 推送后 release.yml 自动触发。用 gh 监控。

Run: `cd /home/cc11001100/github/cyberspacesec/snir-skills && sleep 10 && gh run list --workflow=release.yml --limit 1 && echo "---" && gh run watch $(gh run list --workflow=release.yml --limit 1 --json databaseId --jq '.[0].databaseId') 2>&1 | tail -20`
Expected:
  - Exit code: 0
  - 工作流触发，三个 job（preflight、goreleaser、docker）依次执行
  - 最终状态含 "success" 或 "completed" (success)
  - 若失败，根据失败 job 名定位（preflight 失败=代码/测试问题，goreleaser 失败=发布配置，docker 失败=镜像推送）

- [ ] **Step 4: 验证新版本 Release 与 assets 已生成 — 确认发布产物完整**

Run: `cd /home/cc11001100/github/cyberspacesec/snir-skills && gh release view v0.1.1 --json tagName,assets --jq '{tag: .tagName, asset_count: (.assets|length), assets: [.assets[].name]}' 2>&1 | head -30`
Expected:
  - Exit code: 0
  - tagName: "v0.1.1"
  - asset_count ≥ 20（多平台 tar.gz/zip + checksums.txt）
  - assets 含 "snir-skills_Linux_x86_64.tar.gz"、"snir-skills_Darwin_arm64.tar.gz"、"checksums.txt"

- [ ] **Step 5: 验证 install-snir.sh 能从新 release 安装 — 端到端验证发布链路可用**

用自带脚本安装 v0.1.1 到临时前缀（不覆盖现有 snir），验证 release 产物可被脚本正确下载安装。

Run: `cd /tmp && SNIR_VERSION=v0.1.1 SNIR_PREFIX=/tmp/snir-install-test /home/cc11001100/github/cyberspacesec/snir-skills/scripts/install-snir.sh 2>&1 | tail -5 && /tmp/snir-install-test/snir version 2>&1 | head -3 && rm -rf /tmp/snir-install-test`
Expected:
  - Exit code: 0
  - 脚本输出 "installed snir to /tmp/snir-install-test"
  - `/tmp/snir-install-test/snir version` 输出 snir logo + vX.Y.Z
  - 无 error

- [ ] **Step 6: 提交（本 Task 的发布动作已通过 git tag 完成，无需额外 commit）**

Task 5 的发布通过 Step 2 的 tag + Step 3-5 的验证完成，无额外代码提交。

---

## 发布流程现状说明（非 Task，供执行者参考）

项目的发布基础设施**已完整存在且 v0.1.0 已成功发布**（20+ 多平台 assets，见 `gh release view v0.1.0`）。本 Plan 不重建发布流程，只做：

1. 升级 Go 工具链（Task 1-2）
2. 本地验证发布配置在新 Go 下不破（Task 3）
3. 用新 tag v0.1.1 触发一次完整 release.yml 全链路（Task 5），验证升级后发布仍端到端可用

发布链路结构（`release.yml`）：
- **触发**：push tag `v*`
- **preflight job**：checkout → setup-go(go-version-file) → tidy 验证 → gofmt → go vet → skill bundle 校验 → Chrome 配置 → go test -race → go build
- **goreleaser job**（needs preflight）：checkout → setup-go → goreleaser release --clean → 产出多平台 binary + deb/rpm/archlinux 包 + checksums + GitHub Release
- **docker job**（needs goreleaser）：checkout → buildx → login ghcr.io → build-push linux/amd64+arm64 镜像

触发新版本发布的标准命令序列（Task 5 Step 1-2 已编码）：
```bash
git push origin main                              # 先推代码
git tag -a v0.1.1 -m "Release v0.1.1: ..."        # 打 tag
git push origin v0.1.1                            # 推 tag 触发 release.yml
```
