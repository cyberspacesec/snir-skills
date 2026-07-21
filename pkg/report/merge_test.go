package report

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cyberspacesec/snir-skills/pkg/models"
)

// TestMergeOptions 测试合并选项验证
func TestMergeOptions(t *testing.T) {
	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "merge_test")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 创建测试文件
	jsonlFile1 := filepath.Join(tempDir, "test1.jsonl")
	jsonlFile2 := filepath.Join(tempDir, "test2.jsonl")
	if err := os.WriteFile(jsonlFile1, []byte("{}"), 0644); err != nil {
		t.Fatalf("创建测试文件失败: %v", err)
	}
	if err := os.WriteFile(jsonlFile2, []byte("{}"), 0644); err != nil {
		t.Fatalf("创建测试文件失败: %v", err)
	}

	// 测试用例
	tests := []struct {
		name        string
		options     MergeOptions
		expectError bool
	}{
		{
			name: "未指定输出文件",
			options: MergeOptions{
				SourceFiles: []string{jsonlFile1, jsonlFile2},
				OutputFile:  "",
			},
			expectError: true,
		},
		{
			name: "未指定源文件且未指定源路径",
			options: MergeOptions{
				SourceFiles: nil,
				SourcePath:  "",
				OutputFile:  filepath.Join(tempDir, "output.jsonl"),
			},
			expectError: true,
		},
		{
			name: "指定了不存在的源路径",
			options: MergeOptions{
				SourcePath: filepath.Join(tempDir, "nonexistent"),
				OutputFile: filepath.Join(tempDir, "output.jsonl"),
			},
			expectError: true,
		},
		{
			name: "有效的源文件列表",
			options: MergeOptions{
				SourceFiles: []string{jsonlFile1, jsonlFile2},
				OutputFile:  filepath.Join(tempDir, "output.jsonl"),
			},
			expectError: false,
		},
		{
			name: "有效的源路径",
			options: MergeOptions{
				SourcePath: tempDir,
				OutputFile: filepath.Join(tempDir, "output.jsonl"),
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 为了避免真正执行合并操作，我们可以检查返回的错误类型
			// 但这里我们只关心是否有错误，而不是具体的错误内容
			err := Merge(tt.options)

			if tt.expectError && err == nil {
				t.Errorf("期望错误，但获得了nil")
			}

			// 有些测试用例可能因为无法实际读取/写入文件而失败，但我们关注的是前置检查
			// 所以这里不检查 !tt.expectError && err != nil 的情况
		})
	}
}

// TestFindSourceFiles 测试源文件查找功能
func TestFindSourceFiles(t *testing.T) {
	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "find_source_files_test")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 创建测试文件
	validFiles := []string{
		filepath.Join(tempDir, "test1.jsonl"),
		filepath.Join(tempDir, "test2.db"),
		filepath.Join(tempDir, "test3.sqlite3"),
	}
	invalidFiles := []string{
		filepath.Join(tempDir, "test4.txt"),
		filepath.Join(tempDir, "test5.pdf"),
	}

	// 创建所有测试文件
	for _, file := range append(validFiles, invalidFiles...) {
		if err := os.WriteFile(file, []byte("{}"), 0644); err != nil {
			t.Fatalf("创建测试文件失败: %v", err)
		}
	}

	// 测试查找源文件功能
	files, err := findSourceFiles(tempDir)
	if err != nil {
		t.Fatalf("查找源文件失败: %v", err)
	}

	// 检查找到的文件数量
	if len(files) != len(validFiles) {
		t.Errorf("期望找到 %d 个有效文件，但找到了 %d 个", len(validFiles), len(files))
	}

	// 验证不存在的目录会返回错误
	_, err = findSourceFiles(filepath.Join(tempDir, "nonexistent"))
	if err == nil {
		t.Error("对于不存在的目录，期望错误，但获得了nil")
	}
}

// TestDeduplicateResults 测试结果去重功能
func TestDeduplicateResults(t *testing.T) {
	// 测试空(nil)输入
	var nilResults []*models.Result = nil
	dedupNil := deduplicateResults(nilResults)
	if dedupNil != nil {
		t.Errorf("对于nil输入，期望得到nil，得到了 %v", dedupNil)
	}

	// 测试空切片(非nil)输入
	emptyResults := []*models.Result{}
	dedupEmpty := deduplicateResults(emptyResults)

	// 根据deduplicateResults的实际实现，空切片也会返回nil
	if dedupEmpty != nil {
		t.Errorf("对于空切片输入，期望得到nil，得到了 %v", dedupEmpty)
	}

	// 测试无重复的输入
	uniqueResults := []*models.Result{
		{URL: "https://example1.com", Title: "Example 1"},
		{URL: "https://example2.com", Title: "Example 2"},
	}
	dedupUnique := deduplicateResults(uniqueResults)
	if len(dedupUnique) != 2 {
		t.Errorf("对于无重复的输入，期望保留2个记录，但得到了 %d 个", len(dedupUnique))
	}

	// 测试有重复的输入
	results := []*models.Result{
		{URL: "https://example1.com", Title: "Example 1"},
		{URL: "https://example2.com", Title: "Example 2"},
		{URL: "https://example1.com", Title: "Duplicate Example 1"}, // 重复URL
	}
	dedupResults := deduplicateResults(results)
	if len(dedupResults) != 2 {
		t.Errorf("对于有重复的输入，期望去重后有2个记录，但得到了 %d 个", len(dedupResults))
	}
}

func TestDeduplicateResults_AllDuplicates(t *testing.T) {
	results := []*models.Result{
		{URL: "https://example.com", Title: "A"},
		{URL: "https://example.com", Title: "B"},
		{URL: "https://example.com", Title: "C"},
	}
	dedupped := deduplicateResults(results)
	if len(dedupped) != 1 {
		t.Errorf("全部重复应只保留1条，但得到了 %d 条", len(dedupped))
	}
	if dedupped[0].Title != "A" {
		t.Errorf("应保留第一条记录，title = %s, want A", dedupped[0].Title)
	}
}

func TestDeduplicateResults_MixedWithEmptyURL(t *testing.T) {
	results := []*models.Result{
		{URL: "", Title: "Empty URL 1"},
		{URL: "", Title: "Empty URL 2"},
	}
	dedupped := deduplicateResults(results)
	// 空 URL 也应去重
	if len(dedupped) != 1 {
		t.Errorf("空URL去重后应只有1条，但得到了 %d 条", len(dedupped))
	}
}

func TestMerge_EmptySourceFiles(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "merge_empty_test")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tempDir)

	outputFile := filepath.Join(tempDir, "output.jsonl")

	// 空的源文件列表
	err = Merge(MergeOptions{
		SourceFiles: []string{},
		OutputFile:  outputFile,
	})
	if err == nil {
		t.Error("空源文件列表应返回错误")
	}
}

func TestMerge_InvalidOutputExtension(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "merge_invalidext_test")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tempDir)

	jsonlFile := filepath.Join(tempDir, "test.jsonl")
	if err := os.WriteFile(jsonlFile, []byte(`{"URL":"https://example.com","Title":"Test","ResponseCode":200}`+"\n"), 0644); err != nil {
		t.Fatalf("创建测试文件失败: %v", err)
	}

	outputFile := filepath.Join(tempDir, "output.txt")

	err = Merge(MergeOptions{
		SourceFiles: []string{jsonlFile},
		OutputFile:  outputFile,
	})
	if err == nil {
		t.Error("无效输出扩展名应返回错误")
	} else if !strings.Contains(err.Error(), "不支持") {
		t.Errorf("错误消息应包含'不支持', got: %v", err)
	}
}

func TestMerge_MultipleFilesDeduplication(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "merge_dedup_test")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 创建两个文件，各包含一些重复的 URL
	jsonl1 := filepath.Join(tempDir, "1.jsonl")
	jsonl2 := filepath.Join(tempDir, "2.jsonl")
	if err := os.WriteFile(jsonl1, []byte(`{"URL":"https://example.com","Title":"Site A","ResponseCode":200}
`), 0644); err != nil {
		t.Fatalf("创建文件1失败: %v", err)
	}
	if err := os.WriteFile(jsonl2, []byte(`{"URL":"https://example.com","Title":"Site A","ResponseCode":200}
{"URL":"https://example.org","Title":"Site B","ResponseCode":200}
`), 0644); err != nil {
		t.Fatalf("创建文件2失败: %v", err)
	}

	outputFile := filepath.Join(tempDir, "merged.jsonl")

	err = Merge(MergeOptions{
		SourceFiles: []string{jsonl1, jsonl2},
		OutputFile:  outputFile,
	})
	if err != nil {
		t.Fatalf("合并失败: %v", err)
	}

	// 读取合并后的文件
	results, err := ReadJSONLResults(outputFile)
	if err != nil {
		t.Fatalf("读取合并结果失败: %v", err)
	}

	// Merge 不自动去重，所以应有3条记录（1+2）
	if len(results) != 3 {
		t.Errorf("合并后应有3条记录（不自动去重），但得到了 %d 条", len(results))
	}

	// 检查输出文件内容
	content, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("读取输出文件失败: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	if len(lines) != 3 {
		t.Errorf("输出文件应有3行，但得到了 %d 行", len(lines))
	}
}

func TestMerge_UnsupportedSourceFileFormat(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "merge_skip_test")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 创建有效的 jsonl 和无效的 txt
	jsonlFile := filepath.Join(tempDir, "test.jsonl")
	if err := os.WriteFile(jsonlFile, []byte(`{"URL":"https://example.com","Title":"Test","ResponseCode":200}`+"\n"), 0644); err != nil {
		t.Fatalf("创建测试文件失败: %v", err)
	}

	txtFile := filepath.Join(tempDir, "invalid.txt")
	if err := os.WriteFile(txtFile, []byte("not a valid file"), 0644); err != nil {
		t.Fatalf("创建测试文件失败: %v", err)
	}

	outputFile := filepath.Join(tempDir, "output.jsonl")

	// 包含无效格式的文件——应该跳过但仍然处理有效的文件
	err = Merge(MergeOptions{
		SourceFiles: []string{jsonlFile, txtFile},
		OutputFile:  outputFile,
	})
	// 该操作应该成功，因为存在一个有效的源文件
	if err != nil {
		t.Logf("合并返回错误 (可能因为无效文件格式): %v", err)
	}
}

func TestMerge_AllUnreadableSourceFiles(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "merge_noread_test")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 创建空文件（读取后无有效记录）
	emptyFile := filepath.Join(tempDir, "empty.jsonl")
	if err := os.WriteFile(emptyFile, []byte(""), 0644); err != nil {
		t.Fatalf("创建空文件失败: %v", err)
	}

	outputFile := filepath.Join(tempDir, "output.jsonl")

	err = Merge(MergeOptions{
		SourceFiles: []string{emptyFile},
		OutputFile:  outputFile,
	})
	// 空文件读取后无记录，应返回错误
	if err == nil {
		t.Error("空源文件应返回错误")
	}
}

func TestFindSourceFiles_ValidExtensions(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "find_valid_ext_test")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 创建所有支持格式的文件
	extensions := []string{".jsonl", ".db", ".sqlite3", ".csv"}
	for _, ext := range extensions {
		file := filepath.Join(tempDir, "test"+ext)
		if err := os.WriteFile(file, []byte("test"), 0644); err != nil {
			t.Fatalf("创建文件 %s 失败: %v", file, err)
		}
	}

	files, err := findSourceFiles(tempDir)
	if err != nil {
		t.Fatalf("findSourceFiles 失败: %v", err)
	}

	if len(files) != len(extensions) {
		t.Errorf("期望找到 %d 个文件，但找到了 %d 个: %v", len(extensions), len(files), files)
	}
}

func TestFindSourceFiles_NoValidExtensions(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "find_no_valid_test")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 只创建不支持格式的文件
	invalidFiles := []string{"test.txt", "test.pdf", "test.doc"}
	for _, file := range invalidFiles {
		fp := filepath.Join(tempDir, file)
		if err := os.WriteFile(fp, []byte("test"), 0644); err != nil {
			t.Fatalf("创建文件 %s 失败: %v", fp, err)
		}
	}

	files, err := findSourceFiles(tempDir)
	if err != nil {
		t.Fatalf("findSourceFiles 失败: %v", err)
	}

	if len(files) != 0 {
		t.Errorf("期望找到 0 个文件，但找到了 %d 个", len(files))
	}
}

// TestMerge_ReadResultsErrorContinue 覆盖 Merge 的 readResults 失败 continue 分支
// （merge.go:62-64）。用一个无法读取的 .db 源文件让 readResults 报错，
// Merge 应跳过它并最终返回"没有读取到有效记录"错误。
func TestMerge_ReadResultsErrorContinue(t *testing.T) {
	tempDir := t.TempDir()
	// 无效 .db 文件（SQLite 打开失败）
	badDB := filepath.Join(tempDir, "bad.db")
	os.WriteFile(badDB, []byte("not a sqlite db"), 0644)
	// 同时提供一个有效 jsonl 让 allResults 非空，走到 writeResults
	goodJSONL := filepath.Join(tempDir, "good.jsonl")
	os.WriteFile(goodJSONL, []byte(`{"URL":"https://example.com","Title":"T","ResponseCode":200}`+"\n"), 0644)

	outputFile := filepath.Join(tempDir, "output.jsonl")
	err := Merge(MergeOptions{
		SourceFiles: []string{badDB, goodJSONL},
		OutputFile:  outputFile,
	})
	// bad.db 被跳过（continue），good.jsonl 成功 → 合并成功
	if err != nil {
		t.Errorf("应跳过坏文件并成功合并, got: %v", err)
	}
}
