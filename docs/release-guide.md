# Go-SNIR 发布指南

本文档描述了如何通过 GitHub Actions 和 GoReleaser 发布 Go-SNIR 的新版本。

## CI/CD 工作流

### CI

`.github/workflows/ci.yml` 会在以下场景运行：

- 推送到 `main`
- 向 `main` 发起 Pull Request
- 手动触发 `workflow_dispatch`

CI 会执行以下检查：

- `go mod tidy` 后确认 `go.mod` 和 `go.sum` 无未提交变更
- `gofmt`
- `go vet`
- skill bundle 文件结构校验
- 带 race detector 和 coverage 的 `go test ./...`
- Linux、macOS、Windows 常规构建
- Linux、macOS、Windows 交叉编译主程序
- Docker 镜像构建校验
- GoReleaser 配置检查和 snapshot 构建

### Release

`.github/workflows/release.yml` 只在推送 `v*` tag 时运行。发布前会先执行 preflight 检查，包括 tidy、gofmt、vet、skill bundle 校验、race 测试和常规构建。preflight 通过后，工作流会：

1. 使用 GoReleaser 创建 GitHub Release 并上传跨平台产物
2. 在发布成功后构建并推送 GHCR Docker 镜像

## 先决条件

1. 安装 GoReleaser：

```bash
# 使用Homebrew (MacOS)
brew install goreleaser

# 使用Go安装
go install github.com/goreleaser/goreleaser@latest
```

2. 确保你有适当的 GitHub 权限来推送新标签并创建 Releases。

## 发布流程

### 1. 更新版本号

首先，更新 `Makefile` 中的 `VERSION` 变量：

```makefile
VERSION := v1.0.0  # 将此更改为新版本号
```

### 2. 更新CHANGELOG

如果你维护一个更改日志，确保更新 `CHANGELOG.md` 文件，添加新版本的变更内容。

### 3. 提交所有更改

```bash
git add .
git commit -m "准备发布v1.0.0"
```

### 4. 创建新的Git标签

```bash
git tag -a v1.0.0 -m "发布v1.0.0版本"
```

### 5. 推送标签到GitHub

```bash
git push origin v1.0.0
```

这会触发 GitHub Actions release 工作流，自动使用 GoReleaser 构建和发布新版本，并在发布成功后推送 GHCR Docker 镜像。

### 6. 手动发布（可选）

如果你想手动发布而不通过GitHub Actions，可以运行：

```bash
# 测试发布过程（不会实际发布）
make release-test

# 实际发布
make release
```

## 验证发布

发布完成后，查看GitHub Releases页面，确认：

1. 所有平台的二进制文件都已上传
2. 更改日志已正确显示
3. 下载链接工作正常
4. GHCR 镜像 tag 已正确推送

## 发布后任务

1. 更新文档中的版本号引用
2. 通知用户有新版本可用
3. 在 Homebrew tap 中更新版本（如果适用）

## 故障排除

如果遇到发布问题：

1. 检查 GitHub Actions 日志中的错误
2. 确保 `.goreleaser.yml` 配置正确
3. 验证你有正确的 GitHub 权限
4. 确保 `GITHUB_TOKEN` 有足够的权限

如需更多帮助，请参考 [GoReleaser 文档](https://goreleaser.com/)。
