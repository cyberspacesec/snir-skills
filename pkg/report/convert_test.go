package report

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/cyberspacesec/go-snir/pkg/models"
)

// TestIsValidExtension 测试文件扩展名验证
func TestIsValidExtension(t *testing.T) {
	tests := []struct {
		name     string
		ext      string
		expected bool
	}{
		{"SQLite数据库格式", ".sqlite3", true},
		{"数据库格式", ".db", true},
		{"JSONL格式", ".jsonl", true},
		{"CSV格式", ".csv", true},
		{"不支持的格式", ".txt", false},
		{"不支持的格式", ".pdf", false},
		{"空字符串", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidExtension(tt.ext)
			if result != tt.expected {
				t.Errorf("isValidExtension(%q) = %v, 期望 %v", tt.ext, result, tt.expected)
			}
		})
	}
}

// TestConvertOptions 测试转换选项验证
func TestConvertOptions(t *testing.T) {
	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "convert_test")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 创建测试文件
	jsonlFile := filepath.Join(tempDir, "test.jsonl")
	if err := os.WriteFile(jsonlFile, []byte("{}"), 0644); err != nil {
		t.Fatalf("创建测试文件失败: %v", err)
	}

	// 测试用例
	tests := []struct {
		name        string
		options     ConvertOptions
		expectError bool
	}{
		{
			name: "缺少输入文件",
			options: ConvertOptions{
				FromFile: "",
				ToFile:   filepath.Join(tempDir, "output.jsonl"),
			},
			expectError: true,
		},
		{
			name: "缺少输出文件",
			options: ConvertOptions{
				FromFile: jsonlFile,
				ToFile:   "",
			},
			expectError: true,
		},
		{
			name: "输入文件不存在",
			options: ConvertOptions{
				FromFile: filepath.Join(tempDir, "nonexistent.jsonl"),
				ToFile:   filepath.Join(tempDir, "output.jsonl"),
			},
			expectError: true,
		},
		{
			name: "不支持的输入格式",
			options: ConvertOptions{
				FromFile: filepath.Join(tempDir, "test.txt"),
				ToFile:   filepath.Join(tempDir, "output.jsonl"),
			},
			expectError: true,
		},
		{
			name: "不支持的输出格式",
			options: ConvertOptions{
				FromFile: jsonlFile,
				ToFile:   filepath.Join(tempDir, "output.txt"),
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Convert(tt.options)
			if tt.expectError && err == nil {
				t.Errorf("期望错误，但获得了nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("期望成功，但出错: %v", err)
			}
		})
	}
}

// TestReadResults 测试从不同格式文件读取结果
func TestReadResults(t *testing.T) {
	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "read_results_test")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 测试JSONL读取 (创建有效的JSONL文件)
	jsonlFile := filepath.Join(tempDir, "test.jsonl")
	jsonlContent := `{"URL":"https://example.com","Title":"Example Domain","ResponseCode":200}
{"URL":"https://example.org","Title":"Example.org","ResponseCode":200}`

	if err := os.WriteFile(jsonlFile, []byte(jsonlContent), 0644); err != nil {
		t.Fatalf("创建JSONL测试文件失败: %v", err)
	}

	// 测试从JSONL文件读取
	results, err := ReadJSONLResults(jsonlFile)
	if err != nil {
		t.Fatalf("从JSONL读取失败: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("期望读取到2条记录，但得到了 %d 条", len(results))
	}

	// 验证结果内容
	if results[0].URL != "https://example.com" {
		t.Errorf("第一条记录URL不匹配，期望 https://example.com，得到 %s", results[0].URL)
	}
	if results[1].URL != "https://example.org" {
		t.Errorf("第二条记录URL不匹配，期望 https://example.org，得到 %s", results[1].URL)
	}

	// 测试不支持的格式
	_, err = readResults("/nonexistent/file.xyz", ".xyz")
	if err == nil {
		t.Error("对于不支持的格式，期望错误，但获得了nil")
	}
}

// 测试用的模拟结果
func createTestResults() []*models.Result {
	return []*models.Result{
		{
			URL:            "https://example.com",
			Title:          "Example Domain",
			ResponseCode:   200,
			ResponseReason: "OK",
		},
		{
			URL:            "https://example.org",
			Title:          "Example.org",
			ResponseCode:   200,
			ResponseReason: "OK",
		},
	}
}
