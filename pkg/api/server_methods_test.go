package api

import (
	"testing"

	"github.com/cyberspacesec/snir-skills/pkg/models"
	"github.com/cyberspacesec/snir-skills/pkg/runner"
)

// MockDriver 是一个用于测试的简化模拟驱动
type MockDriver struct {
	WitnessCalled bool
	ReturnResult  *models.Result
	ReturnError   error
}

func (d *MockDriver) Witness(target string, runner *runner.Runner) (*models.Result, error) {
	d.WitnessCalled = true
	if d.ReturnResult != nil {
		d.ReturnResult.URL = target // 确保URL与请求匹配
	}
	return d.ReturnResult, d.ReturnError
}

func (d *MockDriver) Close() {
	// 模拟关闭方法
}

// TestGetBlacklist 测试GetBlacklist方法
func TestGetBlacklist(t *testing.T) {
	// 创建服务器
	server := &Server{
		Options: ServerOptions{
			EnableBlacklist:   true,
			DefaultBlacklist:  true,
			BlacklistPatterns: []string{"test-domain.example"},
		},
	}

	// 创建runner选项
	opts := &runner.Options{}
	// 传递服务器选项到runner选项
	opts.Scan.EnableBlacklist = true
	opts.Scan.DefaultBlacklist = true
	opts.Scan.BlacklistPatterns = []string{"test-domain.example"}

	// 调用GetBlacklist
	blacklist, err := server.GetBlacklist(opts)
	if err != nil {
		t.Fatalf("GetBlacklist返回错误: %v", err)
	}

	// 检查黑名单是否被正确创建
	if blacklist == nil {
		t.Fatal("GetBlacklist返回nil")
	}

	// 测试本地IP是否被阻止（这应该是默认黑名单的一部分）
	isBlacklisted, _ := blacklist.IsBlacklisted("https://127.0.0.1")
	if !isBlacklisted {
		t.Error("默认黑名单规则未正确应用，本地IP应该被阻止")
	}

	// 测试黑名单能否正确检测非黑名单域名
	isBlacklisted, _ = blacklist.IsBlacklisted("https://safe-domain-example.org")
	if isBlacklisted {
		t.Error("安全域名不应被标记为黑名单")
	}
}

// 测试 ProcessScreenshot 方法
// 由于无法直接替换 runner.NewChromeDP，我们采用不同的测试策略
func TestProcessScreenshot(t *testing.T) {
	// 跳过实际测试，因为这个功能需要真实的Chrome实例
	// 在单元测试环境中很难模拟
	t.Skip("跳过ProcessScreenshot测试，需要Chrome实例")
}

// TestCreateScreenshotDir 测试创建截图目录
func TestCreateScreenshotDir(t *testing.T) {
	// 创建临时目录
	tempDir := t.TempDir()

	// 测试创建默认目录
	dir, err := CreateScreenshotDir("")
	if err != nil {
		t.Fatalf("创建默认截图目录失败: %v", err)
	}
	// 可以检查返回的路径是否是绝对路径
	if dir == "" {
		t.Error("CreateScreenshotDir返回空路径")
	}

	// 测试创建指定目录
	testDir := tempDir + "/screenshots"
	dir, err = CreateScreenshotDir(testDir)
	if err != nil {
		t.Fatalf("创建指定截图目录失败: %v", err)
	}
	if dir != testDir {
		t.Errorf("路径不匹配: 期望 %v, 得到 %v", testDir, dir)
	}
}
