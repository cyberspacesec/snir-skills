package islazy

import (
	"os"
	"path/filepath"
	"strings"
)

// DefaultScreenshotDir 是默认的截图保存目录
const DefaultScreenshotDir = "./screenshots"

// CreateDir 创建一个目录（如果它不存在）
// 如果路径为空，则使用默认的截图目录
// 返回创建的目录的绝对路径和可能的错误
func CreateDir(path string) (string, error) {
	// 当提供空路径时使用默认路径
	if path == "" {
		path = DefaultScreenshotDir
	}

	// 获取绝对路径以确保一致性
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}

	// 检查目录是否存在，如果不存在则创建
	if !DirExists(absPath) {
		err = os.MkdirAll(absPath, 0755)
		if err != nil {
			return "", err
		}
	}

	return absPath, nil
}

// FileExists 检查文件是否存在且不是目录
// 返回布尔值：true 表示文件存在，false 表示不存在或是目录
func FileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return info != nil && !info.IsDir()
}

// DirExists 检查目录是否存在
// 返回布尔值：true 表示目录存在，false 表示不存在或不是目录
func DirExists(dirname string) bool {
	info, err := os.Stat(dirname)
	if os.IsNotExist(err) {
		return false
	}
	return info != nil && info.IsDir()
}

// SliceHasStr 检查字符串切片是否包含特定字符串
// 返回布尔值：true 表示找到字符串，false 表示未找到
func SliceHasStr(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}

// UnsafeFilenameChars 定义文件名中不安全的字符
var UnsafeFilenameChars = []string{"\\", "/", ":", "*", "?", "\"", "<", ">", "|", "%"}

// SanitizeFilename 清理字符串以用作文件名
// 替换不安全的字符，处理空格，并确保文件名不为空
// 返回安全的文件名字符串
func SanitizeFilename(filename string) string {
	// 替换不安全的字符
	result := filename

	for _, char := range UnsafeFilenameChars {
		result = strings.ReplaceAll(result, char, "_")
	}

	// 移除前导和尾随空格
	result = strings.TrimSpace(result)

	// 确保文件名不为空
	if result == "" {
		result = "unnamed"
	}

	return result
}
