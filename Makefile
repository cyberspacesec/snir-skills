.PHONY: build clean install test release release-test

# 默认目标
all: build

# 版本信息
VERSION := v0.0.1
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE := $(shell date +%Y-%m-%d)
BUILD_TIME := $(shell date +%H:%M:%S)
LDFLAGS := -ldflags "-X github.com/cyberspacesec/snir-skills/pkg/ascii.version=$(VERSION) -X github.com/cyberspacesec/snir-skills/pkg/ascii.commit=$(COMMIT) -X github.com/cyberspacesec/snir-skills/pkg/ascii.buildDate=$(BUILD_DATE) -X github.com/cyberspacesec/snir-skills/pkg/ascii.buildTime=$(BUILD_TIME)"

# 构建可执行文件
build:
	@echo "正在构建 snir..."
	@go build $(LDFLAGS) -o snir

# 安装到系统
install:
	@echo "正在安装 snir..."
	@go install $(LDFLAGS)

# 清理构建结果
clean:
	@echo "正在清理..."
	@rm -f snir
	@rm -f go-snir
	@rm -rf dist/

# 运行测试
test:
	@echo "正在运行测试..."
	@go test ./...

# 运行测试并生成覆盖率报告
coverage:
	@echo "正在生成测试覆盖率报告..."
	@go test -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "覆盖率报告已生成到 coverage.html"

# GoReleaser - 测试发布过程（不实际发布）
release-test:
	@echo "正在测试 GoReleaser 配置..."
	@goreleaser release --snapshot --clean --skip=publish

# GoReleaser - 创建新的发布版本
release:
	@echo "请确保已经创建了新的 git tag，例如: git tag -a v1.0.0 -m \"发布v1.0.0版本\""
	@echo "然后执行: git push origin v1.0.0"
	@echo "如果已完成上述步骤，按回车继续..."
	@read dummy
	@echo "创建新的发布版本..."
	@goreleaser release --clean

# 帮助信息
help:
	@echo "可用的命令:"
	@echo "  make build        - 构建可执行文件"
	@echo "  make install      - 安装到系统"
	@echo "  make clean        - 清理构建结果"
	@echo "  make test         - 运行测试"
	@echo "  make coverage     - 生成测试覆盖率报告"
	@echo "  make release-test - 测试 GoReleaser 配置"
	@echo "  make release      - 创建新的发布版本"
	@echo "  make help         - 显示帮助信息" 