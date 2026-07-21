package report

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/cyberspacesec/snir-skills/pkg/models"
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

func TestReadCSVResults(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "read_csv_test")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tempDir)

	csvFile := filepath.Join(tempDir, "test.csv")
	content := `URL,标题,响应码,截图路径,扫描时间,最终URL,状态
https://example.com,Test,200,/screenshots/example.png,2024-01-15T10:30:00Z,https://example.com/,成功
https://example.org,Test2,404,,2024-01-15T10:31:00Z,,失败: connection refused
`
	if err := os.WriteFile(csvFile, []byte(content), 0644); err != nil {
		t.Fatalf("写入CSV文件失败: %v", err)
	}

	results, err := readCSVResults(csvFile)
	if err != nil {
		t.Fatalf("readCSVResults 返回错误: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("期望2条结果, got %d", len(results))
	}

	// 验证第一条结果
	if results[0].URL != "https://example.com" {
		t.Errorf("期望URL=https://example.com, got %s", results[0].URL)
	}
	if results[0].Title != "Test" {
		t.Errorf("期望Title=Test, got %s", results[0].Title)
	}
	if results[0].ResponseCode != 200 {
		t.Errorf("期望ResponseCode=200, got %d", results[0].ResponseCode)
	}
	if results[0].Filename != "/screenshots/example.png" {
		t.Errorf("期望Filename=/screenshots/example.png, got %s", results[0].Filename)
	}
	if results[0].FinalURL != "https://example.com/" {
		t.Errorf("期望FinalURL=https://example.com/, got %s", results[0].FinalURL)
	}
	if results[0].Failed {
		t.Error("期望Failed=false")
	}

	// 验证第二条结果
	if results[1].ResponseCode != 404 {
		t.Errorf("期望ResponseCode=404, got %d", results[1].ResponseCode)
	}
	if !results[1].Failed {
		t.Error("期望Failed=true")
	}
	if results[1].FailedReason != "connection refused" {
		t.Errorf("期望FailedReason=connection refused, got %s", results[1].FailedReason)
	}
}

func TestReadCSVResults_FileNotFound(t *testing.T) {
	results, err := readCSVResults("nonexistent.csv")
	if err == nil {
		t.Error("readCSVResults 应返回错误")
	}
	if results != nil {
		t.Error("readCSVResults 应返回 nil results")
	}
}

func TestReadCSVResults_EmptyFile(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "read_csv_empty_test")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tempDir)

	csvFile := filepath.Join(tempDir, "empty.csv")
	if err := os.WriteFile(csvFile, []byte(""), 0644); err != nil {
		t.Fatalf("写入CSV文件失败: %v", err)
	}

	results, err := readCSVResults(csvFile)
	if err == nil {
		t.Error("空CSV文件应返回错误")
	}
	if results != nil {
		t.Error("空CSV文件应返回 nil results")
	}
}

func TestReadResults_JSONL(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "read_results_jsonl_test")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tempDir)

	jsonlFile := filepath.Join(tempDir, "test.jsonl")
	content := `{"URL":"https://example.com","Title":"Test","ResponseCode":200}
{"URL":"https://example.org","Title":"Test2","ResponseCode":200}
`
	if err := os.WriteFile(jsonlFile, []byte(content), 0644); err != nil {
		t.Fatalf("创建JSONL文件失败: %v", err)
	}

	results, err := readResults(jsonlFile, ".jsonl")
	if err != nil {
		t.Fatalf("readResults 失败: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("期望读取2条记录，但得到了 %d 条", len(results))
	}
}

func TestReadResults_UnsupportedExtension(t *testing.T) {
	_, err := readResults("/tmp/test.xyz", ".xyz")
	if err == nil {
		t.Error("不支持的扩展名应返回错误")
	}
}

func TestWriteResults_JSONL(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "write_results_test")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tempDir)

	outputFile := filepath.Join(tempDir, "output.jsonl")
	results := createTestResults()

	err = writeResults(outputFile, ".jsonl", results)
	if err != nil {
		t.Fatalf("writeResults 失败: %v", err)
	}

	// 验证写入的内容
	readResults, err := ReadJSONLResults(outputFile)
	if err != nil {
		t.Fatalf("读取写入结果失败: %v", err)
	}
	if len(readResults) != len(results) {
		t.Errorf("期望 %d 条记录, 但读取到 %d 条", len(results), len(readResults))
	}
}

func TestWriteResults_CSV(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "write_csv_test")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tempDir)

	outputFile := filepath.Join(tempDir, "output.csv")
	results := createTestResults()

	err = writeResults(outputFile, ".csv", results)
	if err != nil {
		t.Fatalf("writeResults CSV 失败: %v", err)
	}

	// 验证文件存在且非空
	if _, err := os.Stat(outputFile); os.IsNotExist(err) {
		t.Error("CSV 输出文件应存在")
	} else {
		content, err := os.ReadFile(outputFile)
		if err != nil {
			t.Fatalf("读取CSV文件失败: %v", err)
		}
		if len(content) == 0 {
			t.Error("CSV 输出文件不应为空")
		}
	}
}

func TestWriteResults_EmptyResults(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "write_empty_test")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tempDir)

	outputFile := filepath.Join(tempDir, "output.jsonl")
	emptyResults := []*models.Result{}

	err = writeResults(outputFile, ".jsonl", emptyResults)
	if err != nil {
		t.Fatalf("writeResults 空结果失败: %v", err)
	}

	content, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("读取文件失败: %v", err)
	}
	if len(content) != 0 {
		t.Errorf("空结果应生成空文件，但得到 %d 字节", len(content))
	}
}

func TestWriteResults_UnsupportedExtension(t *testing.T) {
	err := writeResults("/tmp/test.xyz", ".xyz", nil)
	if err == nil {
		t.Error("不支持的扩展名应返回错误")
	}
}

func TestWriteResults_NestedDirectory(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "write_nested_test")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 输出到不存在的子目录
	outputFile := filepath.Join(tempDir, "sub", "dir", "output.jsonl")
	results := createTestResults()

	err = writeResults(outputFile, ".jsonl", results)
	if err != nil {
		t.Fatalf("writeResults 嵌套目录失败: %v", err)
	}

	// 验证文件存在
	if _, err := os.Stat(outputFile); os.IsNotExist(err) {
		t.Error("嵌套目录的输出文件应存在")
	}
}

func TestReadResults_DB(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "read_db_test")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 先创建并写入 SQLite 数据库
	dbFile := filepath.Join(tempDir, "test.db")
	results := createTestResults()

	err = writeResults(dbFile, ".db", results)
	if err != nil {
		t.Fatalf("写入 DB 失败: %v", err)
	}

	// 读取回来
	readBack, err := readResults(dbFile, ".db")
	if err != nil {
		t.Fatalf("readResults DB 失败: %v", err)
	}
	if len(readBack) != len(results) {
		t.Errorf("期望 %d 条记录，但读取到 %d 条", len(results), len(readBack))
	}
}

func TestReadResults_SQLite3(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "read_sqlite3_test")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tempDir)

	dbFile := filepath.Join(tempDir, "test.sqlite3")
	results := createTestResults()

	err = writeResults(dbFile, ".sqlite3", results)
	if err != nil {
		t.Fatalf("写入 SQLite3 失败: %v", err)
	}

	readBack, err := readResults(dbFile, ".sqlite3")
	if err != nil {
		t.Fatalf("readResults SQLite3 失败: %v", err)
	}
	if len(readBack) != len(results) {
		t.Errorf("期望 %d 条记录，但读取到 %d 条", len(results), len(readBack))
	}
}

func TestConvert_JSONLToDB(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "convert_jsondb_test")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 创建源 JSONL
	jsonlFile := filepath.Join(tempDir, "source.jsonl")
	results := createTestResults()

	err = writeResults(jsonlFile, ".jsonl", results)
	if err != nil {
		t.Fatalf("创建源文件失败: %v", err)
	}

	outputFile := filepath.Join(tempDir, "output.db")

	err = Convert(ConvertOptions{
		FromFile: jsonlFile,
		ToFile:   outputFile,
	})
	if err != nil {
		t.Fatalf("Convert JSONL->DB 失败: %v", err)
	}

	// 验证输出
	readBack, err := readResults(outputFile, ".db")
	if err != nil {
		t.Fatalf("读取输出失败: %v", err)
	}
	if len(readBack) != len(results) {
		t.Errorf("期望 %d 条记录，但得到 %d 条", len(results), len(readBack))
	}
}

func TestConvert_JSONLToCSV(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "convert_jsoncsv_test")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tempDir)

	jsonlFile := filepath.Join(tempDir, "source.jsonl")
	results := createTestResults()

	err = writeResults(jsonlFile, ".jsonl", results)
	if err != nil {
		t.Fatalf("创建源文件失败: %v", err)
	}

	outputFile := filepath.Join(tempDir, "output.csv")

	err = Convert(ConvertOptions{
		FromFile: jsonlFile,
		ToFile:   outputFile,
	})
	if err != nil {
		t.Fatalf("Convert JSONL->CSV 失败: %v", err)
	}

	if _, err := os.Stat(outputFile); os.IsNotExist(err) {
		t.Error("CSV 输出文件应存在")
	}
}

func TestConvert_SameFormat(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "convert_same_test")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tempDir)

	jsonlFile := filepath.Join(tempDir, "source.jsonl")
	results := createTestResults()

	err = writeResults(jsonlFile, ".jsonl", results)
	if err != nil {
		t.Fatalf("创建源文件失败: %v", err)
	}

	outputFile := filepath.Join(tempDir, "output.jsonl")

	err = Convert(ConvertOptions{
		FromFile: jsonlFile,
		ToFile:   outputFile,
	})
	if err != nil {
		t.Fatalf("Convert JSONL->JSONL 失败: %v", err)
	}
}

func TestConvert_DBToJSONL(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "convert_dbjson_test")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tempDir)

	dbFile := filepath.Join(tempDir, "source.db")
	results := createTestResults()

	err = writeResults(dbFile, ".db", results)
	if err != nil {
		t.Fatalf("创建源DB文件失败: %v", err)
	}

	outputFile := filepath.Join(tempDir, "output.jsonl")

	err = Convert(ConvertOptions{
		FromFile: dbFile,
		ToFile:   outputFile,
	})
	if err != nil {
		t.Fatalf("Convert DB->JSONL 失败: %v", err)
	}

	// 验证
	readBack, err := ReadJSONLResults(outputFile)
	if err != nil {
		t.Fatalf("读取输出失败: %v", err)
	}
	// DB roundtrip 后可能字段略有差异（如 id），只检查数量
	if len(readBack) != len(results) {
		t.Errorf("期望 %d 条记录，但得到 %d 条", len(results), len(readBack))
	}
}

// TestConvert_ErrorBranches 覆盖 Convert 的各错误分支。
func TestConvert_ErrorBranches(t *testing.T) {
	t.Run("empty from file", func(t *testing.T) {
		if err := Convert(ConvertOptions{ToFile: "out.jsonl"}); err == nil {
			t.Fatal("期望输入文件为空错误")
		}
	})
	t.Run("empty to file", func(t *testing.T) {
		if err := Convert(ConvertOptions{FromFile: "in.jsonl"}); err == nil {
			t.Fatal("期望输出文件为空错误")
		}
	})
	t.Run("from file not exist", func(t *testing.T) {
		if err := Convert(ConvertOptions{FromFile: "/nonexistent/path/in.jsonl", ToFile: "out.jsonl"}); err == nil {
			t.Fatal("期望输入文件不存在错误")
		}
	})
	t.Run("invalid from ext", func(t *testing.T) {
		tempDir := t.TempDir()
		from := filepath.Join(tempDir, "in.txt")
		if err := os.WriteFile(from, []byte("x"), 0644); err != nil {
			t.Fatalf("写文件失败: %v", err)
		}
		if err := Convert(ConvertOptions{FromFile: from, ToFile: "out.jsonl"}); err == nil {
			t.Fatal("期望不支持的输入格式错误")
		}
	})
	t.Run("invalid to ext", func(t *testing.T) {
		tempDir := t.TempDir()
		from := filepath.Join(tempDir, "in.jsonl")
		if err := os.WriteFile(from, []byte(`{"url":"https://x"}`), 0644); err != nil {
			t.Fatalf("写文件失败: %v", err)
		}
		if err := Convert(ConvertOptions{FromFile: from, ToFile: "out.txt"}); err == nil {
			t.Fatal("期望不支持的输出格式错误")
		}
	})
}

// TestWriteResults_UnsupportedExt 覆盖 writeResults default 分支。
func TestWriteResults_UnsupportedExt(t *testing.T) {
	if err := writeResults(filepath.Join(t.TempDir(), "out.unknown"), ".unknown", createTestResults()); err == nil {
		t.Fatal("期望不支持的格式错误")
	}
}

// TestReadResults_UnsupportedExt 覆盖 readResults default 分支。
func TestReadResults_UnsupportedExt(t *testing.T) {
	if _, err := readResults("file.unknown", ".unknown"); err == nil {
		t.Fatal("期望不支持的格式错误")
	}
}

// TestReadCSVResults_BadHeader 覆盖 readCSVResults 表头为空分支。
func TestReadCSVResults_BadHeader(t *testing.T) {
	tempDir := t.TempDir()
	file := filepath.Join(tempDir, "bad.csv")
	// 空白文件触发"CSV文件为空"分支
	if err := os.WriteFile(file, []byte("\n\n"), 0644); err != nil {
		t.Fatalf("写文件失败: %v", err)
	}
	if _, err := readCSVResults(file); err == nil {
		t.Fatal("期望 CSV 文件为空错误")
	}
}

// TestReadResults_SQLiteOpenFailure 覆盖 readResults 的 SQLite 连接失败分支
// （convert.go:88-90）。用无法创建的目录路径让 NewDB 失败。
func TestReadResults_SQLiteOpenFailure(t *testing.T) {
	_, err := readResults("/nonexistent-dir/cannot-create.db", ".db")
	if err == nil {
		t.Fatal("无效 DB 路径应返回错误")
	}
}

// TestWriteResults_SQLiteCreateFailure 覆盖 writeResults 的 SQLite 创建失败分支
// （convert.go:118-120）。
func TestWriteResults_SQLiteCreateFailure(t *testing.T) {
	err := writeResults("/nonexistent-dir/cannot-create.db", ".db", createTestResults())
	if err == nil {
		t.Fatal("无效 DB 路径应返回错误")
	}
}

// TestWriteResults_CSVCreateWriterFailure 覆盖 writeResults 的 CSV 创建写入器失败分支
// （convert.go:142-144）。用无法创建的路径让 NewCSVWriter 失败。
func TestWriteResults_CSVCreateWriterFailure(t *testing.T) {
	err := writeResults("/nonexistent-dir/sub/out.csv", ".csv", createTestResults())
	if err == nil {
		t.Fatal("无效路径应返回错误")
	}
}

// TestReadCSVResults_NoHeaderTreatsFirstAsData 覆盖 readCSVResults 的
// 无表头分支（convert.go:186-187，首行不以 "URL" 开头时当数据行处理）。
func TestReadCSVResults_NoHeaderTreatsFirstAsData(t *testing.T) {
	tempDir := t.TempDir()
	csvFile := filepath.Join(tempDir, "noheader.csv")
	// 首行不以 URL 开头，无表头，但行>=7列
	content := "http://a.com,Title,200,shot.png,time,http://a.com,成功\n"
	content += "http://b.com,Title2,404,shot2.png,time2,http://b.com,失败\n"
	if err := os.WriteFile(csvFile, []byte(content), 0644); err != nil {
		t.Fatalf("写入文件失败: %v", err)
	}
	results, err := readCSVResults(csvFile)
	if err != nil {
		t.Fatalf("readCSVResults 失败: %v", err)
	}
	// 无表头时首行也当数据，应返回 2 条
	if len(results) != 2 {
		t.Errorf("无表头应将首行当数据, got %d 条", len(results))
	}
}

// TestReadCSVResults_RowTooFewColumns 覆盖 readCSVResults 的
// 行<7列跳过分支（convert.go:189-191）。csv.ReadAll 要求列数一致，
// 故用所有行都<7列的文件，让每行跳过、返回 0 条。
func TestReadCSVResults_RowTooFewColumns(t *testing.T) {
	tempDir := t.TempDir()
	csvFile := filepath.Join(tempDir, "fewcols.csv")
	content := "URL,标题,响应码\n"             // 表头 3 列
	content += "http://a.com,Title,200\n" // 3 列 <7 应跳过
	content += "http://b.com,Title2,404\n"
	if err := os.WriteFile(csvFile, []byte(content), 0644); err != nil {
		t.Fatalf("写入文件失败: %v", err)
	}
	results, err := readCSVResults(csvFile)
	if err != nil {
		t.Fatalf("readCSVResults 失败: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("所有行<7列应全部跳过, got %d 条", len(results))
	}
}

// TestReadCSVResults_ReadAllError 覆盖 readCSVResults 的 csv.ReadAll 失败分支
// （convert.go:169-171）。csv.ReadAll 要求每行列数一致，列数不一致时返回错误。
func TestReadCSVResults_ReadAllError(t *testing.T) {
	tempDir := t.TempDir()
	csvFile := filepath.Join(tempDir, "inconsistent.csv")
	// 第一行 7 列，第二行 5 列 → ReadAll 报错
	content := "URL,标题,响应码,截图,时间,最终URL,状态\n"
	content += "http://a.com,T,200,s,t,u,成功\n"
	content += "http://b.com,T2,404,s2,t2,失败\n"
	content += "http://c.com,T3,404\n" // 4 列，不一致
	if err := os.WriteFile(csvFile, []byte(content), 0644); err != nil {
		t.Fatalf("写入文件失败: %v", err)
	}
	_, err := readCSVResults(csvFile)
	if err == nil {
		t.Fatal("列数不一致应触发 ReadAll 错误")
	}
}
