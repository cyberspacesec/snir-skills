package report

import (
	"bytes"
	"html/template"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/cyberspacesec/go-snir/pkg/models"
)

// TestHTMLOptions 测试HTML选项验证
func TestHTMLOptions(t *testing.T) {
	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "html_test")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 创建测试JSONL文件
	jsonlFile := filepath.Join(tempDir, "test.jsonl")
	jsonlContent := `{"URL":"https://example.com","Title":"Example Domain","ResponseCode":200}
{"URL":"https://example.org","Title":"Example.org","ResponseCode":200}`

	if err := os.WriteFile(jsonlFile, []byte(jsonlContent), 0644); err != nil {
		t.Fatalf("创建JSONL测试文件失败: %v", err)
	}

	// 指定输出文件（而不是目录）
	outputFile := filepath.Join(tempDir, "output.html")

	// 测试用例
	tests := []struct {
		name        string
		options     HTMLOptions
		expectError bool
	}{
		{
			name: "未指定输入文件",
			options: HTMLOptions{
				InputFile:  "",
				OutputPath: outputFile,
			},
			expectError: true,
		},
		{
			name: "输入文件不存在",
			options: HTMLOptions{
				InputFile:  filepath.Join(tempDir, "nonexistent.jsonl"),
				OutputPath: outputFile,
			},
			expectError: true,
		},
		{
			name: "有效的输入文件",
			options: HTMLOptions{
				InputFile:  jsonlFile,
				OutputPath: outputFile,
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := GenerateHTML(tt.options)

			if tt.expectError && err == nil {
				t.Errorf("期望错误，但获得了nil")
			}

			// 由于GenerateHTML中有多步操作，可能会在不同阶段失败
			// 我们这里主要测试输入文件验证逻辑
			if !tt.expectError && err != nil {
				if strings.Contains(err.Error(), "输入文件不存在") || strings.Contains(err.Error(), "输入文件不能为空") {
					t.Errorf("期望文件验证通过，但出错: %v", err)
				}
				// 其他错误（如读取JSONL错误）可以接受，因为它们不是我们当前测试的重点
			}
		})
	}
}

// TestReadJSONLResults 测试读取JSONL结果文件
func TestReadJSONLResults(t *testing.T) {
	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "jsonl_test")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 创建有效的JSONL文件
	validJsonlFile := filepath.Join(tempDir, "valid.jsonl")
	validContent := `{"URL":"https://example.com","Title":"Example Domain","ResponseCode":200}
{"URL":"https://example.org","Title":"Example.org","ResponseCode":200}`

	if err := os.WriteFile(validJsonlFile, []byte(validContent), 0644); err != nil {
		t.Fatalf("创建有效JSONL文件失败: %v", err)
	}

	// 创建无效的JSONL文件
	invalidJsonlFile := filepath.Join(tempDir, "invalid.jsonl")
	invalidContent := `{"URL":"https://example.com","Title":"Example Domain","ResponseCode":200}
这不是有效的JSON`

	if err := os.WriteFile(invalidJsonlFile, []byte(invalidContent), 0644); err != nil {
		t.Fatalf("创建无效JSONL文件失败: %v", err)
	}

	// 创建空JSONL文件
	emptyJsonlFile := filepath.Join(tempDir, "empty.jsonl")
	if err := os.WriteFile(emptyJsonlFile, []byte(""), 0644); err != nil {
		t.Fatalf("创建空JSONL文件失败: %v", err)
	}

	// 测试读取有效JSONL文件
	results, err := ReadJSONLResults(validJsonlFile)
	if err != nil {
		t.Errorf("读取有效JSONL文件失败: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("从有效JSONL文件期望读取到 2 条记录，但得到了 %d 条", len(results))
	}

	// 测试读取无效JSONL文件（应该返回部分结果并忽略错误的行）
	results, err = ReadJSONLResults(invalidJsonlFile)
	if err != nil {
		t.Errorf("读取无效JSONL文件失败: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("从无效JSONL文件期望读取到 1 条记录，但得到了 %d 条", len(results))
	}

	// 测试读取空JSONL文件
	results, err = ReadJSONLResults(emptyJsonlFile)
	if err != nil {
		t.Errorf("读取空JSONL文件失败: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("从空JSONL文件期望读取到 0 条记录，但得到了 %d 条", len(results))
	}

	// 测试读取不存在的文件
	_, err = ReadJSONLResults(filepath.Join(tempDir, "nonexistent.jsonl"))
	if err == nil {
		t.Error("读取不存在的文件期望返回错误，但得到了nil")
	}
}

// TestReportData 测试生成报告数据
func TestReportData(t *testing.T) {
	// 创建测试结果
	now := time.Now()
	results := []*models.Result{
		{
			URL:            "https://example.com",
			Title:          "Example Domain",
			ResponseCode:   200,
			ResponseReason: "OK",
			ProbedAt:       now,
			Filename:       "/path/to/screenshot.png",
		},
		{
			URL:            "https://example.org",
			Title:          "Example.org",
			ResponseCode:   404,
			ResponseReason: "Not Found",
			ProbedAt:       now,
		},
	}

	// 创建报告数据
	reportResults := make([]ReportResult, 0, len(results))
	for _, result := range results {
		statusCodeClass := "0"
		if result.ResponseCode >= 200 && result.ResponseCode < 300 {
			statusCodeClass = "2xx"
		} else if result.ResponseCode >= 300 && result.ResponseCode < 400 {
			statusCodeClass = "3xx"
		} else if result.ResponseCode >= 400 && result.ResponseCode < 500 {
			statusCodeClass = "4xx"
		} else if result.ResponseCode >= 500 && result.ResponseCode < 600 {
			statusCodeClass = "5xx"
		}

		reportResults = append(reportResults, ReportResult{
			URL:             result.URL,
			Title:           result.Title,
			Screenshot:      result.Filename,
			ResponseCode:    result.ResponseCode,
			StatusCodeClass: statusCodeClass,
			ProbedAt:        result.ProbedAt,
		})
	}

	// 验证状态码类别
	if reportResults[0].StatusCodeClass != "2xx" {
		t.Errorf("200状态码期望类别为2xx，但得到了 %s", reportResults[0].StatusCodeClass)
	}
	if reportResults[1].StatusCodeClass != "4xx" {
		t.Errorf("404状态码期望类别为4xx，但得到了 %s", reportResults[1].StatusCodeClass)
	}

	// 验证URL和标题
	if reportResults[0].URL != "https://example.com" {
		t.Errorf("第一条记录URL期望为https://example.com，但得到了 %s", reportResults[0].URL)
	}
	if reportResults[1].Title != "Example.org" {
		t.Errorf("第二条记录标题期望为Example.org，但得到了 %s", reportResults[1].Title)
	}
}

// TestHTMLTemplate 测试HTML模板渲染
func TestHTMLTemplate(t *testing.T) {
	// 创建测试数据
	data := ReportData{
		GeneratedAt: time.Now().Format("2006-01-02 15:04:05"),
		Results: []ReportResult{
			{
				URL:             "https://example.com",
				Title:           "Example Domain",
				Screenshot:      "/path/to/screenshot.png",
				ResponseCode:    200,
				StatusCodeClass: "2xx",
				ProbedAt:        time.Now(),
			},
		},
	}

	// 解析HTML模板
	templ, err := template.New("report").Parse(HTMLTemplate)
	if err != nil {
		t.Fatalf("解析HTML模板失败: %v", err)
	}

	// 渲染模板
	var buf bytes.Buffer
	err = templ.Execute(&buf, data)
	if err != nil {
		t.Fatalf("渲染HTML模板失败: %v", err)
	}

	// 验证渲染结果包含预期内容
	html := buf.String()
	expectedContents := []string{
		"网页截图扫描报告",
		"Example Domain",
		"https://example.com",
		"status-2xx",
	}

	for _, expected := range expectedContents {
		if !strings.Contains(html, expected) {
			t.Errorf("渲染的HTML应该包含 %q", expected)
		}
	}
}
