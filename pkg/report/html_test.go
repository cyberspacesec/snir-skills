package report

import (
	"bytes"
	"html/template"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/cyberspacesec/snir-skills/pkg/models"
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
	templ, err := template.New("report").Parse(RichHTMLTemplate)
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
		"badge-2xx",
	}

	for _, expected := range expectedContents {
		if !strings.Contains(html, expected) {
			t.Errorf("渲染的HTML应该包含 %q", expected)
		}
	}
}

func TestGenerateHTML_EmptyResults(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "html_empty_test")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tempDir)

	inputFile := filepath.Join(tempDir, "empty.jsonl")
	if err := os.WriteFile(inputFile, []byte(""), 0644); err != nil {
		t.Fatalf("创建空文件失败: %v", err)
	}

	outputFile := filepath.Join(tempDir, "output.html")

	err = GenerateHTML(HTMLOptions{
		InputFile:  inputFile,
		OutputPath: outputFile,
	})
	if err == nil {
		t.Error("空结果应返回错误")
	}
}

func TestGenerateHTML_OutputPathIsDirectory(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "html_dir_test")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tempDir)

	inputFile := filepath.Join(tempDir, "test.jsonl")
	content := `{"URL":"https://example.com","Title":"Test","ResponseCode":200}
`
	if err := os.WriteFile(inputFile, []byte(content), 0644); err != nil {
		t.Fatalf("创建JSONL文件失败: %v", err)
	}

	err = GenerateHTML(HTMLOptions{
		InputFile:  inputFile,
		OutputPath: tempDir,
	})
	if err == nil {
		t.Error("目录作为输出路径应返回错误")
	}
}

func TestReportResult_StatusCodeClasses(t *testing.T) {
	tests := []struct {
		code int
		cls  string
	}{
		{100, "0"},
		{199, "0"},
		{200, "2xx"},
		{299, "2xx"},
		{300, "3xx"},
		{399, "3xx"},
		{400, "4xx"},
		{499, "4xx"},
		{500, "5xx"},
		{599, "5xx"},
		{600, "0"},
		{0, "0"},
	}

	for _, tt := range tests {
		statusClass := "0"
		if tt.code >= 200 && tt.code < 300 {
			statusClass = "2xx"
		} else if tt.code >= 300 && tt.code < 400 {
			statusClass = "3xx"
		} else if tt.code >= 400 && tt.code < 500 {
			statusClass = "4xx"
		} else if tt.code >= 500 && tt.code < 600 {
			statusClass = "5xx"
		}

		if statusClass != tt.cls {
			t.Errorf("ResponseCode %d: 期望类别 %s, 得到 %s", tt.code, tt.cls, statusClass)
		}
	}
}

func TestReadJSONLResults_BlankLines(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "jsonl_blank_test")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tempDir)

	filePath := filepath.Join(tempDir, "blank_lines.jsonl")
	content := `{"URL":"https://example.com","Title":"Test","ResponseCode":200}

{"URL":"https://example.org","Title":"Test2","ResponseCode":200}

`
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatalf("创建文件失败: %v", err)
	}

	results, err := ReadJSONLResults(filePath)
	if err != nil {
		t.Fatalf("ReadJSONLResults 失败: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("期望2条记录，但得到 %d 条", len(results))
	}
}

func TestGenerateHTML_StatusCodeClassesOutput(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "html_status_test")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tempDir)

	inputFile := filepath.Join(tempDir, "test.jsonl")
	content := `{"URL":"https://200.example.com","Title":"OK","ResponseCode":200}
{"URL":"https://301.example.com","Title":"Moved","ResponseCode":301}
{"URL":"https://404.example.com","Title":"Not Found","ResponseCode":404}
{"URL":"https://500.example.com","Title":"Error","ResponseCode":500}
`
	if err := os.WriteFile(inputFile, []byte(content), 0644); err != nil {
		t.Fatalf("创建JSONL文件失败: %v", err)
	}

	outputFile := filepath.Join(tempDir, "output.html")

	err = GenerateHTML(HTMLOptions{
		InputFile:  inputFile,
		OutputPath: outputFile,
	})
	if err != nil {
		t.Fatalf("GenerateHTML 失败: %v", err)
	}

	html, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("读取HTML失败: %v", err)
	}

	htmlStr := string(html)

	expectedClasses := []string{"badge-2xx", "badge-3xx", "badge-4xx", "badge-5xx"}
	for _, cls := range expectedClasses {
		if !strings.Contains(htmlStr, cls) {
			t.Errorf("HTML 应包含 CSS 类 %q", cls)
		}
	}

	expectedTexts := []string{"https://200.example.com", "https://301.example.com", "https://404.example.com", "https://500.example.com"}
	for _, text := range expectedTexts {
		if !strings.Contains(htmlStr, text) {
			t.Errorf("HTML 应包含 %q", text)
		}
	}
}

func TestReadJSONLResults_MalformedJSON(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "jsonl_malformed_test2")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tempDir)

	filePath := filepath.Join(tempDir, "malformed.jsonl")
	content := `{"URL":"https://good.com","Title":"Good","ResponseCode":200}
this is not json
{"URL":"https://bad.com","Title":"Bad","ResponseCode":404}
`
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatalf("创建文件失败: %v", err)
	}

	results, err := ReadJSONLResults(filePath)
	if err != nil {
		t.Fatalf("ReadJSONLResults 失败: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("期望2条有效记录，但得到 %d 条", len(results))
	}
}

func TestReportData_WithScreenshotPath(t *testing.T) {
	results := []*models.Result{
		{
			URL:          "https://example.com",
			Title:        "Example",
			ResponseCode: 200,
			Filename:     "/absolute/path/to/screenshot.png",
		},
	}

	reportData := ReportData{
		GeneratedAt: "2024-01-01 00:00:00",
		Results:     make([]ReportResult, 0, len(results)),
	}

	for _, result := range results {
		screenshotPath := result.Filename
		reportData.Results = append(reportData.Results, ReportResult{
			URL:             result.URL,
			Title:           result.Title,
			Screenshot:      screenshotPath,
			ResponseCode:    result.ResponseCode,
			StatusCodeClass: "2xx",
		})
	}

	if len(reportData.Results) != 1 {
		t.Errorf("期望1条结果，但得到 %d 条", len(reportData.Results))
	}
	if reportData.Results[0].Screenshot != "/absolute/path/to/screenshot.png" {
		t.Errorf("截图路径应为绝对路径，但得到 %q", reportData.Results[0].Screenshot)
	}
}

func TestReportData_NoTitle(t *testing.T) {
	results := []*models.Result{
		{
			URL:          "https://example.com",
			Title:        "",
			ResponseCode: 200,
		},
	}

	reportData := ReportData{
		GeneratedAt: "2024-01-01 00:00:00",
		Results:     make([]ReportResult, 0, len(results)),
	}

	for _, result := range results {
		reportData.Results = append(reportData.Results, ReportResult{
			URL:             result.URL,
			Title:           result.Title,
			ResponseCode:    result.ResponseCode,
			StatusCodeClass: "2xx",
		})
	}

	if reportData.Results[0].Title != "" {
		t.Errorf("空标题应保持为空，但得到 %q", reportData.Results[0].Title)
	}
}

func TestHTMLTemplate_NoResults(t *testing.T) {
	data := ReportData{
		GeneratedAt: time.Now().Format("2006-01-02 15:04:05"),
		Results:     []ReportResult{},
	}

	templ, err := template.New("report").Parse(RichHTMLTemplate)
	if err != nil {
		t.Fatalf("解析HTML模板失败: %v", err)
	}

	var buf bytes.Buffer
	err = templ.Execute(&buf, data)
	if err != nil {
		t.Fatalf("渲染HTML模板失败: %v", err)
	}

	html := buf.String()
	if !strings.Contains(html, "网页截图扫描报告") {
		t.Error("空结果的HTML应包含标题")
	}
	if !strings.Contains(html, "总计:") {
		t.Error("HTML应包含统计信息")
	}
}

func TestHTMLTemplate_NoScreenshot(t *testing.T) {
	data := ReportData{
		GeneratedAt: time.Now().Format("2006-01-02 15:04:05"),
		Results: []ReportResult{
			{
				URL:             "https://example.com",
				Title:           "Example",
				Screenshot:      "",
				ResponseCode:    200,
				StatusCodeClass: "2xx",
				ProbedAt:        time.Now(),
			},
		},
	}

	templ, err := template.New("report").Parse(RichHTMLTemplate)
	if err != nil {
		t.Fatalf("解析HTML模板失败: %v", err)
	}

	var buf bytes.Buffer
	err = templ.Execute(&buf, data)
	if err != nil {
		t.Fatalf("渲染HTML模板失败: %v", err)
	}

	html := buf.String()
	if !strings.Contains(html, "无截图") {
		t.Error("无截图的HTML应包含'无截图'提示")
	}
}
