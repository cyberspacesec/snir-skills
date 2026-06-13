package report

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/cyberspacesec/snir-skills/pkg/database"
	"github.com/cyberspacesec/snir-skills/pkg/islazy"
	"github.com/cyberspacesec/snir-skills/pkg/log"
	"github.com/cyberspacesec/snir-skills/pkg/models"
	"github.com/cyberspacesec/snir-skills/pkg/runner"
)

// ConvertOptions 包含报告转换选项
type ConvertOptions struct {
	FromFile string // 输入文件
	ToFile   string // 输出文件
}

// 支持的文件扩展名
var supportedExtensions = []string{".sqlite3", ".db", ".jsonl", ".csv"}

// Convert 转换报告格式
func Convert(options ConvertOptions) error {
	// 验证输入文件
	if options.FromFile == "" {
		return fmt.Errorf("输入文件不能为空")
	}

	// 验证输出文件
	if options.ToFile == "" {
		return fmt.Errorf("输出文件不能为空")
	}

	// 检查输入文件存在
	if !islazy.FileExists(options.FromFile) {
		return fmt.Errorf("输入文件不存在: %s", options.FromFile)
	}

	// 获取文件扩展名
	fromExt := strings.ToLower(filepath.Ext(options.FromFile))
	toExt := strings.ToLower(filepath.Ext(options.ToFile))

	// 验证文件扩展名
	if !isValidExtension(fromExt) {
		return fmt.Errorf("不支持的输入文件格式: %s", fromExt)
	}
	if !isValidExtension(toExt) {
		return fmt.Errorf("不支持的输出文件格式: %s", toExt)
	}

	// 读取结果
	results, err := readResults(options.FromFile, fromExt)
	if err != nil {
		return fmt.Errorf("读取输入文件失败: %v", err)
	}

	// 写入结果
	if err := writeResults(options.ToFile, toExt, results); err != nil {
		return fmt.Errorf("写入输出文件失败: %v", err)
	}

	log.Info("报告转换完成", "from", options.FromFile, "to", options.ToFile)
	return nil
}

// isValidExtension 检查文件扩展名是否有效
func isValidExtension(ext string) bool {
	for _, validExt := range supportedExtensions {
		if ext == validExt {
			return true
		}
	}
	return false
}

// readResults 从文件读取结果
func readResults(filePath, ext string) ([]*models.Result, error) {
	switch ext {
	case ".sqlite3", ".db":
		// 从SQLite数据库读取
		db, err := database.NewDB(database.Options{Path: filePath})
		if err != nil {
			return nil, fmt.Errorf("连接数据库失败: %v", err)
		}
		defer db.Close()
		return db.ExportResults()

	case ".jsonl":
		// 从JSONL文件读取
		return ReadJSONLResults(filePath)

	case ".csv":
		// 从CSV文件读取
		return readCSVResults(filePath)

	default:
		return nil, fmt.Errorf("不支持的文件格式: %s", ext)
	}
}

// writeResults 将结果写入文件
func writeResults(filePath, ext string, results []*models.Result) error {
	// 确保目录存在
	dir := filepath.Dir(filePath)
	if _, err := islazy.CreateDir(dir); err != nil {
		return fmt.Errorf("创建输出目录失败: %v", err)
	}

	switch ext {
	case ".sqlite3", ".db":
		// 写入SQLite数据库
		db, err := database.NewDB(database.Options{Path: filePath})
		if err != nil {
			return fmt.Errorf("创建数据库失败: %v", err)
		}
		defer db.Close()
		return db.SaveResults(results)

	case ".jsonl":
		// 写入JSONL文件
		writer, err := runner.NewJSONLWriter(filePath)
		if err != nil {
			return fmt.Errorf("创建JSONL写入器失败: %v", err)
		}
		defer writer.Close()

		for _, result := range results {
			if err := writer.Write(result); err != nil {
				return fmt.Errorf("写入JSONL失败: %v", err)
			}
		}
		return nil

	case ".csv":
		// 写入CSV文件
		writer, err := runner.NewCSVWriter(filePath)
		if err != nil {
			return fmt.Errorf("创建CSV写入器失败: %v", err)
		}
		defer writer.Close()

		for _, result := range results {
			if err := writer.Write(result); err != nil {
				return fmt.Errorf("写入CSV失败: %v", err)
			}
		}
		return nil

	default:
		return fmt.Errorf("不支持的文件格式: %s", ext)
	}
}

// readCSVResults 从CSV文件读取结果
func readCSVResults(filePath string) ([]*models.Result, error) {
	// CSV读取较为复杂，需要解析CSV格式并转换为Result对象
	// 暂时返回未实现
	return nil, fmt.Errorf("从CSV读取结果暂未实现")
}
