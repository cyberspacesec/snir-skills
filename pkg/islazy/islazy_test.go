package islazy

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCreateDir(t *testing.T) {
	tests := []struct {
		name          string
		path          string
		expectError   bool
		expectDefault bool
	}{
		{
			name:          "空路径使用默认值",
			path:          "",
			expectError:   false,
			expectDefault: true,
		},
		{
			name:          "有效路径正常创建",
			path:          "test_dir",
			expectError:   false,
			expectDefault: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 运行测试
			result, err := CreateDir(tt.path)

			// 验证结果
			if tt.expectError && err == nil {
				t.Errorf("期望错误但获得了成功")
			}

			if !tt.expectError && err != nil {
				t.Errorf("期望成功但得到错误: %v", err)
			}

			if tt.expectDefault && filepath.Base(result) != "screenshots" {
				t.Errorf("期望默认路径但获得: %s", result)
			}

			// 清理测试目录
			if result != "" && !tt.expectDefault {
				os.RemoveAll(result)
			}
		})
	}

	// 清理默认目录
	os.RemoveAll("./screenshots")
}

func TestFileExists(t *testing.T) {
	// 创建临时测试文件
	tempFile, err := os.CreateTemp("", "file_exists_test")
	if err != nil {
		t.Fatalf("无法创建临时文件: %v", err)
	}
	defer os.Remove(tempFile.Name())
	tempFile.Close()

	// 创建临时测试目录
	tempDir, err := os.MkdirTemp("", "dir_exists_test")
	if err != nil {
		t.Fatalf("无法创建临时目录: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tests := []struct {
		name     string
		filename string
		expected bool
	}{
		{
			name:     "存在的文件",
			filename: tempFile.Name(),
			expected: true,
		},
		{
			name:     "不存在的文件",
			filename: "non_existent_file.txt",
			expected: false,
		},
		{
			name:     "目录而非文件",
			filename: tempDir,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FileExists(tt.filename)
			if result != tt.expected {
				t.Errorf("FileExists(%s) = %v, 期望 %v", tt.filename, result, tt.expected)
			}
		})
	}
}

func TestDirExists(t *testing.T) {
	// 创建临时测试文件
	tempFile, err := os.CreateTemp("", "file_exists_test")
	if err != nil {
		t.Fatalf("无法创建临时文件: %v", err)
	}
	defer os.Remove(tempFile.Name())
	tempFile.Close()

	// 创建临时测试目录
	tempDir, err := os.MkdirTemp("", "dir_exists_test")
	if err != nil {
		t.Fatalf("无法创建临时目录: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tests := []struct {
		name     string
		dirname  string
		expected bool
	}{
		{
			name:     "存在的目录",
			dirname:  tempDir,
			expected: true,
		},
		{
			name:     "不存在的目录",
			dirname:  "non_existent_directory",
			expected: false,
		},
		{
			name:     "文件而非目录",
			dirname:  tempFile.Name(),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DirExists(tt.dirname)
			if result != tt.expected {
				t.Errorf("DirExists(%s) = %v, 期望 %v", tt.dirname, result, tt.expected)
			}
		})
	}
}

func TestSliceHasStr(t *testing.T) {
	tests := []struct {
		name     string
		slice    []string
		str      string
		expected bool
	}{
		{
			name:     "存在的字符串",
			slice:    []string{"apple", "banana", "orange"},
			str:      "banana",
			expected: true,
		},
		{
			name:     "不存在的字符串",
			slice:    []string{"apple", "banana", "orange"},
			str:      "grape",
			expected: false,
		},
		{
			name:     "空切片",
			slice:    []string{},
			str:      "apple",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SliceHasStr(tt.slice, tt.str)
			if result != tt.expected {
				t.Errorf("SliceHasStr(%v, %s) = %v, 期望 %v", tt.slice, tt.str, result, tt.expected)
			}
		})
	}
}

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		expected string
	}{
		{
			name:     "包含不安全字符的文件名",
			filename: "file/with:invalid*chars?.txt",
			expected: "file_with_invalid_chars_.txt",
		},
		{
			name:     "带前导和尾随空格的文件名",
			filename: "  filename  ",
			expected: "filename",
		},
		{
			name:     "空文件名",
			filename: "",
			expected: "unnamed",
		},
		{
			name:     "多种不安全字符组合",
			filename: "my/file:with*lot\"of?invalid<>chars|.txt",
			expected: "my_file_with_lot_of_invalid__chars_.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeFilename(tt.filename)
			if result != tt.expected {
				t.Errorf("SanitizeFilename(%s) = %s, 期望 %s", tt.filename, result, tt.expected)
			}
		})
	}
}
